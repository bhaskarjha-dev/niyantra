package main

import "testing"

func TestValidateServeAuthConfig(t *testing.T) {
	tests := []struct {
		name    string
		auth    string
		enabled bool
		wantErr bool
	}{
		{name: "disabled", auth: "", enabled: false, wantErr: false},
		{name: "valid", auth: "admin:s3cure-pass", enabled: true, wantErr: false},
		{name: "missing colon", auth: "admin", enabled: false, wantErr: true},
		{name: "missing user", auth: ":secret", enabled: false, wantErr: true},
		{name: "missing pass", auth: "admin:", enabled: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnabled, err := validateServeAuthConfig(tt.auth)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateServeAuthConfig(%q) error = %v, wantErr %v", tt.auth, err, tt.wantErr)
			}
			if gotEnabled != tt.enabled {
				t.Fatalf("validateServeAuthConfig(%q) enabled = %v, want %v", tt.auth, gotEnabled, tt.enabled)
			}
		})
	}
}

func TestDisplayDashboardAddress(t *testing.T) {
	tests := []struct {
		name string
		bind string
		port int
		want string
	}{
		{name: "localhost default", bind: "127.0.0.1", port: 9222, want: "http://localhost:9222"},
		{name: "localhost name", bind: "localhost", port: 9333, want: "http://localhost:9333"},
		{name: "empty bind", bind: "", port: 9444, want: "http://localhost:9444"},
		{name: "lan bind", bind: "0.0.0.0", port: 9555, want: "http://0.0.0.0:9555"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := displayDashboardAddress(tt.bind, tt.port)
			if got != tt.want {
				t.Fatalf("displayDashboardAddress(%q, %d) = %q, want %q", tt.bind, tt.port, got, tt.want)
			}
		})
	}
}

func TestShouldWarnInsecureBind(t *testing.T) {
	tests := []struct {
		name        string
		bind        string
		authEnabled bool
		want        bool
	}{
		{name: "localhost no auth", bind: "127.0.0.1", authEnabled: false, want: false},
		{name: "localhost named no auth", bind: "localhost", authEnabled: false, want: false},
		{name: "lan no auth", bind: "0.0.0.0", authEnabled: false, want: true},
		{name: "lan with auth", bind: "0.0.0.0", authEnabled: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldWarnInsecureBind(tt.bind, tt.authEnabled)
			if got != tt.want {
				t.Fatalf("shouldWarnInsecureBind(%q, %v) = %v, want %v", tt.bind, tt.authEnabled, got, tt.want)
			}
		})
	}
}
