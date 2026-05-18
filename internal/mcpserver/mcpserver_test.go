package mcpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// testLogger returns a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestHTTPHandlerResponds verifies the Streamable HTTP handler returns
// a valid MCP JSON-RPC response (or acceptable error) for an initialize request.
func TestHTTPHandlerResponds(t *testing.T) {
	// Create a minimal MCPServer without a real store/tracker.
	// The handler should still respond to the MCP initialize method.
	m := New(nil, nil, testLogger(), "test")
	handler := m.HTTPHandler()

	// Send a valid JSON-RPC initialize request
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// The SDK should respond — either 200 with JSON-RPC result, or a protocol-level error.
	// Any response other than a panic or 5xx means the handler is wired correctly.
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected non-5xx response from MCP handler, got %d: %s", resp.StatusCode, string(respBody))
	}
}

// TestHTTPHandlerRejectsGETWithoutSession verifies GET without session returns
// an appropriate error (sessions require POST initialization first).
func TestHTTPHandlerRejectsGETWithoutSession(t *testing.T) {
	m := New(nil, nil, testLogger(), "test")
	handler := m.HTTPHandler()

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// GET without a valid session should be rejected
	if rec.Code == http.StatusOK {
		t.Error("expected non-200 for GET without session, got 200")
	}
}

func TestHTTPHandlerListsAllRegisteredTools(t *testing.T) {
	m := New(nil, nil, testLogger(), "test")
	handler := m.HTTPHandler()

	initReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
	))
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("Accept", "application/json, text/event-stream")
	initRec := httptest.NewRecorder()
	handler.ServeHTTP(initRec, initReq)

	if initRec.Code != http.StatusOK {
		t.Fatalf("initialize returned %d: %s", initRec.Code, initRec.Body.String())
	}

	sessionID := initRec.Result().Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("expected initialize response to include Mcp-Session-Id")
	}

	listReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	))
	listReq.Header.Set("Content-Type", "application/json")
	listReq.Header.Set("Accept", "application/json, text/event-stream")
	listReq.Header.Set("Mcp-Session-Id", sessionID)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("tools/list returned %d: %s", listRec.Code, listRec.Body.String())
	}

	rawBody := strings.TrimSpace(listRec.Body.String())
	if strings.HasPrefix(rawBody, "event:") {
		var dataLines []string
		for _, line := range strings.Split(rawBody, "\n") {
			if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}
		rawBody = strings.Join(dataLines, "\n")
	}

	var response struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(rawBody), &response); err != nil {
		t.Fatalf("unmarshal tools/list response: %v; body=%q", err, listRec.Body.String())
	}

	got := make([]string, 0, len(response.Result.Tools))
	for _, tool := range response.Result.Tools {
		got = append(got, tool.Name)
	}
	slices.Sort(got)

	want := []string{
		"analyze_spending",
		"best_model",
		"codex_status",
		"copilot_status",
		"git_commit_costs",
		"model_availability",
		"plugin_status",
		"quota_forecast",
		"quota_status",
		"switch_recommendation",
		"token_usage_stats",
		"budget_forecast",
		"usage_intelligence",
	}
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("registered tools mismatch\ngot:  %v\nwant: %v", got, want)
	}
}
