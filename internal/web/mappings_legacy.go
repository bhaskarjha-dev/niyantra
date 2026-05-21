package web

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func storedToCodexSnapshot(s store.StoredSnapshot) *store.CodexSnapshot {
	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var fiveHourPct float64
	var sevenDayPct *float64
	var codeReviewPct *float64
	var fiveHourReset *time.Time
	var sevenDayReset *time.Time
	var creditsBalance *float64

	if dataMap != nil {
		if pct, ok := dataMap["five_hour_pct"].(float64); ok {
			fiveHourPct = pct
		}
		if pctVal, ok := dataMap["seven_day_pct"]; ok && pctVal != nil {
			if f, ok := pctVal.(float64); ok {
				sevenDayPct = &f
			}
		}
		if pctVal, ok := dataMap["code_review_pct"]; ok && pctVal != nil {
			if f, ok := pctVal.(float64); ok {
				codeReviewPct = &f
			}
		}
		if rstVal, ok := dataMap["five_hour_reset"].(string); ok && rstVal != "" {
			if t, err := time.Parse(time.RFC3339, rstVal); err == nil {
				fiveHourReset = &t
			}
		}
		if rstVal, ok := dataMap["seven_day_reset"].(string); ok && rstVal != "" {
			if t, err := time.Parse(time.RFC3339, rstVal); err == nil {
				sevenDayReset = &t
			}
		}
		if balVal, ok := dataMap["credits_balance"]; ok && balVal != nil {
			if f, ok := balVal.(float64); ok {
				creditsBalance = &f
			}
		}
	}

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	}

	var ownerAccountID int64
	ownerAccountID, _ = strconv.ParseInt(s.AccountID, 10, 64)

	return &store.CodexSnapshot{
		ID:             idVal,
		AccountID:      s.AccountID,
		OwnerAccountID: ownerAccountID,
		Email:          s.Email,
		FiveHourPct:    fiveHourPct,
		SevenDayPct:    sevenDayPct,
		CodeReviewPct:  codeReviewPct,
		FiveHourReset:  fiveHourReset,
		SevenDayReset:  sevenDayReset,
		PlanType:       s.PlanTier,
		CreditsBalance: creditsBalance,
		CapturedAt:     s.CapturedAt,
		CaptureMethod:  s.CaptureMethod,
		CaptureSource:  s.CaptureSource,
	}
}

func storedToClaudeSnapshot(s store.StoredSnapshot) *store.ClaudeSnapshot {
	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var fiveHourPct float64
	var sevenDayPct *float64
	var fiveHourReset *time.Time
	var sevenDayReset *time.Time

	if dataMap != nil {
		if pct, ok := dataMap["five_hour_pct"].(float64); ok {
			fiveHourPct = pct
		}
		if pctVal, ok := dataMap["seven_day_pct"]; ok && pctVal != nil {
			if f, ok := pctVal.(float64); ok {
				sevenDayPct = &f
			}
		}
		if rstVal, ok := dataMap["five_hour_reset"].(string); ok && rstVal != "" {
			if t, err := time.Parse(time.RFC3339, rstVal); err == nil {
				fiveHourReset = &t
			}
		}
		if rstVal, ok := dataMap["seven_day_reset"].(string); ok && rstVal != "" {
			if t, err := time.Parse(time.RFC3339, rstVal); err == nil {
				sevenDayReset = &t
			}
		}
	}

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	}

	return &store.ClaudeSnapshot{
		ID:            idVal,
		FiveHourPct:   fiveHourPct,
		SevenDayPct:   sevenDayPct,
		FiveHourReset: fiveHourReset,
		SevenDayReset: sevenDayReset,
		CapturedAt:    s.CapturedAt,
		Source:        s.CaptureSource,
	}
}

func storedToCursorSnapshot(s store.StoredSnapshot) *store.CursorSnapshot {
	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var modelsMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.ModelsJSON), &modelsMap)

	var requestsUsed int
	var requestsMax int
	var startOfMonth string
	if dataMap != nil {
		if u, ok := dataMap["premium_used"].(float64); ok {
			requestsUsed = int(u)
		}
		if m, ok := dataMap["premium_limit"].(float64); ok {
			requestsMax = int(m)
		}
		if som, ok := dataMap["start_of_month"].(string); ok {
			startOfMonth = som
		}
	}

	var billingModel string
	var usedCents int
	var limitCents int
	var autoPct float64
	var apiPct float64
	var cycleEnd string
	if modelsMap != nil {
		if bm, ok := modelsMap["billingModel"].(string); ok {
			billingModel = bm
		}
		if uc, ok := modelsMap["usedCents"].(float64); ok {
			usedCents = int(uc)
		}
		if lc, ok := modelsMap["limitCents"].(float64); ok {
			limitCents = int(lc)
		}
		if ap, ok := modelsMap["autoPct"].(float64); ok {
			autoPct = ap
		}
		if ap, ok := modelsMap["apiPct"].(float64); ok {
			apiPct = ap
		}
		if ce, ok := modelsMap["cycleEnd"].(string); ok {
			cycleEnd = ce
		}
	}

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	}

	var accountID int64
	accountID, _ = strconv.ParseInt(s.AccountID, 10, 64)

	return &store.CursorSnapshot{
		ID:            idVal,
		AccountID:     accountID,
		Email:         s.Email,
		BillingModel:  billingModel,
		PlanTier:      s.PlanTier,
		RequestsUsed:  requestsUsed,
		RequestsMax:   requestsMax,
		UsedCents:     usedCents,
		LimitCents:    limitCents,
		UsagePct:      s.OverallPct,
		AutoPct:       autoPct,
		APIPct:        apiPct,
		CycleStart:    startOfMonth,
		CycleEnd:      cycleEnd,
		CapturedAt:    s.CapturedAt,
		CaptureMethod: s.CaptureMethod,
		CaptureSource: s.CaptureSource,
	}
}

func storedToCopilotSnapshot(s store.StoredSnapshot) *store.CopilotSnapshot {
	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var modelsMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.ModelsJSON), &modelsMap)

	var premiumPct float64
	var chatPct float64
	var username string
	if dataMap != nil {
		if pct, ok := dataMap["premium_pct"].(float64); ok {
			premiumPct = pct
		}
		if pct, ok := dataMap["chat_pct"].(float64); ok {
			chatPct = pct
		}
		if u, ok := dataMap["username"].(string); ok {
			username = u
		}
	}

	var hasPremium bool
	var hasChat bool
	if modelsMap != nil {
		if hp, ok := modelsMap["hasPremium"].(bool); ok {
			hasPremium = hp
		}
		if hc, ok := modelsMap["hasChat"].(bool); ok {
			hasChat = hc
		}
	}

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	}

	var accountID int64
	accountID, _ = strconv.ParseInt(s.AccountID, 10, 64)

	return &store.CopilotSnapshot{
		ID:            idVal,
		AccountID:     accountID,
		Email:         s.Email,
		Username:      username,
		Plan:          s.PlanTier,
		PremiumPct:    premiumPct,
		ChatPct:       chatPct,
		HasPremium:    hasPremium,
		HasChat:       hasChat,
		CapturedAt:    s.CapturedAt,
		CaptureMethod: s.CaptureMethod,
		CaptureSource: s.CaptureSource,
	}
}


