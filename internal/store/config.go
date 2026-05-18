package store

import (
	"fmt"
	"strconv"
	"strings"
)

// ConfigEntry represents a server-level configuration entry.
type ConfigEntry struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	Category    string `json:"category"`
	Label       string `json:"label"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updatedAt"`
}

// GetConfig returns a single config value as string.
func (s *Store) GetConfig(key string) string {
	return s.resolveConfigValue(key, s.getConfigRaw(key))
}

// GetConfigInt returns a config value as int with a default fallback.
func (s *Store) GetConfigInt(key string, defaultVal int) int {
	val := s.GetConfig(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// GetConfigFloat returns a config value as float64 with a default fallback.
func (s *Store) GetConfigFloat(key string, defaultVal float64) float64 {
	val := s.GetConfig(key)
	if val == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal
	}
	return f
}

// GetConfigBool returns a config value as bool.
func (s *Store) GetConfigBool(key string) bool {
	return s.GetConfig(key) == "true"
}

// SetConfig updates a config value and returns the old value.
func (s *Store) SetConfig(key, value string) (string, error) {
	oldVal := s.GetConfig(key)
	valueType, category, label := inferConfigMetadata(key, value)
	storedValue := value
	existingRaw := s.getConfigRaw(key)

	if IsSensitiveConfigKey(key) {
		if value == "" {
			s.deleteStoredSecretRef(existingRaw)
			storedValue = ""
		} else {
			secured, err := s.secureValueForStorage(key, value, existingRaw)
			if err != nil {
				return "", err
			}
			storedValue = secured
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO config (key, value, value_type, category, label, description, updated_at)
		VALUES (?, ?, ?, ?, ?, '', datetime('now'))
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = datetime('now')
	`, key, storedValue, valueType, category, label)
	if err != nil {
		return "", fmt.Errorf("store: update config %s: %w", key, err)
	}

	return oldVal, nil
}

func inferConfigMetadata(key, value string) (valueType, category, label string) {
	valueType = "string"
	if value == "true" || value == "false" {
		valueType = "bool"
	} else if _, err := strconv.Atoi(value); err == nil {
		valueType = "int"
	} else if _, err := strconv.ParseFloat(value, 64); err == nil {
		valueType = "float"
	}

	category = "general"
	if strings.HasPrefix(key, "plugin_") {
		category = "plugins"
	}

	label = strings.ReplaceAll(key, "_", " ")
	if label == "" {
		label = key
	}
	return valueType, category, label
}

// AllConfig returns all config entries, optionally filtered by category.
func (s *Store) AllConfig(category string) ([]*ConfigEntry, error) {
	query := `SELECT key, value, value_type, category, label, COALESCE(description,''), updated_at FROM config`
	args := []interface{}{}

	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY category, key`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: query config: %w", err)
	}
	defer rows.Close()

	var entries []*ConfigEntry
	for rows.Next() {
		e := &ConfigEntry{}
		if err := rows.Scan(&e.Key, &e.Value, &e.ValueType, &e.Category, &e.Label, &e.Description, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Value = s.resolveConfigValue(e.Key, e.Value)
		entries = append(entries, e)
	}
	return entries, nil
}

// ConfigMap returns all config as a key→value map for fast lookups.
func (s *Store) ConfigMap() map[string]string {
	m := make(map[string]string)
	rows, err := s.db.Query(`SELECT key, value FROM config`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			m[k] = s.resolveConfigValue(k, v)
		}
	}
	return m
}

// ConfigTypeMap returns all config as a key→valueType map for type validation.
func (s *Store) ConfigTypeMap() map[string]string {
	m := make(map[string]string)
	rows, err := s.db.Query(`SELECT key, value_type FROM config`)
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var k, vt string
		if err := rows.Scan(&k, &vt); err == nil {
			m[k] = vt
		}
	}
	return m
}
