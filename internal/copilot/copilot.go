// Package copilot provides GitHub Copilot credential detection and usage API polling.
//
// Based on deep analysis of CodexBar (steipete/CodexBar) and the VS Code Copilot extension:
//
// Single endpoint is polled:
//
//	GET https://api.github.com/copilot_internal/user
//
// Auth: GitHub PAT with `read:user` scope, passed as `Authorization: token <PAT>`.
// Headers mimic the VS Code Copilot extension to get a valid response.
//
// The response contains `quotaSnapshots.premiumInteractions` with a `percentRemaining`
// field indicating credit/quota usage, and `copilotPlan` for plan identification.
package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ── Errors ──────────────────────────────────────────────────────────

var (
	ErrNoPAT           = errors.New("copilot: no personal access token configured")
	ErrUnauthorized    = errors.New("copilot: unauthorized (401) — check PAT scope (needs read:user)")
	ErrForbidden       = errors.New("copilot: forbidden (403) — Copilot not enabled for this account")
	ErrServerError     = errors.New("copilot: GitHub API server error")
	ErrNetworkError    = errors.New("copilot: network error")
	ErrInvalidResponse = errors.New("copilot: invalid response")
	ErrNoCopilot       = errors.New("copilot: no Copilot subscription found")
)

// ── API Response Types ──────────────────────────────────────────────

// UsageResponse is the top-level response from GET /copilot_internal/user.
type UsageResponse struct {
	CopilotPlan       string         `json:"copilot_plan"`
	AccessTypeSKU     string         `json:"access_type_sku"`
	QuotaSnapshots    QuotaSnapshots `json:"quota_snapshots"`
	LimitedUserQuotas *LimitedQuotas `json:"limited_user_quotas"`
	MonthlyQuotas     *LimitedQuotas `json:"monthly_quotas"`
}

// LimitedQuotas represents individual chat/completions limits for free_limited_copilot SKU.
type LimitedQuotas struct {
	Chat        int64 `json:"chat"`
	Completions int64 `json:"completions"`
}

// QuotaSnapshots contains the quota windows from the Copilot API.
type QuotaSnapshots struct {
	PremiumInteractions *QuotaWindow `json:"premium_interactions"`
	Chat                *QuotaWindow `json:"chat"`
}

// QuotaWindow is a single quota window (e.g. premium interactions or chat).
type QuotaWindow struct {
	PercentRemaining *float64 `json:"percent_remaining"`
	PercentUsed      *float64 `json:"percent_used"`
	IsUnlimited      bool     `json:"is_unlimited"`
	OverQuota        bool     `json:"over_quota"`
}

// UsedPercent returns the used percentage (0-100) for this window.
func (w *QuotaWindow) UsedPercent() float64 {
	if w == nil {
		return 0
	}
	if w.IsUnlimited {
		return 0
	}
	if w.PercentUsed != nil {
		return *w.PercentUsed
	}
	if w.PercentRemaining != nil {
		return 100 - *w.PercentRemaining
	}
	return 0
}

// HasData returns true if this window contains valid percentage data.
func (w *QuotaWindow) HasData() bool {
	if w == nil {
		return false
	}
	return w.PercentRemaining != nil || w.PercentUsed != nil || w.IsUnlimited
}

// Identity holds the GitHub user identity from /user endpoint.
type Identity struct {
	Login string `json:"login"`
	Email string `json:"email"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
}

// Snapshot is the merged result from the Copilot usage API.
type Snapshot struct {
	Plan       string  // copilot plan: pro, pro+, free, business, enterprise
	PremiumPct float64 // premium interactions: % used (0-100)
	ChatPct    float64 // chat: % used (0-100)
	HasPremium bool    // whether premium data is available
	HasChat    bool    // whether chat data is available
	Username   string  // GitHub username
	Email      string  // GitHub email (may be empty for private emails)
}

// ── Endpoints ───────────────────────────────────────────────────────

var (
	usageURL    = "https://api.github.com/copilot_internal/user"
	identityURL = "https://api.github.com/user"
	timeout     = 15 * time.Second
)

// ── Client ──────────────────────────────────────────────────────────

// Client polls GitHub's Copilot usage API.
type Client struct {
	pat    string
	logger *slog.Logger
	http   *http.Client
}

// NewClient creates a Copilot API client.
func NewClient(pat string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		pat:    pat,
		logger: logger,
		http:   &http.Client{Timeout: timeout},
	}
}

// FetchSnapshot polls the Copilot usage API and returns a Snapshot.
func (c *Client) FetchSnapshot(ctx context.Context) (*Snapshot, error) {
	if c.pat == "" {
		return nil, ErrNoPAT
	}

	// 1) Fetch usage data
	usage, err := c.fetchUsage(ctx)
	if err != nil {
		return nil, err
	}

	plan := normalizePlan(usage.CopilotPlan)
	if usage.AccessTypeSKU == "free_limited_copilot" {
		plan = "Free"
	}

	snap := &Snapshot{
		Plan: plan,
	}

	// Parse premium interactions
	if usage.QuotaSnapshots.PremiumInteractions != nil && usage.QuotaSnapshots.PremiumInteractions.HasData() {
		snap.HasPremium = true
		snap.PremiumPct = usage.QuotaSnapshots.PremiumInteractions.UsedPercent()
	}

	// Parse chat
	if usage.QuotaSnapshots.Chat != nil && usage.QuotaSnapshots.Chat.HasData() {
		snap.HasChat = true
		snap.ChatPct = usage.QuotaSnapshots.Chat.UsedPercent()
	}

	// Fallback to limited user quotas if standard ones are missing
	if !snap.HasPremium && !snap.HasChat && usage.LimitedUserQuotas != nil && usage.MonthlyQuotas != nil {
		if usage.MonthlyQuotas.Completions > 0 {
			snap.HasPremium = true
			usedCompletions := usage.MonthlyQuotas.Completions - usage.LimitedUserQuotas.Completions
			if usedCompletions < 0 {
				usedCompletions = 0
			}
			snap.PremiumPct = (float64(usedCompletions) / float64(usage.MonthlyQuotas.Completions)) * 100.0
		}
		if usage.MonthlyQuotas.Chat > 0 {
			snap.HasChat = true
			usedChat := usage.MonthlyQuotas.Chat - usage.LimitedUserQuotas.Chat
			if usedChat < 0 {
				usedChat = 0
			}
			snap.ChatPct = (float64(usedChat) / float64(usage.MonthlyQuotas.Chat)) * 100.0
		}
	}

	if !snap.HasPremium && !snap.HasChat {
		return nil, fmt.Errorf("%w: no quota data in response", ErrNoCopilot)
	}

	// 2) Fetch identity (best-effort — don't fail if this errors)
	identity, err := c.fetchIdentity(ctx)
	if err == nil && identity != nil {
		snap.Username = identity.Login
		snap.Email = identity.Email
	} else {
		c.logger.Debug("Copilot identity fetch failed (non-fatal)", "error", err)
	}

	c.logger.Info("Copilot snapshot fetched",
		"plan", snap.Plan,
		"premiumPct", snap.PremiumPct, "hasPremium", snap.HasPremium,
		"chatPct", snap.ChatPct, "hasChat", snap.HasChat,
		"username", snap.Username)

	return snap, nil
}

// fetchUsage calls GET /copilot_internal/user with PAT auth.
func (c *Client) fetchUsage(ctx context.Context) (*UsageResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageURL, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	var result UsageResponse
	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// fetchIdentity calls GET /user with PAT auth to get username/email.
func (c *Client) fetchIdentity(ctx context.Context) (*Identity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, identityURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Niyantra/1.0")

	var result Identity
	if err := c.doJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// addHeaders sets the required headers mimicking the VS Code Copilot extension.
// These headers are essential — the API returns 403 without them.
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "token "+c.pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Editor-Version", "vscode/1.96.2")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.26.7")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.26.7")
	req.Header.Set("X-Github-Api-Version", "2025-04-01")
}

func (c *Client) doJSON(req *http.Request, out interface{}) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	default:
		if resp.StatusCode >= 500 {
			return fmt.Errorf("%w: HTTP %d", ErrServerError, resp.StatusCode)
		}
		return fmt.Errorf("%w: HTTP %d", ErrInvalidResponse, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: read: %v", ErrNetworkError, err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: JSON: %v", ErrInvalidResponse, err)
	}
	return nil
}

// normalizePlan converts the raw copilot_plan string to a display label.
func normalizePlan(raw string) string {
	switch raw {
	case "individual", "pro":
		return "Pro"
	case "individual_pro_plus", "pro+", "pro_plus":
		return "Pro+"
	case "business":
		return "Business"
	case "enterprise":
		return "Enterprise"
	case "free":
		return "Free"
	case "":
		return "unknown"
	default:
		return raw
	}
}

// hostConfig is a helper struct for parsing hosts.json.
type hostConfig struct {
	User       string `json:"user"`
	OAuthToken string `json:"oauth_token"`
}

// copilotDataDir returns the platform-specific GitHub Copilot config root.
func copilotDataDir() string {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, _ := os.UserHomeDir()
			appdata = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appdata, "github-copilot")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "github-copilot")
	default:
		home, _ := os.UserHomeDir()
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "github-copilot")
	}
}

// DetectCredentials discovers the GitHub Copilot token from hosts.json or GitHub CLI.
func DetectCredentials(logger *slog.Logger) (string, string, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Try GitHub CLI first
	if token, err := detectGHCLIToken(logger); err == nil && token != "" {
		return token, "github_cli", nil
	}

	dataDir := copilotDataDir()
	hostsPath := filepath.Join(dataDir, "hosts.json")

	// Try the primary platform-specific path
	token, err := extractTokenFromFile(hostsPath, logger)
	if err == nil && token != "" {
		return token, "hosts.json", nil
	}

	// Fallback path: ~/.config/github-copilot/hosts.json (common for unix/macOS setups)
	home, _ := os.UserHomeDir()
	if home != "" {
		fallbackPath := filepath.Join(home, ".config", "github-copilot", "hosts.json")
		if fallbackPath != hostsPath {
			token, err := extractTokenFromFile(fallbackPath, logger)
			if err == nil && token != "" {
				return token, "hosts.json", nil
			}
		}
	}

	return "", "", fmt.Errorf("copilot: hosts.json not found and gh CLI detection failed")
}

// detectGHCLIToken detects the token by executing `gh auth token`.
func detectGHCLIToken(logger *slog.Logger) (string, error) {
	path, err := exec.LookPath("gh")
	if err != nil {
		logger.Debug("Copilot: gh CLI not found in PATH")
		return "", err
	}

	cmd := exec.Command(path, "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		logger.Debug("Copilot: gh auth token command failed", "error", err)
		return "", err
	}

	token := strings.TrimSpace(string(out))
	if token == "" {
		logger.Debug("Copilot: gh auth token returned empty token")
		return "", fmt.Errorf("empty token")
	}

	logger.Debug("Copilot: token auto-detected via gh CLI")
	return token, nil
}

// extractTokenFromFile reads a hosts.json file and extracts the oauth_token.
func extractTokenFromFile(path string, logger *slog.Logger) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Debug("Copilot: failed to read hosts.json", "path", path, "error", err)
		return "", err
	}

	var config map[string]hostConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Debug("Copilot: failed to unmarshal hosts.json", "path", path, "error", err)
		return "", err
	}

	// Try github.com first
	if cfg, ok := config["github.com"]; ok && cfg.OAuthToken != "" {
		logger.Debug("Copilot: token auto-detected for github.com", "path", path)
		return cfg.OAuthToken, nil
	}

	// Fallback to any host with an oauth_token
	for host, cfg := range config {
		if cfg.OAuthToken != "" {
			logger.Debug("Copilot: token auto-detected", "host", host, "path", path)
			return cfg.OAuthToken, nil
		}
	}

	return "", fmt.Errorf("no oauth_token found in %s", path)
}
