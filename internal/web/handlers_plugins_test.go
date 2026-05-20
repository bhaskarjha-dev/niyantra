package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bhaskarjha-com/niyantra/internal/agent"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func TestHandlePluginsReturnsEmptyArrays(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)

	srv := &Server{
		logger:   slog.Default(),
		store:    openTestStore(t),
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	rec := httptest.NewRecorder()
	srv.handlePlugins(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got := string(payload["plugins"]); got != "[]" {
		t.Fatalf("plugins = %s, want []", got)
	}
	if got := string(payload["errors"]); got != "[]" {
		t.Fatalf("errors = %s, want []", got)
	}
}

func TestHandlePluginConfigRefreshesRuntimeAndDataSource(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)
	createTestPlugin(t, home, "fixture-plugin")

	st := openTestStore(t)
	poller := agent.NewPollingAgent(nil, st, nil, time.Hour, slog.Default())
	poller.SetPollingCheck(func() bool { return false })

	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}
	srv.agentMgr.Start(poller)
	t.Cleanup(srv.agentMgr.Stop)

	enableBody := bytes.NewBufferString(`{"enabled":"true","api_key":"sekret","region":"us-east-1"}`)
	enableReq := httptest.NewRequest(http.MethodPut, "/api/plugins/fixture-plugin/config", enableBody)
	enableReq.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginConfig(rec, enableReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("enable config: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	runtimePlugins := srv.agentMgr.Agent().Plugins()
	if len(runtimePlugins) != 1 {
		t.Fatalf("expected 1 runtime plugin, got %d", len(runtimePlugins))
	}
	if !runtimePlugins[0].Enabled {
		t.Fatal("expected runtime plugin to be enabled")
	}
	if runtimePlugins[0].Config["api_key"] != "sekret" {
		t.Fatalf("api_key = %q, want %q", runtimePlugins[0].Config["api_key"], "sekret")
	}
	if runtimePlugins[0].Config["region"] != "us-east-1" {
		t.Fatalf("region = %q, want %q", runtimePlugins[0].Config["region"], "us-east-1")
	}

	sources, err := st.AllDataSources()
	if err != nil {
		t.Fatalf("AllDataSources: %v", err)
	}
	source := findDataSourceByID(sources, "plugin_fixture-plugin")
	if source == nil {
		t.Fatal("expected plugin data source to exist")
	}
	if !source.Enabled {
		t.Fatal("expected plugin data source to be enabled")
	}

	disableReq := httptest.NewRequest(http.MethodPut, "/api/plugins/fixture-plugin/config", bytes.NewBufferString(`{"enabled":"false"}`))
	disableReq.SetPathValue("id", "fixture-plugin")
	disableRec := httptest.NewRecorder()
	srv.handlePluginConfig(disableRec, disableReq)

	if disableRec.Code != http.StatusOK {
		t.Fatalf("disable config: expected 200, got %d (%s)", disableRec.Code, disableRec.Body.String())
	}

	runtimePlugins = srv.agentMgr.Agent().Plugins()
	if len(runtimePlugins) != 1 {
		t.Fatalf("expected 1 runtime plugin after disable, got %d", len(runtimePlugins))
	}
	if runtimePlugins[0].Enabled {
		t.Fatal("expected runtime plugin to be disabled")
	}

	sources, err = st.AllDataSources()
	if err != nil {
		t.Fatalf("AllDataSources after disable: %v", err)
	}
	source = findDataSourceByID(sources, "plugin_fixture-plugin")
	if source == nil {
		t.Fatal("expected plugin data source to still exist after disable")
	}
	if source.Enabled {
		t.Fatal("expected plugin data source to be disabled")
	}
}

func TestHandlePluginConfigRejectsEnableWithMissingRequiredConfig(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)
	createTestPlugin(t, home, "fixture-plugin")

	st := openTestStore(t)
	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/plugins/fixture-plugin/config", bytes.NewBufferString(`{"enabled":"true"}`))
	req.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if st.GetConfigBool("plugin_fixture-plugin_enabled") {
		t.Fatal("plugin should not be enabled when required config is missing")
	}
}

func TestHandlePluginConfigSurfacesSecretWriteFailure(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)
	createTestPlugin(t, home, "fixture-plugin")

	dbPath := filepath.Join(t.TempDir(), "niyantra-test.db")
	st, err := store.Open(dbPath, store.WithSecretBackend(failingSecretBackend{}))
	if err != nil {
		t.Fatalf("store.Open(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { st.Close() })

	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/plugins/fixture-plugin/config", bytes.NewBufferString(`{"api_key":"sekret"}`))
	req.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginConfig(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := st.GetConfig("plugin_fixture-plugin_api_key"); got != "" {
		t.Fatalf("api key should not be persisted after secret backend failure, got %q", got)
	}
}

func TestHandlePluginRunDoesNotPersistSnapshots(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)
	createTestPlugin(t, home, "fixture-plugin")

	st := openTestStore(t)
	if _, err := st.SetConfig("plugin_fixture-plugin_api_key", "sekret"); err != nil {
		t.Fatalf("SetConfig api_key: %v", err)
	}

	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/fixture-plugin/run", nil)
	req.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginRun(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}
	if st.PluginSnapshotCount() != 0 {
		t.Fatalf("expected manual test run to avoid persistence, got %d snapshots", st.PluginSnapshotCount())
	}
	if _, err := st.LatestPluginSnapshot("fixture-plugin"); err == nil {
		t.Fatal("expected no persisted plugin snapshot after manual test run")
	}
}

func TestHandlePluginRunRejectsMissingRequiredConfig(t *testing.T) {
	home := t.TempDir()
	setPluginHomeEnv(t, home)
	createTestPlugin(t, home, "fixture-plugin")

	st := openTestStore(t)
	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/fixture-plugin/run", nil)
	req.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginRun(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}
	if st.PluginSnapshotCount() != 0 {
		t.Fatalf("expected no plugin snapshots after rejected run, got %d", st.PluginSnapshotCount())
	}
}

func TestHandlePluginStatusReturnsDirectSnapshot(t *testing.T) {
	st := openTestStore(t)
	if _, err := st.InsertPluginSnapshot(&store.PluginSnapshot{
		PluginID:      "fixture-plugin",
		Provider:      "fixture",
		Label:         "Fixture Plugin",
		Email:         "fixture@example.com",
		UsagePct:      12.5,
		UsageDisplay:  "12.5%",
		Plan:          "Pro",
		ModelsJSON:    "[]",
		MetadataJSON:  "{}",
		CaptureMethod: "plugin",
	}); err != nil {
		t.Fatalf("InsertPluginSnapshot: %v", err)
	}

	srv := &Server{
		logger:   slog.Default(),
		store:    st,
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/fixture-plugin/status", nil)
	req.SetPathValue("id", "fixture-plugin")
	rec := httptest.NewRecorder()
	srv.handlePluginStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var snap store.PluginSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if snap.PluginID != "fixture-plugin" {
		t.Fatalf("pluginId = %q, want %q", snap.PluginID, "fixture-plugin")
	}
}

func TestHandlePluginStatusReturnsNotFoundWithoutSnapshot(t *testing.T) {
	srv := &Server{
		logger:   slog.Default(),
		store:    openTestStore(t),
		agentMgr: agent.NewManager(slog.Default()),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/missing/status", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()
	srv.handlePluginStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func setPluginHomeEnv(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	volume := filepath.VolumeName(home)
	t.Setenv("HOMEDRIVE", volume)
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, volume))
}

func createTestPlugin(t *testing.T, home, pluginID string) {
	t.Helper()

	pluginDir := filepath.Join(home, ".niyantra", "plugins", pluginID)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", pluginDir, err)
	}

	manifest := fmt.Sprintf(`{
  "id": %q,
  "name": "Fixture Plugin",
  "version": "1.0.0",
  "description": "Fixture plugin for tests",
  "author": "tests",
  "entryPoint": "capture.ps1",
  "timeout": 5,
  "config": {
    "api_key": { "type": "string", "label": "API Key", "required": true, "secret": true },
    "region": { "type": "string", "label": "Region", "default": "us-default-1" }
  }
}`, pluginID)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(plugin.json): %v", err)
	}

	script := `$inputJson = [Console]::In.ReadToEnd()
$request = $inputJson | ConvertFrom-Json
$apiKey = ""
$region = ""
if ($request.config.api_key) { $apiKey = [string]$request.config.api_key }
if ($request.config.region) { $region = [string]$request.config.region }
$result = @{
  status = "ok"
  data = @{
    provider = "fixture"
    label = "Fixture Plugin"
    email = "fixture@example.com"
    usage_pct = 12.5
    usage_display = "12.5%"
    plan = "Pro"
    models = @()
    metadata = @{
      api_key = $apiKey
      region = $region
    }
  }
}
[Console]::Out.Write(($result | ConvertTo-Json -Depth 6 -Compress))
`
	if err := os.WriteFile(filepath.Join(pluginDir, "capture.ps1"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(capture.ps1): %v", err)
	}
}

func findDataSourceByID(sources []*store.DataSource, id string) *store.DataSource {
	for _, source := range sources {
		if source.ID == id {
			return source
		}
	}
	return nil
}

type failingSecretBackend struct{}

func (failingSecretBackend) Set(string, string) error {
	return errors.New("secret backend unavailable")
}

func (failingSecretBackend) Get(string) (string, error) {
	return "", errors.New("secret backend unavailable")
}

func (failingSecretBackend) Delete(string) error {
	return nil
}
