package readiness

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
)

// AccountReadiness represents the readiness state of a single account.
type AccountReadiness struct {
	AccountID           int64             `json:"accountId"`
	LatestSnapshotID    int64             `json:"latestSnapshotId"`
	Email               string            `json:"email"`
	PlanName            string            `json:"planName"`
	PlanTier            string            `json:"planTier"`
	OverageCredits      float64           `json:"overageCredits"`
	HasClaimedBonus2026 int               `json:"hasClaimedBonus2026"`
	Provider            string            `json:"provider"`
	Notes               string            `json:"notes"`
	Tags                string            `json:"tags"`
	PinnedGroup         string            `json:"pinnedGroup"`
	CreditRenewalDay    int               `json:"creditRenewalDay"`
	LastSeen            time.Time         `json:"lastSeen"`
	Staleness           time.Duration     `json:"-"`
	StalenessLabel      string            `json:"stalenessLabel"`
	IsReady             bool              `json:"isReady"`
	UnifiedPool         bool              `json:"unifiedPool"`
	Groups              []GroupReadiness  `json:"groups"`
	Models              []ModelDetail     `json:"models"`
	PromptCredits       float64           `json:"promptCredits"`
	MonthlyCredits      int               `json:"monthlyCredits"`
	AICredits           []client.AICredit `json:"aiCredits"`
}

// ModelDetail is a per-model quota entry for the dashboard.
type ModelDetail struct {
	ModelID           string  `json:"modelId"`
	Label             string  `json:"label"`
	RemainingPercent  float64 `json:"remainingPercent"`
	IsExhausted       bool    `json:"isExhausted"`
	ResetSeconds      float64 `json:"resetSeconds"`
	GroupKey          string  `json:"groupKey"`
	IsEstimated       bool    `json:"isEstimated,omitempty"`
	Basis             string  `json:"basis,omitempty"`
	Confidence        string  `json:"confidence,omitempty"`
	UnavailableReason string  `json:"unavailableReason,omitempty"`
}

// GroupReadiness represents the readiness state of a single quota group.
type GroupReadiness struct {
	GroupKey          string     `json:"groupKey"`
	DisplayName       string     `json:"displayName"`
	RemainingPercent  float64    `json:"remainingPercent"`
	IsExhausted       bool       `json:"isExhausted"`
	IsReady           bool       `json:"isReady"`
	Color             string     `json:"color"`
	ResetTime         *time.Time `json:"resetTime,omitempty"`
	TimeUntilResetSec float64    `json:"timeUntilResetSec"`
	IsEstimated       bool       `json:"isEstimated,omitempty"`
	Basis             string     `json:"basis,omitempty"`
	Confidence        string     `json:"confidence,omitempty"`
}

// Calculate computes readiness for all accounts from their latest snapshots.
// threshold is the minimum remainingFraction to consider "ready" (0.0 = any remaining).
func Calculate(snapshots []*client.Snapshot, threshold float64) []AccountReadiness {
	var results []AccountReadiness

	for _, snap := range snapshots {
		if snap == nil {
			continue
		}

		now := time.Now()
		staleness := now.Sub(snap.CapturedAt)

		var rawResp struct {
			UserStatus struct {
				CascadeModelConfigData struct {
					UnifiedPool bool `json:"unifiedPool"`
				} `json:"cascadeModelConfigData"`
			} `json:"userStatus"`
		}
		_ = json.Unmarshal([]byte(snap.RawJSON), &rawResp)

		ar := AccountReadiness{
			AccountID:        snap.AccountID,
			LatestSnapshotID: snap.ID,
			Email:            snap.Email,
			PlanName:         snap.PlanName,
			LastSeen:         snap.CapturedAt,
			Staleness:        staleness,
			StalenessLabel:   formatStaleness(staleness),
			IsReady:          true,
			PromptCredits:    snap.PromptCredits,
			MonthlyCredits:   snap.MonthlyCredits,
			AICredits:        snap.AICredits,
			UnifiedPool:      rawResp.UserStatus.CascadeModelConfigData.UnifiedPool,
		}

		// Q3: Always show actual time-ago — the time IS the staleness indicator
		// Visual dimming in the UI handles the "stale" signal instead

		// Per-model details + build corrected models for group computation
		// Bug fix: GroupModels must use reset-time-corrected values, not raw snapshot data.
		// Without this, group-level columns (Claude+GPT) show stale values even after reset.
		// Dynamic migration of snapshot models to ensure all 4 contemporary Google Gemini models are displayed
		var upgradedModels []client.ModelQuota
		var otherModels []client.ModelQuota
		var geminiModels []client.ModelQuota
		for _, m := range snap.Models {
			if client.GroupForModel(m.ModelID, m.Label) == client.GroupGeminiUnified {
				geminiModels = append(geminiModels, m)
			} else {
				otherModels = append(otherModels, m)
			}
		}

		upgradedModels = append(upgradedModels, otherModels...)

		if len(geminiModels) > 0 {
			var refFlash, refPro *client.ModelQuota
			for idx := range geminiModels {
				m := &geminiModels[idx]
				if strings.Contains(strings.ToLower(m.Label), "flash") || m.ModelID == "MODEL_PLACEHOLDER_M133" || m.ModelID == "MODEL_PLACEHOLDER_M20" || m.ModelID == "gemini-2.5-flash" {
					if refFlash == nil || m.ModelID == "MODEL_PLACEHOLDER_M133" {
						refFlash = m
					}
				}
				if strings.Contains(strings.ToLower(m.Label), "pro") || m.ModelID == "MODEL_PLACEHOLDER_M16" || m.ModelID == "MODEL_PLACEHOLDER_M36" || m.ModelID == "gemini-3.1-pro" || m.ModelID == "gemini-pro" {
					if refPro == nil || m.ModelID == "MODEL_PLACEHOLDER_M16" {
						refPro = m
					}
				}
			}

			if refFlash == nil && refPro != nil {
				clone := *refPro
				refFlash = &clone
			} else if refFlash == nil {
				refFlash = &client.ModelQuota{
					RemainingPercent: 100.0,
				}
			}
			if refPro == nil && refFlash != nil {
				clone := *refFlash
				refPro = &clone
			} else if refPro == nil {
				refPro = &client.ModelQuota{
					RemainingPercent: 100.0,
				}
			}

			// Initialize the 4 contemporary models from reference/synthesized objects
			m133 := *refFlash
			m133.ModelID = "MODEL_PLACEHOLDER_M133"
			m133.Label = "Gemini 3.5 Flash (High)"

			m20 := *refFlash
			m20.ModelID = "MODEL_PLACEHOLDER_M20"
			m20.Label = "Gemini 3.5 Flash (Medium)"

			m16 := *refPro
			m16.ModelID = "MODEL_PLACEHOLDER_M16"
			m16.Label = "Gemini 3.1 Pro (High)"

			m36 := *refPro
			m36.ModelID = "MODEL_PLACEHOLDER_M36"
			m36.Label = "Gemini 3.1 Pro (Low)"

			// Restore specific values if they were present
			for _, m := range geminiModels {
				switch m.ModelID {
				case "MODEL_PLACEHOLDER_M133":
					m133 = m
				case "MODEL_PLACEHOLDER_M20":
					m20 = m
				case "MODEL_PLACEHOLDER_M16":
					m16 = m
				case "MODEL_PLACEHOLDER_M36":
					m36 = m
				default:
					if m.Label == "Gemini 3.5 Flash (High)" || m.Label == "Gemini 3 Flash (High)" {
						m133 = m
						m133.ModelID = "MODEL_PLACEHOLDER_M133"
						m133.Label = "Gemini 3.5 Flash (High)"
					} else if m.Label == "Gemini 3.5 Flash (Medium)" || m.Label == "Gemini 3 Flash (Medium)" {
						m20 = m
						m20.ModelID = "MODEL_PLACEHOLDER_M20"
						m20.Label = "Gemini 3.5 Flash (Medium)"
					} else if m.Label == "Gemini 3.1 Pro (High)" {
						m16 = m
						m16.ModelID = "MODEL_PLACEHOLDER_M16"
						m16.Label = "Gemini 3.1 Pro (High)"
					} else if m.Label == "Gemini 3.1 Pro (Low)" {
						m36 = m
						m36.ModelID = "MODEL_PLACEHOLDER_M36"
						m36.Label = "Gemini 3.1 Pro (Low)"
					}
				}
			}

			upgradedModels = append(upgradedModels, m133, m20, m16, m36)
		}

		correctedModels := make([]client.ModelQuota, 0, len(upgradedModels))
		for _, m := range upgradedModels {
			m = client.ApplyResetInference(m, now)
			resetSec := 0.0
			if m.ResetTime != nil {
				resetSec = m.ResetTime.Sub(now).Seconds()
				if resetSec < 0 {
					resetSec = 0
				}
			}
			ar.Models = append(ar.Models, ModelDetail{
				ModelID:           m.ModelID,
				Label:             m.Label,
				RemainingPercent:  m.RemainingPercent,
				IsExhausted:       m.IsExhausted,
				ResetSeconds:      resetSec,
				GroupKey:          client.GroupForModel(m.ModelID, m.Label),
				IsEstimated:       m.IsEstimated,
				Basis:             m.Basis,
				Confidence:        m.Confidence,
				UnavailableReason: m.UnavailableReason,
			})

			// Build corrected ModelQuota for group computation
			correctedModels = append(correctedModels, m)
		}

		// Group models using CORRECTED values (not raw snapshot)
		groups := client.GroupModels(correctedModels)
		for _, g := range groups {
			groupEstimated := false
			for _, m := range correctedModels {
				if client.GroupForModel(m.ModelID, m.Label) == g.GroupKey && m.IsEstimated {
					groupEstimated = true
					break
				}
			}
			gr := GroupReadiness{
				GroupKey:         g.GroupKey,
				DisplayName:      g.DisplayName,
				RemainingPercent: g.RemainingFraction * 100,
				IsExhausted:      g.IsExhausted,
				Color:            g.Color,
				ResetTime:        g.ResetTime,
				IsEstimated:      groupEstimated,
			}
			if groupEstimated {
				gr.Basis = "contains_post_reset_estimate"
				gr.Confidence = "low"
			}

			if g.ResetTime != nil {
				sec := time.Until(*g.ResetTime).Seconds()
				if sec < 0 {
					sec = 0
				}
				gr.TimeUntilResetSec = sec
			}

			gr.IsReady = !g.IsExhausted && g.RemainingFraction > threshold
			if !gr.IsReady {
				ar.IsReady = false
			}

			ar.Groups = append(ar.Groups, gr)
		}

		if len(ar.Groups) == 0 {
			ar.IsReady = false
		}

		results = append(results, ar)
	}

	// Sort: ready accounts first, then by most recently seen
	sort.Slice(results, func(i, j int) bool {
		if results[i].IsReady != results[j].IsReady {
			return results[i].IsReady
		}
		return results[i].LastSeen.After(results[j].LastSeen)
	})

	return results
}

// formatStaleness returns a human-readable staleness label.
func formatStaleness(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}
