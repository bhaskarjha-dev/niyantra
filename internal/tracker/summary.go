package tracker

import (
	"fmt"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

// UsageSummary contains computed usage statistics for an Antigravity model.
type UsageSummary struct {
	AccountID         int64      `json:"accountId"`
	AccountEmail      string     `json:"accountEmail"`
	Provider          string     `json:"provider"`
	PlanName          string     `json:"planName"`
	ModelID           string     `json:"modelId"`
	Label             string     `json:"label"`
	Group             string     `json:"group"`
	RemainingFraction float64    `json:"remainingFraction"`
	UsagePercent      float64    `json:"usagePercent"`
	IsExhausted       bool       `json:"isExhausted"`
	ResetTime         *time.Time `json:"resetTime"`
	TimeUntilReset    string     `json:"timeUntilReset"`

	// Intelligence (requires ≥10 min of data for sliding-window, ≥30 min for cycle-based)
	CurrentRate         float64    `json:"currentRate"`         // usage fraction per hour
	ProjectedUsage      float64    `json:"projectedUsage"`      // projected usage at reset (0.0-1.0)
	ProjectedExhaustion *time.Time `json:"projectedExhaustion"` // when quota hits 0 at current rate
	HasIntelligence     bool       `json:"hasIntelligence"`     // true if rate data is available
	RateSource          string     `json:"rateSource"`          // "sliding_window" or "cycle_average"

	// History
	CompletedCycles int     `json:"completedCycles"`
	AvgPerCycle     float64 `json:"avgPerCycle"`
	PeakCycle       float64 `json:"peakCycle"`
	TotalTracked    float64 `json:"totalTracked"`

	// Active cycle info
	CycleAge       string `json:"cycleAge"`
	CycleSnapshots int    `json:"cycleSnapshots"`
}

// BudgetForecast summarizes budget headroom against recurring subscription
// commitments. Niyantra does not yet persist an observed daily spend ledger, so
// these values describe recurring commitments rather than a true usage-based
// month-to-date forecast.
type BudgetForecast struct {
	MonthlyBudget            float64 `json:"monthlyBudget"`
	CurrentSpend             float64 `json:"currentSpend"`
	RecurringMonthlySpend    float64 `json:"recurringMonthlySpend"`
	ProjectedMonthlySpend    float64 `json:"projectedMonthlySpend"`
	BurnRatePerDay           float64 `json:"burnRate"`
	DaysUntilBudgetExhausted *int    `json:"daysUntilBudgetExhausted"`
	OnTrack                  bool    `json:"onTrack"`
	DayOfMonth               int     `json:"dayOfMonth"`
	DaysInMonth              int     `json:"daysInMonth"`
	DataMode                 string  `json:"dataMode"`
	ObservedSpendAvailable   bool    `json:"observedSpendAvailable"`
}

// UsageSummaryForModel computes intelligence for a single model.
func (t *Tracker) UsageSummaryForModel(modelID string, accountID int64, model client.ModelQuota) (*UsageSummary, error) {
	summary := &UsageSummary{
		ModelID:           model.ModelID,
		Label:             model.Label,
		Group:             client.GroupForModel(model.ModelID, model.Label),
		RemainingFraction: model.RemainingFraction,
		UsagePercent:      (1.0 - model.RemainingFraction) * 100,
		IsExhausted:       model.IsExhausted,
		ResetTime:         model.ResetTime,
	}

	if model.ResetTime != nil {
		remaining := time.Until(*model.ResetTime)
		if remaining < 0 {
			remaining = 0
		}
		summary.TimeUntilReset = formatDuration(remaining)
	}

	// Get active cycle
	activeCycle, err := t.store.ActiveCycle(modelID, accountID)
	if err != nil {
		return summary, nil // Return basic summary without intelligence
	}

	// Get completed history
	history, err := t.store.CycleHistory(modelID, accountID, 50)
	if err != nil {
		return summary, nil
	}

	summary.CompletedCycles = len(history)

	// Compute history stats
	if len(history) > 0 {
		var totalDelta float64
		for _, cycle := range history {
			totalDelta += cycle.TotalDelta
			if cycle.PeakUsage > summary.PeakCycle {
				summary.PeakCycle = cycle.PeakUsage
			}
		}
		summary.AvgPerCycle = totalDelta / float64(len(history))
		summary.TotalTracked = totalDelta
	}

	// Add active cycle intelligence
	if activeCycle != nil {
		summary.TotalTracked += activeCycle.TotalDelta
		if activeCycle.PeakUsage > summary.PeakCycle {
			summary.PeakCycle = activeCycle.PeakUsage
		}
		summary.CycleSnapshots = activeCycle.SnapshotCount

		cycleAge := time.Since(activeCycle.CycleStart)
		summary.CycleAge = formatDuration(cycleAge)

		// F7: Rate calculation — use cycle-average as fallback.
		// The sliding-window rate (from forecast package) is injected via
		// ApplySlidingRate() and is strongly preferred over this lifetime average.
		// The cycle average is only used when no recent snapshot data is available.
		if cycleAge.Minutes() >= 30 && activeCycle.TotalDelta > 0 {
			summary.CurrentRate = activeCycle.TotalDelta / cycleAge.Hours()
			summary.HasIntelligence = true
			summary.RateSource = "cycle_average"

			computeProjections(summary)
		}
	}

	return summary, nil
}

// computeProjections computes projected usage and exhaustion time from CurrentRate.
// This is called both by the cycle-average fallback and after ApplySlidingRate.
func computeProjections(s *UsageSummary) {
	if s.ResetTime == nil || s.CurrentRate <= 0 {
		return
	}

	hoursLeft := time.Until(*s.ResetTime).Hours()
	if hoursLeft <= 0 {
		return
	}

	currentUsage := 1.0 - s.RemainingFraction
	projected := currentUsage + s.CurrentRate*hoursLeft
	if projected > 1.0 {
		projected = 1.0
	}
	s.ProjectedUsage = projected

	// Calculate when quota will exhaust at current rate
	if s.RemainingFraction > 0 {
		hoursToExhaust := s.RemainingFraction / s.CurrentRate
		exhaustTime := time.Now().Add(time.Duration(hoursToExhaust * float64(time.Hour)))
		if exhaustTime.Before(*s.ResetTime) {
			s.ProjectedExhaustion = &exhaustTime
		}
	}
}

// ApplySlidingRate replaces the cycle-average rate with an accurate
// sliding-window rate computed from recent snapshot history.
// This should be called after UsageSummaryForModel to upgrade the rate data.
func ApplySlidingRate(s *UsageSummary, rate float64, dataPoints int) {
	if s == nil || dataPoints < 2 {
		return
	}

	s.CurrentRate = rate
	s.HasIntelligence = true
	s.RateSource = "sliding_window"

	// Clear old projections and recompute with accurate rate
	s.ProjectedUsage = 0
	s.ProjectedExhaustion = nil
	computeProjections(s)
}

// AllUsageSummaries computes intelligence for all models of an account.
func (t *Tracker) AllUsageSummaries(snap *client.Snapshot, accountID int64) ([]*UsageSummary, error) {
	if snap == nil || len(snap.Models) == 0 {
		return nil, nil
	}

	var summaries []*UsageSummary
	for _, model := range snap.Models {
		if model.ModelID == "" {
			continue
		}
		s, err := t.UsageSummaryForModel(model.ModelID, accountID, model)
		if err != nil {
			t.logger.Warn("Failed to compute summary", "model", model.ModelID, "error", err)
			continue
		}
		s.AccountID = accountID
		s.AccountEmail = snap.Email
		s.Provider = "antigravity"
		s.PlanName = snap.PlanName
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// ComputeBudgetForecast compares the configured monthly budget against current
// recurring subscription commitments. It intentionally does not extrapolate a
// usage-based monthly spend because the store lacks trustworthy observed daily
// spend data.
func ComputeBudgetForecast(db *store.Store) *BudgetForecast {
	budget := db.GetConfigFloat("budget_monthly", 0)
	if budget <= 0 {
		return nil
	}

	// Get current month's spend from subscriptions
	overview, err := db.SubscriptionOverview()
	if err != nil {
		return nil
	}

	now := time.Now()
	dayOfMonth := now.Day()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()

	forecast := &BudgetForecast{
		MonthlyBudget:          budget,
		CurrentSpend:           overview.TotalMonthlySpend,
		RecurringMonthlySpend:  overview.TotalMonthlySpend,
		ProjectedMonthlySpend:  overview.TotalMonthlySpend,
		DayOfMonth:             dayOfMonth,
		DaysInMonth:            daysInMonth,
		DataMode:               "recurring_subscriptions",
		ObservedSpendAvailable: false,
	}

	// Expose a daily equivalent of recurring commitments for compatibility with
	// existing clients. This is not an observed burn rate.
	if daysInMonth > 0 {
		forecast.BurnRatePerDay = overview.TotalMonthlySpend / float64(daysInMonth)
	}

	forecast.OnTrack = forecast.ProjectedMonthlySpend <= budget

	return forecast
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "<1m"
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
