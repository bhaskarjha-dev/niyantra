package main

import (
	"fmt"

	"github.com/bhaskarjha-com/niyantra/internal/web"
)

func validateServeAuthConfig(auth string) (bool, error) {
	if auth == "" {
		return false, nil
	}
	if err := web.ValidateBasicAuthConfig(auth); err != nil {
		return false, err
	}
	return true, nil
}

func displayDashboardAddress(bind string, port int) string {
	if bind == "" || bind == "127.0.0.1" || bind == "localhost" {
		return fmt.Sprintf("http://localhost:%d", port)
	}
	return fmt.Sprintf("http://%s:%d", bind, port)
}

func shouldWarnInsecureBind(bind string, authEnabled bool) bool {
	if authEnabled {
		return false
	}
	return bind != "127.0.0.1" && bind != "localhost"
}
