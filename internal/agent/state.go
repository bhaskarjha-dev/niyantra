package agent

import (
	"log/slog"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/notify"
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
	"github.com/bhaskarjha-com/niyantra/internal/store"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// NewPollingAgent creates a new auto-capture agent.
func NewPollingAgent(c *client.Client, s *store.Store, t *tracker.Tracker, interval time.Duration, logger *slog.Logger) *PollingAgent {
	return &PollingAgent{
		client:    c,
		store:     s,
		tracker:   t,
		interval:  interval,
		logger:    logger,
		maxFails:  3,
		authFails: make(map[string]int),
	}
}

// SetPollingCheck sets the function called before each poll to check if polling is enabled.
func (a *PollingAgent) SetPollingCheck(fn func() bool) {
	a.pollingCheck = fn
}

// SetNotifier sets the notification engine for quota alerts.
func (a *PollingAgent) SetNotifier(n *notify.Engine) {
	a.notifier = n
}

// SetSessionManagers initializes session detection for all providers.
func (a *PollingAgent) SetSessionManagers(idleTimeout time.Duration) {
	a.antigravitySM = tracker.NewSessionManager(a.store, "antigravity", idleTimeout, a.logger)
	a.codexSM = tracker.NewSessionManager(a.store, "codex", idleTimeout, a.logger)
	a.claudeSM = tracker.NewSessionManager(a.store, "claude", idleTimeout, a.logger)
}

// LastPollTime returns the time of the last poll attempt.
func (a *PollingAgent) LastPollTime() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastPollTime
}

// LastPollOK returns whether the last poll was successful.
func (a *PollingAgent) LastPollOK() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastPollOK
}

// SetPlugins sets the list of discovered plugins for the agent to poll.
func (a *PollingAgent) SetPlugins(plugins []*plugin.Plugin) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(plugins) == 0 {
		a.plugins = nil
		return
	}

	copied := make([]*plugin.Plugin, len(plugins))
	copy(copied, plugins)
	a.plugins = copied
}

// Plugins returns a snapshot of the configured plugin list.
func (a *PollingAgent) Plugins() []*plugin.Plugin {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.plugins) == 0 {
		return nil
	}

	copied := make([]*plugin.Plugin, len(a.plugins))
	copy(copied, a.plugins)
	return copied
}
