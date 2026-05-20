package client

import (
	"strings"
	"testing"
)

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

// TestRealWorldThreeProcessScenario mirrors the exact bug scenario:
// Main (PID 6628) + IDE Hub (PID 24980, stale) + IDE Workspace (PID 27288, active).
// After dedup, only Main + IDE Workspace should remain, and the workspace
// process should have correct port fields for prioritized connection.
func TestRealWorldThreeProcessScenario(t *testing.T) {
	procs := []*processInfo{
		{
			PID:                 6628,
			CSRFToken:           "1e10038a-029f-45b1-bd74-ff1c289a6764",
			ExtensionServerPort: 0,
			HTTPSServerPort:     0, // Main uses --https_server_port 0 (dynamic)
			LSPPort:             0,
			CommandLine:         `C:\Users\air\AppData\Local\Programs\Antigravity\resources\bin\language_server.exe --standalone --override_ide_name antigravity --subclient_type hub --override_ide_version 2.0.1 --override_user_agent_name antigravity --https_server_port 0 --csrf_token 1e10038a-029f-45b1-bd74-ff1c289a6764`,
		},
		{
			PID:                 24980,
			CSRFToken:           "7062b590-1734-41a0-a800-9c83b3d94653",
			ExtensionServerPort: 64136,
			HTTPSServerPort:     0,
			LSPPort:             0,
			CommandLine:         `"c:\Users\air\AppData\Local\Programs\Antigravity IDE\resources\app\node_modules\@anthropic\language_server_windows_x64.exe" --csrf_token 7062b590-1734-41a0-a800-9c83b3d94653 --extension_server_port 64136 --extension_server_csrf_token 37cbd63d-822a-4f88-9251-4f1dae0f21f8 --app_data_dir antigravity-ide --subclient_type ide --cloud_code_endpoint https://cloudcode-pa.googleapis.com`,
		},
		{
			PID:                 27288,
			CSRFToken:           "3079e398-5b41-4dcf-8d42-efe15b29da61",
			ExtensionServerPort: 52705,
			HTTPSServerPort:     52706,
			LSPPort:             54482,
			CommandLine:         `"c:\Users\air\AppData\Local\Programs\Antigravity IDE\resources\app\node_modules\@anthropic\language_server_windows_x64.exe" --enable_lsp --csrf_token 3079e398-5b41-4dcf-8d42-efe15b29da61 --extension_server_port 52705 --extension_server_csrf_token 6d72b020-09ba-4c8b-9d4b-316a4ee8863e --https_server_port 52706 --lsp_port 54482 --workspace_id file_d_3A_dev_pro_niyantra --cloud_code_endpoint https://daily-cloudcode-pa.googleapis.com --subclient_type ide --app_data_dir antigravity-ide --parent_pipe_path \\.\pipe\server_0c9ecbfd8009471d`,
		},
	}

	deduped := deduplicateProcesses(procs)

	// Should have 2 groups: Main + IDE
	if len(deduped) != 2 {
		t.Fatalf("expected 2 processes after dedup, got %d", len(deduped))
	}

	var main, ide *processInfo
	for _, p := range deduped {
		sig := getToolSignature(p.CommandLine)
		if strings.Contains(sig, "main|") {
			main = p
		} else if strings.Contains(sig, "ide|") {
			ide = p
		}
	}

	// Main process (PID 6628) should be retained
	if main == nil {
		t.Fatal("main process not found after dedup")
	}
	if main.PID != 6628 {
		t.Fatalf("expected main PID 6628, got %d", main.PID)
	}

	// IDE: workspace (PID 27288) should be preferred over hub (PID 24980)
	if ide == nil {
		t.Fatal("IDE process not found after dedup")
	}
	if ide.PID != 27288 {
		t.Fatalf("expected IDE workspace PID 27288 to be selected, got %d", ide.PID)
	}

	// Verify port fields are correctly set for port prioritization
	if ide.HTTPSServerPort != 52706 {
		t.Fatalf("expected IDE HTTPSServerPort 52706, got %d", ide.HTTPSServerPort)
	}
	if ide.ExtensionServerPort != 52705 {
		t.Fatalf("expected IDE ExtensionServerPort 52705, got %d", ide.ExtensionServerPort)
	}
	if ide.LSPPort != 54482 {
		t.Fatalf("expected IDE LSPPort 54482, got %d", ide.LSPPort)
	}
}

// TestParseFlagExtractsHTTPSPort verifies parseFlag and parseFlagInt correctly
// extract the HTTPS server port from real-world command lines.
func TestParseFlagExtractsHTTPSPort(t *testing.T) {
	tests := []struct {
		name    string
		cmdLine string
		flag    string
		want    int
	}{
		{
			name:    "workspace with https_server_port space-separated",
			cmdLine: `language_server.exe --https_server_port 52706 --lsp_port 54482`,
			flag:    "--https_server_port",
			want:    52706,
		},
		{
			name:    "workspace with https_server_port=0 (dynamic)",
			cmdLine: `language_server.exe --https_server_port 0 --csrf_token abc`,
			flag:    "--https_server_port",
			want:    0,
		},
		{
			name:    "hub without https_server_port",
			cmdLine: `language_server.exe --extension_server_port 64136 --csrf_token abc`,
			flag:    "--https_server_port",
			want:    0,
		},
		{
			name:    "lsp port extraction",
			cmdLine: `language_server.exe --lsp_port 54482 --workspace_id test`,
			flag:    "--lsp_port",
			want:    54482,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFlagInt(tt.cmdLine, tt.flag)
			if got != tt.want {
				t.Fatalf("parseFlagInt(%q, %q) = %d, want %d", tt.cmdLine, tt.flag, got, tt.want)
			}
		})
	}
}
