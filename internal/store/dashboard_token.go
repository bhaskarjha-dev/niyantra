package store

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const DashboardTokenConfigKey = "dashboard_api_token"

// GenerateDashboardToken returns a cryptographically random token suitable for
// authenticating local dashboard API and MCP requests.
func GenerateDashboardToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("store: generate dashboard token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

// EnsureDashboardToken returns the existing dashboard token or creates one.
func (s *Store) EnsureDashboardToken() (token string, created bool, err error) {
	if token := s.GetConfig(DashboardTokenConfigKey); token != "" {
		return token, false, nil
	}
	token, err = GenerateDashboardToken()
	if err != nil {
		return "", false, err
	}
	if _, err := s.SetConfig(DashboardTokenConfigKey, token); err != nil {
		return "", false, err
	}
	return token, true, nil
}

// RotateDashboardToken replaces the dashboard API token.
func (s *Store) RotateDashboardToken() (string, error) {
	token, err := GenerateDashboardToken()
	if err != nil {
		return "", err
	}
	if _, err := s.SetConfig(DashboardTokenConfigKey, token); err != nil {
		return "", err
	}
	return token, nil
}
