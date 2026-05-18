package main

import (
	"fmt"
	"net"

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

func isLocalBind(bind string) bool {
	if bind == "" || bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}

func displayDashboardAddress(bind string, port int) string {
	if isLocalBind(bind) {
		return fmt.Sprintf("http://localhost:%d", port)
	}
	return fmt.Sprintf("http://%s:%d", bind, port)
}

func validateServeExposure(bind string, authEnabled, allowRemote, httpMCPEnabled bool) error {
	if isLocalBind(bind) {
		return nil
	}
	if !allowRemote {
		return fmt.Errorf("refusing non-local bind %q without --allow-remote / NIYANTRA_ALLOW_REMOTE=true", bind)
	}
	if httpMCPEnabled && !authEnabled {
		return fmt.Errorf("refusing HTTP MCP on non-local bind %q without --auth / NIYANTRA_AUTH", bind)
	}
	return nil
}

func shouldWarnInsecureBind(bind string, authEnabled, allowRemote bool) bool {
	if !allowRemote || authEnabled {
		return false
	}
	return !isLocalBind(bind)
}
