package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
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
	CSRFToken           string // --csrf_token: for the main HTTPS server
	ExtCSRFToken        string // --extension_server_csrf_token: for the extension server
	ExtensionServerPort int
	HTTPSServerPort     int // --https_server_port: the port serving Connect RPC endpoints
	LSPPort             int // --lsp_port: Language Server Protocol port (not HTTP)
	CommandLine         string
}

// connection holds a verified language server endpoint.
type connection struct {
	BaseURL       string
	CSRFToken     string
	Port          int
	Protocol      string
	ToolSignature string // groups connections from the same IDE instance
	IsHub         bool   // true = hub process (manages auth, current account)
}

// Client communicates with the local Antigravity language server.
type Client struct {
	transport *http.Client
	conns     []*connection
	connTime  time.Time // when the connections were last verified
	logger    *slog.Logger
}

const connTTL = 30 * time.Second

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

	for _, p := range procs {
		c.logger.Debug("detected language server process",
			"pid", p.PID,
			"httpsPort", p.HTTPSServerPort,
			"extPort", p.ExtensionServerPort,
			"lspPort", p.LSPPort,
			"hasExtCSRF", p.ExtCSRFToken != "",
			"signature", getToolSignature(p.CommandLine),
			"hasWorkspaceId", strings.Contains(strings.ToLower(p.CommandLine), "workspace_id"),
		)
	}

	// NOTE: We intentionally do NOT deduplicate processes.
	// When the user switches accounts in the IDE, the hub process gets the new
	// account but the workspace process keeps the old one. If we deduplicate
	// (preferring workspace), we lose the hub's updated account data.
	// Instead, we query ALL processes and let FetchQuotas deduplicate by email.

	c.logger.Debug("language server processes to probe", "count", len(procs))

	var verified []*connection
	for _, proc := range procs {
		isHub := !strings.Contains(strings.ToLower(proc.CommandLine), "workspace_id")
		toolSig := getToolSignature(proc.CommandLine)

		c.logger.Debug("probing process",
			"pid", proc.PID,
			"httpsPort", proc.HTTPSServerPort,
			"extPort", proc.ExtensionServerPort,
			"isHub", isHub,
			"toolSig", toolSig,
		)

		var conn *connection

		// Strategy 1: Try the known HTTPS server port with the main CSRF token.
		if proc.HTTPSServerPort > 0 {
			vc, err := c.verifyEndpoint(ctx, []int{proc.HTTPSServerPort}, proc.CSRFToken)
			if err == nil {
				conn = vc
			}
		}

		// Strategy 2: Try the extension server port with its own CSRF token.
		if conn == nil && proc.ExtensionServerPort > 0 {
			csrfForExt := proc.ExtCSRFToken
			if csrfForExt == "" {
				csrfForExt = proc.CSRFToken
			}
			ec, err := c.verifyEndpoint(ctx, []int{proc.ExtensionServerPort}, csrfForExt)
			if err == nil {
				conn = ec
			}
		}

		// Strategy 3: Discover ports via netstat and try remaining ones.
		if conn == nil {
			exclude := make(map[int]bool)
			if proc.HTTPSServerPort > 0 {
				exclude[proc.HTTPSServerPort] = true
			}
			if proc.ExtensionServerPort > 0 {
				exclude[proc.ExtensionServerPort] = true
			}
			if proc.LSPPort > 0 {
				exclude[proc.LSPPort] = true
			}

			allPorts, discErr := c.discoverPorts(ctx, proc.PID)
			if discErr == nil && len(allPorts) > 0 {
				var fallbackPorts []int
				for _, p := range allPorts {
					if !exclude[p] {
						fallbackPorts = append(fallbackPorts, p)
					}
				}
				if len(fallbackPorts) > 0 {
					c.logger.Debug("trying netstat-discovered ports", "pid", proc.PID, "ports", fallbackPorts)
					fc, err := c.verifyEndpoint(ctx, fallbackPorts, proc.CSRFToken)
					if err == nil {
						conn = fc
					}
				}
			}
		}

		if conn == nil {
			c.logger.Debug("no valid endpoint found for PID", "pid", proc.PID)
			continue
		}

		// Tag the connection with instance identity for FetchQuotas dedup.
		conn.ToolSignature = toolSig
		conn.IsHub = isHub
		c.logger.Debug("connection established", "pid", proc.PID, "port", conn.Port, "isHub", isHub, "toolSig", toolSig)
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
		return nil, err
	}

	// Sort connections: hub processes first, then workspace processes.
	// The hub always has the current account after an account switch;
	// the workspace may still be running with the old account's auth.
	sort.SliceStable(c.conns, func(i, j int) bool {
		if c.conns[i].IsHub != c.conns[j].IsHub {
			return c.conns[i].IsHub // hubs before workspaces
		}
		return false
	})

	var results []*UserStatusResponse
	seenEmails := make(map[string]bool)
	seenInstances := make(map[string]bool) // track which tool instances already returned data
	var activeConns []*connection

	for _, conn := range c.conns {
		// If we already got a result from this tool instance (e.g., the hub),
		// skip the workspace process — its account data may be stale.
		if conn.ToolSignature != "" && seenInstances[conn.ToolSignature] {
			c.logger.Debug("antigravity: skipping stale instance process",
				"port", conn.Port, "isHub", conn.IsHub, "toolSig", conn.ToolSignature)
			continue
		}

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

		c.logger.Debug("antigravity: fetching quota", "port", conn.Port, "protocol", conn.Protocol, "isHub", conn.IsHub)

		resp, err := c.transport.Do(req)
		if err != nil {
			cancel()
			c.logger.Warn("antigravity: connection failed to port", "port", conn.Port, "error", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			c.logger.Warn("antigravity: server returned non-OK status", "port", conn.Port, "status", resp.StatusCode)
			continue
		}

		// Read the full body BEFORE canceling the context.
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16)) // cap at 64 KiB
		resp.Body.Close()
		cancel() // Safe to cancel now — body is fully read
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
		c.logger.Info("antigravity: quota fetched", "port", conn.Port, "email", email, "isHub", conn.IsHub)

		// Mark this tool instance as seen so workspace processes from the
		// same instance are skipped (they may have stale auth).
		if conn.ToolSignature != "" {
			seenInstances[conn.ToolSignature] = true
		}

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
		return nil, ErrPortNotFound
	}

	return results, nil
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
