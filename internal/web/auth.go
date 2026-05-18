package web

import (
	"fmt"
	"strings"
)

// ValidateBasicAuthConfig verifies that auth is either empty or a non-empty
// user:pass pair. Passwords may contain additional colons.
func ValidateBasicAuthConfig(auth string) error {
	_, _, _, err := parseBasicAuthConfig(auth)
	return err
}

func parseBasicAuthConfig(auth string) (user, pass string, enabled bool, err error) {
	if auth == "" {
		return "", "", false, nil
	}

	parts := strings.SplitN(auth, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false, fmt.Errorf("invalid auth value %q: expected non-empty user:pass", auth)
	}

	return parts[0], parts[1], true, nil
}
