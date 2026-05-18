package notify

import (
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestCheckQuotaGuard(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 15)

	e.mu.Lock()
	if !e.guard["test-model"].IsZero() {
		t.Error("expected guard to be empty for test-model")
	}
	e.mu.Unlock()
}

func TestOnResetClearsGuard(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 10)

	e.mu.Lock()
	e.guard["claude_3.5_sonnet"] = time.Now()
	e.mu.Unlock()

	e.OnReset("claude_3.5_sonnet")

	e.mu.Lock()
	if !e.guard["claude_3.5_sonnet"].IsZero() {
		t.Error("expected guard to be cleared after OnReset")
	}
	e.mu.Unlock()
}

func TestOnNotifyCallback(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 20)

	var mu sync.Mutex
	var calledModel string
	var calledPct float64

	e.SetOnNotify(func(model string, remainingPct float64) {
		mu.Lock()
		calledModel = model
		calledPct = remainingPct
		mu.Unlock()
	})

	e.mu.Lock()
	cb := e.onNotify
	e.mu.Unlock()

	if cb == nil {
		t.Fatal("expected onNotify callback to be set")
	}

	cb("gpt-4o", 8.5)

	mu.Lock()
	defer mu.Unlock()
	if calledModel != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got '%s'", calledModel)
	}
	if calledPct != 8.5 {
		t.Errorf("expected pct 8.5, got %.1f", calledPct)
	}
}

func TestCheckQuotaSkipsWhenDisabled(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(false, 10)

	e.CheckQuota("test-model", 5.0)

	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.guard["test-model"].IsZero() {
		t.Error("expected guard to remain empty when notifications are disabled")
	}
}

func TestCheckQuotaSkipsAboveThreshold(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 10)

	e.CheckQuota("test-model", 50.0)

	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.guard["test-model"].IsZero() {
		t.Error("expected guard to remain empty when above threshold")
	}
}

func TestCheckClaudeQuotaConvertsUsedToRemaining(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 15)

	var gotModel string
	var gotRemaining float64
	e.SetOnNotify(func(model string, remainingPct float64) {
		gotModel = model
		gotRemaining = remainingPct
	})

	e.CheckClaudeQuota("five_hour", 95.0)

	e.mu.Lock()
	guardSet := !e.guard["claude_five_hour"].IsZero()
	e.mu.Unlock()

	if !guardSet {
		t.Fatal("expected guard to be set for claude_five_hour")
	}
	if gotModel != "Claude Code 5-hour" {
		t.Fatalf("expected callback model %q, got %q", "Claude Code 5-hour", gotModel)
	}
	if gotRemaining != 5.0 {
		t.Fatalf("expected remaining 5.0, got %.1f", gotRemaining)
	}
}

func TestCheckUsedQuotaUsesProvidedLabelAndGuardKey(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 20)

	var gotModel string
	var gotRemaining float64
	e.SetOnNotify(func(model string, remainingPct float64) {
		gotModel = model
		gotRemaining = remainingPct
	})

	e.CheckUsedQuota("cursor_usage", "Cursor", 90.0)

	e.mu.Lock()
	guardSet := !e.guard["cursor_usage"].IsZero()
	e.mu.Unlock()

	if !guardSet {
		t.Fatal("expected guard to be set for cursor_usage")
	}
	if gotModel != "Cursor" {
		t.Fatalf("expected callback model %q, got %q", "Cursor", gotModel)
	}
	if gotRemaining != 10.0 {
		t.Fatalf("expected remaining 10.0, got %.1f", gotRemaining)
	}
}

func TestConfigureUpdatesSettings(t *testing.T) {
	e := NewEngine(slog.Default())

	if e.Enabled() {
		t.Error("expected disabled by default")
	}
	if e.Threshold() != 10 {
		t.Errorf("expected default threshold 10, got %.0f", e.Threshold())
	}

	e.Configure(true, 25)
	if !e.Enabled() {
		t.Error("expected enabled after Configure")
	}
	if e.Threshold() != 25 {
		t.Errorf("expected threshold 25, got %.0f", e.Threshold())
	}

	e.Configure(true, -5)
	if e.Threshold() != 25 {
		t.Errorf("expected threshold to remain 25 for negative input, got %.0f", e.Threshold())
	}
}

func TestSendTestReturnsError(t *testing.T) {
	e := NewEngine(slog.Default())
	_ = e.SendTest()
}

func TestGuardTTLExpiry(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 10)
	e.guardTTL = 100 * time.Millisecond

	e.CheckQuota("codex_5h", 5.0)
	e.mu.Lock()
	guardSet := !e.guard["codex_5h"].IsZero()
	e.mu.Unlock()
	if !guardSet {
		t.Fatal("expected guard to be set after CheckQuota below threshold")
	}

	e.mu.Lock()
	guardBefore := e.guard["codex_5h"]
	e.mu.Unlock()

	e.CheckQuota("codex_5h", 5.0)

	e.mu.Lock()
	guardAfter := e.guard["codex_5h"]
	e.mu.Unlock()
	if !guardBefore.Equal(guardAfter) {
		t.Error("expected guard timestamp to remain unchanged (suppressed)")
	}

	time.Sleep(150 * time.Millisecond)

	e.CheckQuota("codex_5h", 5.0)
	e.mu.Lock()
	guardFinal := e.guard["codex_5h"]
	e.mu.Unlock()
	if guardFinal.Equal(guardBefore) {
		t.Error("expected guard timestamp to be updated after TTL expiry")
	}
}

func TestResetGuard(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 10)

	e.mu.Lock()
	e.guard["model_a"] = time.Now()
	e.guard["model_b"] = time.Now()
	e.mu.Unlock()

	e.ResetGuard("model_a")

	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.guard["model_a"].IsZero() {
		t.Error("expected model_a guard to be cleared")
	}
	if e.guard["model_b"].IsZero() {
		t.Error("expected model_b guard to remain intact")
	}
}

func TestResetAllGuards(t *testing.T) {
	e := NewEngine(slog.Default())
	e.Configure(true, 10)

	e.mu.Lock()
	e.guard["model_a"] = time.Now()
	e.guard["model_b"] = time.Now()
	e.guard["claude_5h"] = time.Now()
	e.mu.Unlock()

	e.ResetAllGuards()

	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.guard) != 0 {
		t.Errorf("expected all guards cleared, got %d remaining", len(e.guard))
	}
}
