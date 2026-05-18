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
