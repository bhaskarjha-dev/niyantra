package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/bhaskarjha-com/niyantra/internal/plugin"
)

type pluginInfo struct {
	Manifest     plugin.Manifest   `json:"manifest"`
	Dir          string            `json:"dir"`
	Enabled      bool              `json:"enabled"`
	Config       map[string]string `json:"config"`
	LastCapture  string            `json:"lastCapture,omitempty"`
	CaptureCount int64             `json:"captureCount"`
}

// handlePlugins returns all discovered plugins with their status.
// GET /api/plugins
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	pluginsDir := plugin.DefaultPluginsDir()
	plugins, errs := s.loadConfiguredPlugins()
	result := make([]pluginInfo, 0, len(plugins))
	dataSources := s.pluginSourceIndex()

	for _, p := range plugins {
		info := pluginInfo{
			Manifest: p.Manifest,
			Dir:      p.Dir,
			Enabled:  p.Enabled,
			Config:   make(map[string]string, len(p.Manifest.Config)),
		}

		for key, field := range p.Manifest.Config {
			val := p.Config[key]
			if field.Secret && val != "" {
				info.Config[key] = "configured"
			} else {
				info.Config[key] = val
			}
		}

		if ds, ok := dataSources["plugin_"+p.Manifest.ID]; ok {
			info.LastCapture = ds.LastCapture
			info.CaptureCount = ds.CaptureCount
		}

		result = append(result, info)
	}

	discoveryErrors := make([]string, 0, len(errs))
	for _, err := range errs {
		s.logger.Warn("Plugin discovery error", "error", err)
		discoveryErrors = append(discoveryErrors, err.Error())
	}

	writeJSON(w, map[string]any{
		"plugins":    result,
		"pluginsDir": pluginsDir,
		"errors":     discoveryErrors,
	})
}

// handlePluginStatus returns the latest snapshot for a specific plugin.
// GET /api/plugins/{id}/status
func (s *Server) handlePluginStatus(w http.ResponseWriter, r *http.Request) {
	pluginID := r.PathValue("id")
	if pluginID == "" {
		jsonError(w, "plugin id required", http.StatusBadRequest)
		return
	}

	snap, err := s.store.LatestPluginSnapshot(pluginID)
	if err != nil {
		jsonError(w, "plugin snapshot not found", http.StatusNotFound)
		return
	}

	writeJSON(w, snap)
}

// handlePluginRun manually triggers a plugin capture and returns the result.
// POST /api/plugins/{id}/run
func (s *Server) handlePluginRun(w http.ResponseWriter, r *http.Request) {
	jsonError(w, "manual HTTP plugin execution is disabled; enabled plugins may run only through the local polling agent", http.StatusGone)
}

// handlePluginConfig saves plugin configuration values.
// PUT /api/plugins/{id}/config
func (s *Server) handlePluginConfig(w http.ResponseWriter, r *http.Request) {
	pluginID := r.PathValue("id")
	if pluginID == "" {
		jsonError(w, "plugin id required", http.StatusBadRequest)
		return
	}

	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	target, errs := s.findConfiguredPlugin(pluginID)
	for _, err := range errs {
		s.logger.Warn("Plugin discovery error", "error", err)
	}
	if target == nil {
		jsonError(w, "plugin not found", http.StatusNotFound)
		return
	}

	prospectiveEnabled := target.Enabled
	if enabled, ok := body["enabled"]; ok {
		switch {
		case strings.EqualFold(enabled, "true"):
			prospectiveEnabled = true
		case strings.EqualFold(enabled, "false"):
			prospectiveEnabled = false
		default:
			jsonError(w, "enabled must be true or false", http.StatusBadRequest)
			return
		}
	}

	if prospectiveEnabled {
		if missing := missingRequiredPluginConfig(target, body); len(missing) > 0 {
			jsonError(w, fmt.Sprintf("missing required plugin config: %s", strings.Join(missing, ", ")), http.StatusBadRequest)
			return
		}
	}

	for key, val := range body {
		if key == "enabled" {
			continue
		}
		if _, exists := target.Manifest.Config[key]; !exists {
			continue
		}
		if _, err := s.store.SetConfig("plugin_"+pluginID+"_"+key, val); err != nil {
			jsonError(w, fmt.Sprintf("failed to save plugin config %q", key), http.StatusInternalServerError)
			return
		}
		target.Config[key] = val
	}

	if enabled, ok := body["enabled"]; ok {
		if _, err := s.store.SetConfig("plugin_"+pluginID+"_enabled", strings.ToLower(enabled)); err != nil {
			jsonError(w, "failed to save plugin enabled state", http.StatusInternalServerError)
			return
		}
		target.Enabled = prospectiveEnabled
	}

	if err := s.ensurePluginDataSource(target); err != nil {
		jsonError(w, "failed to update plugin data source", http.StatusInternalServerError)
		return
	}
	s.refreshRuntimePlugins()

	writeJSON(w, map[string]any{
		"status":  "ok",
		"enabled": target.Enabled,
	})
}

func missingRequiredPluginConfig(p *plugin.Plugin, overrides map[string]string) []string {
	if p == nil {
		return nil
	}

	config := make(map[string]string, len(p.Config)+len(overrides))
	for key, val := range p.Config {
		config[key] = val
	}
	for key, val := range overrides {
		if key == "enabled" {
			continue
		}
		if _, exists := p.Manifest.Config[key]; exists {
			config[key] = val
		}
	}

	var missing []string
	for key, field := range p.Manifest.Config {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(config[key]) == "" {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}
