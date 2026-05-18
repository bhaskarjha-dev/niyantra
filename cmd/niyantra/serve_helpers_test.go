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

func TestValidateServeExposure(t *testing.T) {
	tests := []struct {
		name           string
		bind           string
		authEnabled    bool
		allowRemote    bool
		httpMCPEnabled bool
		wantErr        bool
	}{
		{name: "localhost default", bind: "127.0.0.1", wantErr: false},
		{name: "remote bind requires opt in", bind: "0.0.0.0", wantErr: true},
		{name: "remote dashboard allowed with explicit opt in", bind: "0.0.0.0", allowRemote: true, wantErr: false},
		{name: "remote http mcp requires auth", bind: "0.0.0.0", allowRemote: true, httpMCPEnabled: true, wantErr: true},
		{name: "remote http mcp with auth", bind: "0.0.0.0", allowRemote: true, authEnabled: true, httpMCPEnabled: true, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServeExposure(tt.bind, tt.authEnabled, tt.allowRemote, tt.httpMCPEnabled)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateServeExposure(%q, %v, %v, %v) err = %v, wantErr %v", tt.bind, tt.authEnabled, tt.allowRemote, tt.httpMCPEnabled, err, tt.wantErr)
			}
		})
	}
}

func TestShouldWarnInsecureBind(t *testing.T) {
	tests := []struct {
		name        string
		bind        string
		authEnabled bool
		allowRemote bool
		want        bool
	}{
		{name: "localhost no auth", bind: "127.0.0.1", authEnabled: false, want: false},
		{name: "localhost named no auth", bind: "localhost", authEnabled: false, want: false},
		{name: "remote bind without explicit opt in", bind: "0.0.0.0", authEnabled: false, want: false},
		{name: "remote bind no auth", bind: "0.0.0.0", authEnabled: false, allowRemote: true, want: true},
		{name: "remote bind with auth", bind: "0.0.0.0", authEnabled: true, allowRemote: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldWarnInsecureBind(tt.bind, tt.authEnabled, tt.allowRemote)
			if got != tt.want {
				t.Fatalf("shouldWarnInsecureBind(%q, %v, %v) = %v, want %v", tt.bind, tt.authEnabled, tt.allowRemote, got, tt.want)
			}
		})
	}
}
