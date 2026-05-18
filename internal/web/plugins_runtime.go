package web

import (
	"github.com/bhaskarjha-com/niyantra/internal/plugin"
	"github.com/bhaskarjha-com/niyantra/internal/store"
)

func (s *Server) loadConfiguredPlugins() ([]*plugin.Plugin, []error) {
	pluginsDir := plugin.DefaultPluginsDir()
	plugins, errs := plugin.Discover(pluginsDir)
	for _, p := range plugins {
		s.applyStoredPluginState(p)
	}
	return plugins, errs
}

func (s *Server) findConfiguredPlugin(pluginID string) (*plugin.Plugin, []error) {
	plugins, errs := s.loadConfiguredPlugins()
	for _, p := range plugins {
		if p.Manifest.ID == pluginID {
			return p, errs
		}
	}
	return nil, errs
}

func (s *Server) refreshRuntimePlugins() ([]*plugin.Plugin, []error) {
	plugins, errs := s.loadConfiguredPlugins()
	if ag := s.agentMgr.Agent(); ag != nil {
		ag.SetPlugins(plugins)
	}
	return plugins, errs
}

func (s *Server) applyStoredPluginState(p *plugin.Plugin) {
	if p.Config == nil {
		p.Config = make(map[string]string)
	}

	p.Enabled = s.store.GetConfigBool("plugin_" + p.Manifest.ID + "_enabled")
	for key, field := range p.Manifest.Config {
		val := s.store.GetConfig("plugin_" + p.Manifest.ID + "_" + key)
		if val == "" && field.Default != "" {
			val = field.Default
		}
		if val != "" {
			p.Config[key] = val
		}
	}

	s.ensurePluginDataSource(p)
}

func (s *Server) pluginSourceIndex() map[string]*store.DataSource {
	sources, err := s.store.AllDataSources()
	if err != nil {
		s.logger.Warn("Failed to load data sources", "error", err)
		return map[string]*store.DataSource{}
	}

	index := make(map[string]*store.DataSource, len(sources))
	for _, ds := range sources {
		index[ds.ID] = ds
	}
	return index
}

func (s *Server) ensurePluginDataSource(p *plugin.Plugin) {
	sourceID := "plugin_" + p.Manifest.ID
	if err := s.store.ExecRaw(`
		INSERT OR IGNORE INTO data_sources (id, name, source_type, enabled, config_json)
		VALUES (?, ?, 'plugin', ?, '{}')
	`, sourceID, p.Manifest.Name, boolToSQLite(p.Enabled)); err != nil {
		s.logger.Warn("Failed to create plugin data source", "plugin", p.Manifest.ID, "error", err)
		return
	}

	if err := s.store.ExecRaw(`
		UPDATE data_sources
		SET name = ?, source_type = 'plugin', enabled = ?
		WHERE id = ?
	`, p.Manifest.Name, boolToSQLite(p.Enabled), sourceID); err != nil {
		s.logger.Warn("Failed to update plugin data source", "plugin", p.Manifest.ID, "error", err)
	}
}

func boolToSQLite(v bool) int {
	if v {
		return 1
	}
	return 0
}
