package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	keyring "github.com/zalando/go-keyring"
)

const (
	keyringServiceName = "niyantra"
	secretRefPrefix    = "secret://v1/"
)

// SecretBackend persists secrets outside SQLite.
type SecretBackend interface {
	Set(ref, value string) error
	Get(ref string) (string, error)
	Delete(ref string) error
}

type keyringSecretBackend struct{}

func (keyringSecretBackend) Set(ref, value string) error {
	return keyring.Set(keyringServiceName, ref, value)
}

func (keyringSecretBackend) Get(ref string) (string, error) {
	return keyring.Get(keyringServiceName, ref)
}

func (keyringSecretBackend) Delete(ref string) error {
	err := keyring.Delete(keyringServiceName, ref)
	if err != nil && errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// MemorySecretBackend is an in-memory secret backend for tests.
type MemorySecretBackend struct {
	mu     sync.Mutex
	values map[string]string
}

// NewMemorySecretBackend creates an in-memory secret backend for tests.
func NewMemorySecretBackend() SecretBackend {
	return &MemorySecretBackend{values: make(map[string]string)}
}

func (m *MemorySecretBackend) Set(ref, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[ref] = value
	return nil
}

func (m *MemorySecretBackend) Get(ref string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	val, ok := m.values[ref]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return val, nil
}

func (m *MemorySecretBackend) Delete(ref string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, ref)
	return nil
}

// OpenOption customizes Store.Open behavior.
type OpenOption func(*Store)

// WithSecretBackend overrides the default secret backend.
func WithSecretBackend(backend SecretBackend) OpenOption {
	return func(s *Store) {
		s.secrets = backend
	}
}

// WithInsecurePlaintextSecrets allows sensitive config values to remain in
// plaintext SQLite rows when no secure secret backend is available.
func WithInsecurePlaintextSecrets(allowed bool) OpenOption {
	return func(s *Store) {
		s.allowPlaintextSecrets = allowed
	}
}

// IsSensitiveConfigKey returns true when a config key should never be stored
// in plaintext SQLite unless the insecure fallback is explicitly enabled.
func IsSensitiveConfigKey(key string) bool {
	switch key {
	case "copilot_pat",
		"cursor_session_token",
		"gemini_client_secret",
		"smtp_pass",
		"smtp_user",
		"webhook_secret",
		"webpush_vapid_private":
		return true
	}

	if strings.HasPrefix(key, "plugin_") {
		lk := strings.ToLower(key)
		for _, pattern := range []string{"_api_key", "_token", "_secret", "_password", "_pat", "_credential"} {
			if strings.HasSuffix(lk, pattern) {
				return true
			}
		}
	}

	return false
}

func newDefaultSecretBackend() SecretBackend {
	return keyringSecretBackend{}
}

func isSecretRef(value string) bool {
	return strings.HasPrefix(value, secretRefPrefix)
}

func secretRefID(value string) string {
	return strings.TrimPrefix(value, secretRefPrefix)
}

func newSecretRef(key string) (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate secret ref: %w", err)
	}
	return secretRefPrefix + key + "/" + hex.EncodeToString(buf[:]), nil
}

func (s *Store) getConfigRaw(key string) string {
	var val string
	err := s.db.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

func (s *Store) secureValueForStorage(key, plainValue, existingRaw string) (string, error) {
	ref := existingRaw
	if !isSecretRef(ref) {
		var err error
		ref, err = newSecretRef(key)
		if err != nil {
			return "", err
		}
	}

	if s.secrets != nil {
		if err := s.secrets.Set(secretRefID(ref), plainValue); err == nil {
			return ref, nil
		} else if !s.allowPlaintextSecrets {
			return "", fmt.Errorf("secure secret storage unavailable for %s: %w", key, err)
		}
	}

	if !s.allowPlaintextSecrets {
		return "", fmt.Errorf("secure secret storage unavailable for %s; rerun with --insecure-plaintext-secrets only if you accept plaintext SQLite secret storage", key)
	}

	return plainValue, nil
}

func (s *Store) resolveConfigValue(key, raw string) string {
	if raw == "" || !IsSensitiveConfigKey(key) {
		return raw
	}
	if !isSecretRef(raw) {
		return raw
	}
	if s.secrets == nil {
		return ""
	}
	value, err := s.secrets.Get(secretRefID(raw))
	if err != nil {
		return ""
	}
	return value
}

func (s *Store) deleteStoredSecretRef(raw string) {
	if s.secrets == nil || !isSecretRef(raw) {
		return
	}
	_ = s.secrets.Delete(secretRefID(raw))
}

func (s *Store) migrateSensitiveConfigSecrets() error {
	rows, err := s.db.Query(`SELECT key, value FROM config ORDER BY key`)
	if err != nil {
		return fmt.Errorf("store: query secrets for migration: %w", err)
	}
	defer rows.Close()

	type entry struct {
		key   string
		value string
	}
	var pending []entry
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("store: scan secret migration rows: %w", err)
		}
		if !IsSensitiveConfigKey(key) || value == "" || isSecretRef(value) {
			continue
		}
		pending = append(pending, entry{key: key, value: value})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate secret migration rows: %w", err)
	}

	for _, item := range pending {
		storedValue, err := s.secureValueForStorage(item.key, item.value, "")
		if err != nil {
			return err
		}
		if storedValue == item.value {
			continue
		}
		if _, err := s.db.Exec(`UPDATE config SET value = ?, updated_at = datetime('now') WHERE key = ?`, storedValue, item.key); err != nil {
			return fmt.Errorf("store: migrate secret %s: %w", item.key, err)
		}
	}

	return nil
}
