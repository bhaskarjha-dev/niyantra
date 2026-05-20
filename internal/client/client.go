package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/gemini"
)

// Sentinel errors for client operations.
var (
	ErrProcessNotFound  = errors.New("antigravity: language server process not found")
	ErrPortNotFound     = errors.New("antigravity: no listening port found")
	ErrConnectionFailed = errors.New("antigravity: connection failed")
	ErrInvalidResponse  = errors.New("antigravity: invalid response")
	ErrNotAuthenticated = errors.New("antigravity: not authenticated")
)

// lsEndpoint is the Connect RPC service path for quota retrieval.
const lsEndpoint = "/exa.language_server_pb.LanguageServerService/GetUserStatus"

// processInfo holds auto-detected process metadata.
type processInfo struct {
	PID                 int
	CSRFToken           string
	ExtensionServerPort int
	CommandLine         string
}

// connection holds a verified language server endpoint.
type connection struct {
	BaseURL   string
	CSRFToken string
	Port      int
	Protocol  string
}

// Client communicates with the local Antigravity language server.
type Client struct {
	transport *http.Client
	conns     []*connection
	connTime  time.Time // when the connections were last verified
	logger    *slog.Logger
}

// connTTL is the maximum age of a cached connection before re-verification.
// Prevents stale connections after language server restarts.
const connTTL = 10 * time.Minute

// New returns a Client ready to detect the language server.
func New(logger *slog.Logger) *Client {
	return &Client{
		transport: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxConnsPerHost:       2,
				ResponseHeaderTimeout: 12 * time.Second,
				IdleConnTimeout:       45 * time.Second,
				TLSHandshakeTimeout:   6 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // language server uses self-signed certs
				},
			},
		},
		logger: logger,
	}
}

// Detect locates and verifies connections to all running language servers.
// Subsequent calls return immediately if connections are cached.
func (c *Client) Detect(ctx context.Context) error {
	if len(c.conns) > 0 && time.Since(c.connTime) < connTTL {
		return nil
	}
	if len(c.conns) > 0 {
		c.logger.Debug("connections TTL expired, re-detecting language servers")
		c.conns = nil
	}

	c.logger.Debug("searching for Antigravity language servers")

	procs, err := c.detectProcesses(ctx)
	if err != nil {
		return err
	}

	c.logger.Debug("language server processes located", "count", len(procs))

	var verified []*connection
	for _, proc := range procs {
		ports, err := c.discoverPorts(ctx, proc.PID)
		if err != nil || len(ports) == 0 {
			continue
		}

		c.logger.Debug("candidate ports discovered for PID", "pid", proc.PID, "count", len(ports))

		conn, err := c.verifyEndpoint(ctx, ports, proc.CSRFToken)
		if err != nil {
			continue
		}
		verified = append(verified, conn)
	}

	if len(verified) == 0 {
		return ErrPortNotFound
	}

	c.conns = verified
	c.connTime = time.Now()
	c.logger.Info("language server connections established", "count", len(verified))

	return nil
}

// FetchQuotas retrieves the current quota status from the language servers.
//
// The LS maintains its own cache that refreshes on a ~60-120s timer.
// Data is always for the CORRECT current account but may be slightly stale.
// Users can fine-tune quota values via the Quick Adjust feature after snapping.
func (c *Client) FetchQuotas(ctx context.Context) ([]*UserStatusResponse, error) {
	err := c.Detect(ctx)
	if err != nil {
		c.logger.Debug("no active local language servers found, trying CLI PA API fallback", "error", err)
		return c.fetchQuotasFallback(ctx)
	}

	var results []*UserStatusResponse
	seenEmails := make(map[string]bool)
	var activeConns []*connection

	for _, conn := range c.conns {
		url := conn.BaseURL + lsEndpoint
		payload := `{"metadata":{"ideName":"antigravity","extensionName":"antigravity","locale":"en"}}`

		fetchCtx, cancel := context.WithTimeout(ctx, 12*time.Second)

		req, err := http.NewRequestWithContext(fetchCtx, http.MethodPost, url, strings.NewReader(payload))
		if err != nil {
			cancel()
			c.logger.Warn("antigravity: build request failed", "port", conn.Port, "error", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		if conn.CSRFToken != "" {
			req.Header.Set("X-Codeium-Csrf-Token", conn.CSRFToken)
		}

		resp, err := c.transport.Do(req)
		cancel()
		if err != nil {
			c.logger.Warn("antigravity: connection failed to port", "port", conn.Port, "error", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			c.logger.Warn("antigravity: server returned non-OK status", "port", conn.Port, "status", resp.StatusCode)
			continue
		}

		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16)) // cap at 64 KiB
		resp.Body.Close()
		if err != nil {
			c.logger.Warn("antigravity: read body failed", "port", conn.Port, "error", err)
			continue
		}
		if len(raw) == 0 {
			c.logger.Warn("antigravity: empty body", "port", conn.Port)
			continue
		}

		var out UserStatusResponse
		if err := json.Unmarshal(raw, &out); err != nil {
			c.logger.Warn("antigravity: parse body failed", "port", conn.Port, "error", err)
			continue
		}
		out.OriginalRawJSON = string(raw)

		if out.UserStatus == nil {
			c.logger.Warn("antigravity: not authenticated or empty userStatus", "port", conn.Port)
			continue
		}

		email := strings.ToLower(out.UserStatus.Email)
		if seenEmails[email] {
			c.logger.Debug("antigravity: skipping duplicate account", "email", email, "port", conn.Port)
			activeConns = append(activeConns, conn)
			continue
		}

		seenEmails[email] = true
		results = append(results, &out)
		activeConns = append(activeConns, conn)
	}

	c.conns = activeConns

	if len(results) == 0 {
		c.logger.Debug("all local language server connections failed, trying CLI PA API fallback")
		return c.fetchQuotasFallback(ctx)
	}

	return results, nil
}

// fetchQuotasFallback uses Gemini CLI credentials to query Google PA API directly when no server is active.
func (c *Client) fetchQuotasFallback(ctx context.Context) ([]*UserStatusResponse, error) {
	c.logger.Debug("detecting local Gemini CLI credentials")
	creds, err := gemini.DetectCredentials(c.logger)
	if err != nil {
		return nil, fmt.Errorf("fallback failed: %w", err)
	}

	clientID, clientSecret := gemini.ExtractOAuthClientCreds(c.logger)
	geminiClient := gemini.NewClient(creds, c.logger, clientID, clientSecret)

	snap, err := geminiClient.FetchSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("fallback API poll failed: %w", err)
	}

	if snap == nil || snap.Email == "" {
		return nil, fmt.Errorf("fallback returned empty snapshot or email")
	}

	usr := convertGeminiSnapshotToUserStatusResponse(snap)
	return []*UserStatusResponse{usr}, nil
}

// convertGeminiSnapshotToUserStatusResponse converts a gemini.Snapshot to a client.UserStatusResponse.
func convertGeminiSnapshotToUserStatusResponse(gSnap *gemini.Snapshot) *UserStatusResponse {
	userStatus := &UserStatus{
		Email: gSnap.Email,
		Name:  strings.Split(gSnap.Email, "@")[0],
		PlanStatus: &PlanStatus{
			PlanInfo: &PlanInfo{
				PlanName:             gSnap.Tier,
				MonthlyPromptCredits: 0,
			},
			AvailablePromptCredits: 0,
		},
		CascadeModelConfigData: &CascadeModelConfigData{
			ClientModelConfigs: []ModelConfig{},
		},
		UserTier: &UserTier{
			ID:               gSnap.Tier,
			Name:             gSnap.Tier,
			AvailableCredits: []rawCredit{},
		},
	}

	for _, m := range gSnap.Models {
		userStatus.CascadeModelConfigData.ClientModelConfigs = append(
			userStatus.CascadeModelConfigData.ClientModelConfigs,
			ModelConfig{
				Label: m.ModelID,
				ModelOrAlias: &ModelOrAlias{
					Model: m.ModelID,
				},
				QuotaInfo: &QuotaInfo{
					RemainingFraction: m.RemainingFraction,
					ResetTime:         m.ResetTime,
				},
			},
		)
	}

	rawBytes, _ := json.Marshal(gSnap)

	return &UserStatusResponse{
		UserStatus:      userStatus,
		OriginalRawJSON: string(rawBytes),
	}
}

// Reset forces the next call to Detect to re-discover the language servers.
func (c *Client) Reset() {
	c.conns = nil
}

// IsConnected reports whether any cached connection exists.
func (c *Client) IsConnected() bool {
	return len(c.conns) > 0
}

// ConnInfo exposes connection details for diagnostics.
type ConnInfo struct {
	BaseURL   string
	CSRFToken string
	Port      int
	Protocol  string
}

// ConnectionInfo returns cached connection details (first connection for backward compatibility), or nil.
func (c *Client) ConnectionInfo() *ConnInfo {
	if len(c.conns) == 0 {
		return nil
	}
	return &ConnInfo{
		BaseURL:   c.conns[0].BaseURL,
		CSRFToken: c.conns[0].CSRFToken,
		Port:      c.conns[0].Port,
		Protocol:  c.conns[0].Protocol,
	}
}
