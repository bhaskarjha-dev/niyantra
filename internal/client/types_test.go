package client

import "testing"

func TestGroupModelsMarksGroupExhaustedWhenAnyMemberIsExhausted(t *testing.T) {
	groups := GroupModels([]ModelQuota{
		{
			ModelID:           "claude-sonnet",
			Label:             "Claude Sonnet",
			RemainingFraction: 1.0,
			IsExhausted:       false,
		},
		{
			ModelID:           "gpt-4.1",
			Label:             "GPT-4.1",
			RemainingFraction: 0.0,
			IsExhausted:       true,
		},
	})

	for _, group := range groups {
		if group.GroupKey != GroupClaudeGPT {
			continue
		}
		if !group.IsExhausted {
			t.Fatalf("expected %s to be exhausted when any member is exhausted", group.GroupKey)
		}
		if group.RemainingPercent != 50 {
			t.Fatalf("remaining percent = %.0f, want 50", group.RemainingPercent)
		}
		return
	}

	t.Fatal("claude_gpt group not found")
}

func TestGroupForModelReturnsUnknownForUnmappedModel(t *testing.T) {
	if got := GroupForModel("local-llm-v1", "Local LLM"); got != GroupUnknown {
		t.Fatalf("GroupForModel(local-llm-v1) = %q, want %q", got, GroupUnknown)
	}
}

func TestGroupForModelClassifiesKnownFamilies(t *testing.T) {
	tests := []struct {
		modelID string
		label   string
		want    string
	}{
		{modelID: "claude-sonnet-4", label: "Claude Sonnet", want: GroupClaudeGPT},
		{modelID: "gpt-4.1", label: "GPT-4.1", want: GroupClaudeGPT},
		{modelID: "gemini-2.5-pro", label: "Gemini 2.5 Pro", want: GroupGeminiPro},
		{modelID: "gemini-2.5-flash", label: "Gemini 2.5 Flash", want: GroupGeminiFlash},
	}

	for _, tt := range tests {
		t.Run(tt.want+"/"+tt.modelID, func(t *testing.T) {
			if got := GroupForModel(tt.modelID, tt.label); got != tt.want {
				t.Fatalf("GroupForModel(%q, %q) = %q, want %q", tt.modelID, tt.label, got, tt.want)
			}
		})
	}
}

func TestGroupModelsOmitsEmptyGroupsAndKeepsUnknownSeparate(t *testing.T) {
	groups := GroupModels([]ModelQuota{
		{
			ModelID:           "local-llm-v1",
			Label:             "Local LLM",
			RemainingFraction: 0.7,
			RemainingPercent:  70,
		},
	})

	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].GroupKey != GroupUnknown {
		t.Fatalf("group key = %q, want %q", groups[0].GroupKey, GroupUnknown)
	}
	if groups[0].DisplayName != "Unknown" {
		t.Fatalf("display name = %q, want Unknown", groups[0].DisplayName)
	}
}
