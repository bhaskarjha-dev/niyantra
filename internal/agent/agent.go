// Package agent provides background polling for auto-capture.
package agent

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/codex"
	"github.com/bhaskarjha-com/niyantra/internal/copilot"
	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/cursor"
	"github.com/bhaskarjha-com/niyantra/internal/notify"
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// PollingAgent polls registered AI providers at a configurable interval.
type PollingAgent struct {
	client   *client.Client
	store    *store.Store
	tracker  *tracker.Tracker
	interval time.Duration
	logger   *slog.Logger

	// pollingCheck is called before each tick; return false to skip.
	pollingCheck func() bool

	notifier *notify.Engine

	// Session managers for usage detection
	antigravitySM *tracker.SessionManager
	codexSM       *tracker.SessionManager
	claudeSM      *tracker.SessionManager

	// Codex state
	codexClient    *codex.Client
	codexAuthFails int

	// Cursor state
	cursorAuthFails int

	// Copilot state
	copilotAuthFails int

	// F18: Plugin system
	plugins []*plugin.Plugin

	// Backoff state for consecutive failures
	mu           sync.Mutex
	failCount    int
	maxFails     int // pause after this many consecutive failures
	lastPollTime time.Time
	lastPollOK   bool

	// Unified backoff tracker
	authFails map[string]int
}



// Run starts the polling loop. Blocks until ctx is cancelled.
// The poll interval is re-read from the store on each iteration,
// so changes via Settings take effect on the next cycle without restart.
func (a *PollingAgent) Run(ctx context.Context) error {
	a.logger.Info("Auto-capture agent started", "interval", a.interval)
	defer func() {
		// Close any active sessions on shutdown
		if a.antigravitySM != nil {
			a.antigravitySM.Close()
		}
		if a.codexSM != nil {
			a.codexSM.Close()
		}
		if a.claudeSM != nil {
			a.claudeSM.Close()
		}
		a.logger.Info("Auto-capture agent stopped")
	}()

	// Poll immediately on start
	a.poll(ctx)

	for {
		// Re-read interval from store each iteration (F2: live reload)
		interval := a.store.GetConfigInt("poll_interval", 300)
		if interval < 30 {
			interval = 30
		}

		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-timer.C:
			a.poll(ctx)
		case <-ctx.Done():
			timer.Stop()
			return nil
		}
	}
}

// poll performs a single capture cycle across all providers concurrently.
func (a *PollingAgent) poll(ctx context.Context) {
	if a.pollingCheck != nil && !a.pollingCheck() {
		return
	}

	start := time.Now()
	var wg sync.WaitGroup
	// Semaphore: limit to 4 concurrent provider calls
	sem := make(chan struct{}, 4)

	providers := core.All()

	for _, p := range providers {
		if p.ID() == "plugin" {
			continue // skip placeholder
		}

		if !a.isProviderEnabled(p) {
			continue
		}

		wg.Add(1)
		go func(p core.Provider) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			providerStart := time.Now()

			// Check auth failure/consecutive backoff status
			a.mu.Lock()
			fails := a.authFails[p.ID()]
			if fails >= 3 {
				if p.ID() == "antigravity" {
					a.authFails["antigravity"]++
					if a.authFails["antigravity"] >= 6 {
						a.authFails["antigravity"] = 0
						a.mu.Unlock()
						a.logger.Info("Antigravity: retrying after backoff")
					} else {
						a.mu.Unlock()
						a.logger.Debug("Antigravity polling paused (backoff)", "failures", fails)
						return
					}
				} else {
					a.mu.Unlock()
					a.logger.Debug("Provider polling paused (auth failures)", "provider", p.ID(), "failures", fails)
					return
				}
			} else {
				a.mu.Unlock()
			}

			// Gather configured credentials
			creds, err := a.getCredentials(p)
			if err != nil {
				a.logger.Debug("Provider skipped (credentials check)", "provider", p.ID(), "error", err)
				return
			}

			// Proactive Codex token refresh
			if p.ID() == "codex" {
				cCreds, err := codex.DetectCredentials(a.logger)
				if err == nil && cCreds != nil {
					if cCreds.IsExpiringSoon(6*time.Hour) && cCreds.RefreshToken != "" {
						a.logger.Info("Codex token expiring soon, refreshing")
						newTokens, err := codex.RefreshToken(ctx, cCreds.RefreshToken)
						if err != nil {
							if errors.Is(err, codex.ErrRefreshTokenReused) {
								a.mu.Lock()
								a.authFails["codex"] = 3
								a.mu.Unlock()
								a.logger.Error("Codex refresh token reused — re-authenticate via 'codex auth'")
								return
							}
							a.logger.Warn("Codex token refresh failed", "error", err)
						} else {
							if err := codex.WriteCredentials(newTokens.AccessToken, newTokens.RefreshToken, newTokens.IDToken); err != nil {
								a.logger.Error("Failed to save refreshed Codex credentials", "error", err)
							} else {
								creds.AccessToken = newTokens.AccessToken
								a.logger.Info("Codex token refreshed")
							}
						}
					}
				}
			}

			// Fetch metrics from provider
			snap, err := p.Fetch(ctx, creds)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if p.ID() == "claude" && strings.Contains(err.Error(), "stale or missing") {
					a.logger.Debug("Claude Code statusline data is stale or missing, skipping")
					return
				}

				// Check for known auth error types
				isAuthError := false
				if errors.Is(err, codex.ErrUnauthorized) || errors.Is(err, codex.ErrForbidden) ||
					errors.Is(err, cursor.ErrUnauthorized) || errors.Is(err, cursor.ErrForbidden) ||
					errors.Is(err, copilot.ErrUnauthorized) || errors.Is(err, copilot.ErrForbidden) {
					isAuthError = true
				}

				if isAuthError || p.ID() == "antigravity" {
					a.mu.Lock()
					a.authFails[p.ID()]++
					fails := a.authFails[p.ID()]
					if p.ID() == "antigravity" {
						a.failCount = fails
						a.lastPollOK = false
						a.lastPollTime = time.Now().UTC()
					}
					a.mu.Unlock()
					a.logger.Warn("Provider auth/consecutive error", "provider", p.ID(), "error", err, "failures", fails)
					a.store.LogError("server", "snap_failed", p.ID(), map[string]interface{}{
						"error":  err.Error(),
						"method": "auto",
					})
				} else {
					a.logger.Warn("Provider poll failed", "provider", p.ID(), "error", err)
				}
				return
			}

			// Successful execution - reset failure track
			a.mu.Lock()
			a.authFails[p.ID()] = 0
			if p.ID() == "antigravity" {
				a.failCount = 0
				a.lastPollOK = true
				a.lastPollTime = time.Now().UTC()
			}
			a.mu.Unlock()

			// Link the local account ownership ID
			email := snap.Email
			if email == "" {
				email = snap.AccountID
			}
			if email == "" {
				email = p.Name() + " Account"
			}
			accountID, err := a.store.GetOrCreateAccount(email, snap.PlanTier, p.ID())
			if err != nil {
				a.logger.Warn("Failed to resolve local account ownership", "provider", p.ID(), "email", email, "error", err)
			} else {
				snap.AccountID = strconv.FormatInt(accountID, 10)
			}

			// Store the unified snapshot
			if err := a.store.SaveSnapshot(ctx, snap); err != nil {
				a.logger.Error("Failed to store snapshot", "provider", p.ID(), "error", err)
				return
			}

			a.store.LogInfo("server", "snap", snap.Email, map[string]interface{}{
				"provider": p.ID(), "plan": snap.PlanTier, "method": "auto",
			})

			// Auto-link subscriptions (generic)
			if accountID > 0 {
				a.autoLink(snap, accountID)
			}

			// Update last capture times
			sourceID := p.ID()
			if sourceID == "claude" {
				sourceID = "claude_code"
			}
			a.store.UpdateSourceCapture(sourceID)

			// Check alerts
			a.checkProviderNotifications(p, snap)

			// Feed active session logs
			a.reportProviderSession(p, snap)

			a.logger.Debug("Provider poll complete", "provider", p.ID(), "elapsed", time.Since(providerStart).Truncate(time.Millisecond))
		}(p)
	}

	wg.Wait()
	elapsed := time.Since(start)
	a.logger.Info("Poll cycle complete", "elapsed", elapsed.Truncate(time.Millisecond), "providers", len(providers))

	a.cleanupOldSnapshots()
}


