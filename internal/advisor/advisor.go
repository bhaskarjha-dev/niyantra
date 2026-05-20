// Package advisor provides account switching recommendations based on
// multi-factor scoring of quota status, burn rates, and reset timers.
//
// The advisor is stateless — a pure function that takes current data
// and returns a recommendation. No database writes, no side effects.
package advisor

import (
	"fmt"
	"math"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/readiness"
	"github.com/bhaskarjha-com/niyantra/internal/tracker"
)

// Recommendation is the advisor's output.
type Recommendation struct {
	Action           string         `json:"action"` // "switch", "stay", "wait"
	Mode             string         `json:"mode"`   // "ranking" or "current_account"
	CurrentAccountID int64          `json:"currentAccountId,omitempty"`
	BestAccount      *AccountScore  `json:"bestAccount"`
	Alternatives     []AccountScore `json:"alternatives"`
	Reason           string         `json:"reason"`
	GeneratedAt      time.Time      `json:"generatedAt"`
}

// AccountScore represents a scored account with breakdown factors.
type AccountScore struct {
	AccountID      int64   `json:"accountId"`
	Email          string  `json:"email"`
	PlanName       string  `json:"planName"`
	Score          float64 `json:"score"`
	RemainingPct   float64 `json:"remainingPct"`
	BurnRate       float64 `json:"burnRate"`
	MinutesToReset int     `json:"minutesToReset"`
	IsExhausted    bool    `json:"isExhausted"`
	TTXLabel       string  `json:"ttxLabel,omitempty"` // F7: human-readable "~2.5h", "~45m", "idle"
}

// Scoring weights (60% remaining, 20% burn rate, 20% reset time)
const (
	weightRemaining = 0.60
	weightBurnRate  = 0.20
	weightResetTime = 0.20
	switchThreshold = 15.0 // Score difference to trigger "switch"
	exhaustedPct    = 10.0 // Below this %, account is considered near-exhausted
	waitResetMin    = 30   // Minutes until reset to suggest "wait"
)

// Recommend produces a switching recommendation from current data.
// snapshots: latest snapshot per account (from store.LatestPerAccount)
// summaries: per-model usage summaries keyed by account ID (nil if no intelligence)
func Recommend(snapshots []*client.Snapshot, summariesByAccount map[int64][]*tracker.UsageSummary) *Recommendation {
	return RecommendWithCurrent(snapshots, summariesByAccount, 0)
}

// RecommendWithCurrent produces switching guidance only when the caller supplies
// the current account. Without that context, it returns ranked options.
func RecommendWithCurrent(snapshots []*client.Snapshot, summariesByAccount map[int64][]*tracker.UsageSummary, currentAccountID int64) *Recommendation {
	rec := &Recommendation{
		GeneratedAt:      time.Now().UTC(),
		CurrentAccountID: currentAccountID,
	}
	if currentAccountID > 0 {
		rec.Mode = "current_account"
	} else {
		rec.Mode = "ranking"
	}

	if len(snapshots) == 0 {
		rec.Action = "rank"
		rec.Reason = "No accounts tracked yet. Capture a snapshot first."
		return rec
	}

	// Calculate readiness for each account
	accounts := readiness.Calculate(snapshots, 0.0)

	// Score each account
	var scores []AccountScore
	for _, acct := range accounts {
		score := scoreAccount(acct, summariesByAccount)
		// N12: Penalize stale accounts — stale data should never be recommended
		if acct.Staleness > 6*time.Hour {
			score.Score *= 0.3
		}
		scores = append(scores, score)
	}

	if len(scores) == 0 {
		rec.Action = "rank"
		rec.Reason = "No scoreable accounts found."
		return rec
	}

	// Find the best score
	bestIdx := 0
	for i, s := range scores {
		if s.Score > scores[bestIdx].Score {
			bestIdx = i
		}
	}

	best := scores[bestIdx]
	rec.BestAccount = &best

	// Build alternatives (all except best, sorted by score descending)
	for i, s := range scores {
		if i != bestIdx {
			rec.Alternatives = append(rec.Alternatives, s)
		}
	}

	// Without a caller-supplied current account, only rank accounts. Do not
	// invent a "switch from" account from slice order or score ordering.
	if currentAccountID <= 0 {
		rec.Action = "rank"
		rec.Reason = fmt.Sprintf("Best ranked account is %s with %.0f%% remaining (score %.0f). No current account was supplied, so no switch action is inferred.",
			best.Email, best.RemainingPct, best.Score)
		return rec
	}

	var current *AccountScore
	for i := range scores {
		if scores[i].AccountID == currentAccountID {
			current = &scores[i]
			break
		}
	}
	if current == nil {
		rec.Action = "rank"
		rec.Mode = "ranking"
		rec.Reason = fmt.Sprintf("Current account %d was not found. Best ranked account is %s with %.0f%% remaining.",
			currentAccountID, best.Email, best.RemainingPct)
		return rec
	}

	// Determine action
	if len(scores) == 1 {
		// Only one account — always "stay"
		rec.Action = "stay"
		rec.Reason = fmt.Sprintf("Only one account tracked (%s). Remaining: %.0f%%.", best.Email, best.RemainingPct)
		return rec
	}

	// Check if ALL accounts are near-exhausted → "wait" if any reset soon
	allExhausted := true
	var soonestReset int
	soonestEmail := ""
	for _, s := range scores {
		if s.RemainingPct >= exhaustedPct {
			allExhausted = false
		}
		if soonestReset == 0 || (s.MinutesToReset > 0 && s.MinutesToReset < soonestReset) {
			soonestReset = s.MinutesToReset
			soonestEmail = s.Email
		}
	}

	if allExhausted && soonestReset > 0 && soonestReset <= waitResetMin {
		rec.Action = "wait"
		rec.Reason = fmt.Sprintf("All accounts below %.0f%%. %s resets in %d minutes — wait for it.", exhaustedPct, soonestEmail, soonestReset)
		return rec
	}

	if len(rec.Alternatives) > 0 {
		if best.AccountID != current.AccountID && best.Score-current.Score >= switchThreshold {
			rec.Action = "switch"
			ttxContext := ""
			if best.TTXLabel != "" {
				ttxContext = fmt.Sprintf(" TTX: %s.", best.TTXLabel)
			}
			rec.Reason = fmt.Sprintf("Switch from %s to %s (%.0f%% remaining, score %.0f).%s Current account scores %.0f.",
				current.Email, best.Email, best.RemainingPct, best.Score, ttxContext, current.Score)
		} else {
			rec.Action = "stay"
			ttxContext := ""
			if best.TTXLabel != "" {
				ttxContext = fmt.Sprintf(" TTX: %s.", best.TTXLabel)
			}
			rec.Reason = fmt.Sprintf("Stay on %s. Best ranked account is %s with %.0f%% remaining (score %.0f).%s No significant advantage in switching.",
				current.Email, best.Email, best.RemainingPct, best.Score, ttxContext)
		}
	} else {
		rec.Action = "stay"
		rec.Reason = fmt.Sprintf("Using %s with %.0f%% remaining.", best.Email, best.RemainingPct)
	}

	return rec
}

// scoreAccount computes a weighted score for a single account.
func scoreAccount(acct readiness.AccountReadiness, summariesByAccount map[int64][]*tracker.UsageSummary) AccountScore {
	as := AccountScore{
		AccountID: acct.AccountID,
		Email:     acct.Email,
		PlanName:  acct.PlanName,
	}

	// 1. Remaining % — weighted average across all groups
	totalRemaining := 0.0
	groupCount := 0
	minResetMinutes := 0
	isExhausted := false

	for _, g := range acct.Groups {
		totalRemaining += g.RemainingPercent
		groupCount++
		if g.IsExhausted {
			isExhausted = true
		}
		resetMin := int(g.TimeUntilResetSec / 60)
		if minResetMinutes == 0 || (resetMin > 0 && resetMin < minResetMinutes) {
			minResetMinutes = resetMin
		}
	}

	if groupCount > 0 {
		as.RemainingPct = totalRemaining / float64(groupCount)
	}
	as.MinutesToReset = minResetMinutes
	as.IsExhausted = isExhausted

	// 2. Burn rate — from tracker intelligence (if available)
	burnRateBonus := 50.0 // default: no data = neutral
	minTTXHours := -1.0   // F7: track minimum TTX across all models
	if summariesByAccount != nil {
		if summaries, ok := summariesByAccount[acct.AccountID]; ok {
			avgRate := 0.0
			rateCount := 0
			for _, s := range summaries {
				if s.HasIntelligence {
					avgRate += s.CurrentRate
					rateCount++
					// F7: Track projected exhaustion to find minimum TTX
					if s.CurrentRate > 0 && s.RemainingFraction > 0 {
						ttx := s.RemainingFraction / s.CurrentRate
						if minTTXHours < 0 || ttx < minTTXHours {
							minTTXHours = ttx
						}
					}
				}
			}
			if rateCount > 0 {
				avgRate /= float64(rateCount)
				as.BurnRate = avgRate
				// Lower burn rate = higher bonus (100 - rate scaled to 0-100)
				burnRateBonus = math.Max(0, 100-avgRate*100)
			}
		}
	}

	// F7: Set TTX label from minimum TTX across this account's models
	if minTTXHours >= 0 {
		if minTTXHours <= 0 {
			as.TTXLabel = "exhausted"
		} else if minTTXHours < 1.0/60.0 {
			as.TTXLabel = "<1m"
		} else if minTTXHours < 1.0 {
			mins := int(minTTXHours * 60)
			as.TTXLabel = fmt.Sprintf("~%dm", mins)
		} else if minTTXHours >= 24 {
			days := minTTXHours / 24
			if days >= 2 {
				as.TTXLabel = fmt.Sprintf("~%.0fd", days)
			} else {
				as.TTXLabel = fmt.Sprintf("~%.0fh", minTTXHours)
			}
		} else {
			as.TTXLabel = fmt.Sprintf("~%.1fh", minTTXHours)
		}
	} else if as.BurnRate == 0 && as.RemainingPct > 0 {
		as.TTXLabel = "idle"
	}

	// 3. Reset time bonus — only relevant when remaining is low
	resetBonus := 50.0 // default: neutral
	if as.RemainingPct < 20 && minResetMinutes > 0 {
		// Sooner reset = higher bonus (inverted scale)
		hoursUntil := float64(minResetMinutes) / 60.0
		resetBonus = math.Max(0, 100-(hoursUntil/5.0*100))
	}

	// Compute weighted score
	as.Score = (as.RemainingPct * weightRemaining) +
		(burnRateBonus * weightBurnRate) +
		(resetBonus * weightResetTime)

	// Round to 1 decimal
	as.Score = math.Round(as.Score*10) / 10

	return as
}
