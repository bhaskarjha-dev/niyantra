package web

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/client"
	"github.com/bhaskarjha-com/niyantra/internal/core"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func storedToClientSnapshot(s store.StoredSnapshot) *client.Snapshot {
	var models []client.ModelQuota
	_ = json.Unmarshal([]byte(s.ModelsJSON), &models)

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	} else {
		var h int64
		for _, c := range s.ID {
			h = 31*h + int64(c)
		}
		idVal = h
		if idVal < 0 {
			idVal = -idVal
		}
	}

	var accountID int64
	accountID, _ = strconv.ParseInt(s.AccountID, 10, 64)

	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var promptCredits float64
	var monthlyCredits int
	var rawJSONStr string
	var aiCredits []client.AICredit

	if dataMap != nil {
		if pc, ok := dataMap["prompt_credits"].(float64); ok {
			promptCredits = pc
		}
		if mcVal, ok := dataMap["monthly_credits"]; ok {
			switch v := mcVal.(type) {
			case float64:
				monthlyCredits = int(v)
			case int64:
				monthlyCredits = int(v)
			case int:
				monthlyCredits = v
			}
		}
		if rjs, ok := dataMap["raw_json"].(string); ok {
			rawJSONStr = rjs
		}
		if aicStr, ok := dataMap["ai_credits_json"].(string); ok && aicStr != "" {
			_ = json.Unmarshal([]byte(aicStr), &aiCredits)
		}
	}

	if rawJSONStr == "" {
		rawJSONStr = `{"userStatus":{"cascadeModelConfigData":{"unifiedPool":false}}}`
	}

	return &client.Snapshot{
		ID:             idVal,
		AccountID:      accountID,
		CapturedAt:     s.CapturedAt,
		Email:          s.Email,
		PlanName:       s.PlanTier,
		PromptCredits:  promptCredits,
		MonthlyCredits: monthlyCredits,
		Models:         models,
		AICredits:      aiCredits,
		RawJSON:        rawJSONStr,
		CaptureMethod:  s.CaptureMethod,
		CaptureSource:  s.CaptureSource,
		SourceID:       s.Provider,
	}
}

func clientToCoreSnapshot(snap *client.Snapshot, accountID int64) *core.Snapshot {
	var allModels []any
	for _, m := range snap.Models {
		allModels = append(allModels, m)
	}

	data := map[string]any{
		"prompt_credits":  snap.PromptCredits,
		"monthly_credits": snap.MonthlyCredits,
		"raw_json":        snap.RawJSON,
		"ai_credits_json": "[]",
	}
	if len(snap.AICredits) > 0 {
		if b, err := json.Marshal(snap.AICredits); err == nil {
			data["ai_credits_json"] = string(b)
		}
	}

	return &core.Snapshot{
		Provider:      "antigravity",
		AccountID:     strconv.FormatInt(accountID, 10),
		Email:         snap.Email,
		OverallPct:    0.0,
		PlanTier:      snap.PlanName,
		CostUSD:       0.0,
		Data:          data,
		Models:        allModels,
		CaptureMethod: snap.CaptureMethod,
		CaptureSource: snap.CaptureSource,
	}
}

func storedToPluginSnapshot(s store.StoredSnapshot) *store.PluginSnapshot {
	var dataMap map[string]interface{}
	_ = json.Unmarshal([]byte(s.DataJSON), &dataMap)

	var usageDisplay string
	var metadataMap map[string]interface{}
	if dataMap != nil {
		if ud, ok := dataMap["usage_display"].(string); ok {
			usageDisplay = ud
		}
		if md, ok := dataMap["metadata"].(map[string]interface{}); ok {
			metadataMap = md
		} else {
			metadataMap = make(map[string]interface{})
			for k, v := range dataMap {
				if k != "usage_display" && k != "metadata" {
					metadataMap[k] = v
				}
			}
		}
	}

	metadataJSON := "{}"
	if len(metadataMap) > 0 {
		if b, err := json.Marshal(metadataMap); err == nil {
			metadataJSON = string(b)
		}
	}

	pluginID := strings.TrimPrefix(s.Provider, "plugin_")

	var idVal int64
	if id, err := strconv.ParseInt(s.ID, 10, 64); err == nil {
		idVal = id
	}

	return &store.PluginSnapshot{
		ID:            idVal,
		PluginID:      pluginID,
		Provider:      s.Provider,
		Label:         "",
		Email:         s.Email,
		UsagePct:      s.OverallPct,
		UsageDisplay:  usageDisplay,
		Plan:          s.PlanTier,
		ModelsJSON:    s.ModelsJSON,
		MetadataJSON:  metadataJSON,
		CapturedAt:    s.CapturedAt.Format(time.RFC3339),
		CaptureMethod: s.CaptureMethod,
	}
}
