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
		{modelID: "gemini-2.5-pro", label: "Gemini 2.5 Pro", want: GroupGeminiUnified},
		{modelID: "gemini-2.5-flash", label: "Gemini 2.5 Flash", want: GroupGeminiUnified},
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

func TestDeduplicateProcesses(t *testing.T) {
	procs := []*processInfo{
		{
			PID:                 100,
			CSRFToken:           "token1",
			ExtensionServerPort: 1111,
			CommandLine:         `C:\Path\To\antigravity.exe --extension_server_port=1111 --csrf_token=token1 --other_stable_flag`,
		},
		{
			PID:                 200,
			CSRFToken:           "token2",
			ExtensionServerPort: 2222,
			CommandLine:         `C:\Path\To\antigravity.exe --extension_server_port=2222 --csrf_token=token2 --other_stable_flag`,
		},
		{
			PID:                 150,
			CSRFToken:           "token3",
			ExtensionServerPort: 3333,
			CommandLine:         `C:\Path\To\antigravity-ide.exe --extension_server_port=3333 --csrf_token=token3 --extension_server_csrf_token=csrf1 --https_server_port=4444 --lsp_port=5555 --parent_pipe_path=\\.\pipe\1`,
		},
		{
			PID:                 250,
			CSRFToken:           "token4",
			ExtensionServerPort: 6666,
			CommandLine:         `C:\Path\To\antigravity-ide.exe --extension_server_port=6666 --csrf_token=token4 --extension_server_csrf_token=csrf2 --https_server_port=7777 --lsp_port=8888 --parent_pipe_path=\\.\pipe\2`,
		},
	}

	deduped := deduplicateProcesses(procs)

	// We expect 2 processes left:
	// 1. Antigravity Main from PID 200 (since 200 > 100)
	// 2. Antigravity IDE from PID 250 (since 250 > 150)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 processes after deduplication, got %d", len(deduped))
	}

	var hasPID200, hasPID250 bool
	for _, p := range deduped {
		if p.PID == 200 {
			hasPID200 = true
		}
		if p.PID == 250 {
			hasPID250 = true
		}
	}

	if !hasPID200 {
		t.Error("expected process with PID 200 (newest Main) to be retained")
	}
	if !hasPID250 {
		t.Error("expected process with PID 250 (newest IDE) to be retained")
	}
}

func TestDeduplicateProcessesWorkspacePreference(t *testing.T) {
	procs := []*processInfo{
		{
			PID:         300, // Hub process (no workspace_id)
			CommandLine: `C:\Path\To\antigravity-ide.exe --extension_server_port=3000 --csrf_token=tok1 --subclient_type=ide`,
		},
		{
			PID:         200, // Workspace process (has workspace_id), lower PID but preferred
			CommandLine: `C:\Path\To\antigravity-ide.exe --extension_server_port=4000 --csrf_token=tok2 --subclient_type=ide --workspace_id=my_workspace`,
		},
	}

	deduped := deduplicateProcesses(procs)

	if len(deduped) != 1 {
		t.Fatalf("expected 1 process after deduplication, got %d", len(deduped))
	}

	if deduped[0].PID != 200 {
		t.Fatalf("expected workspace process with PID 200 to be selected, got PID %d", deduped[0].PID)
	}
}
