package web

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func hasRequestBody(r *http.Request) bool {
	return r.ContentLength > 0 || len(r.TransferEncoding) > 0
}

// basicAuth wraps a handler with HTTP basic authentication.
func (s *Server) basicAuth(next http.Handler) http.Handler {
	user, pass, enabled, err := parseBasicAuthConfig(s.auth)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "server authentication is misconfigured",
			})
		})
	}
	if !enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check only
		// /healthz: container probes need unauthenticated access
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Niyantra"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// securityMiddleware enforces CORS, Content-Type, and browser security policies.
func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ── Security Headers ──────────────────────────────────────────
		// Set on ALL responses to protect against XSS, clickjacking,
		// MIME sniffing, and unauthorized browser feature access.
		// Critical for cloud PWA deployment where the dashboard is public.

		// Content-Security-Policy: restrict script/style/connect sources.
		// 'self' = same-origin only. 'unsafe-inline' required for style attributes
		// used by Chart.js canvas rendering and inline component styles.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; connect-src 'self'; font-src 'self'; "+
				"object-src 'none'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// Prevent clickjacking by disallowing framing from any origin.
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing — browser must respect Content-Type.
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Limit Referer header leakage to same-origin only.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Disable unnecessary browser features (camera, mic, geolocation, etc.).
		w.Header().Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()")

		// ── CORS ──────────────────────────────────────────────────────
		// Only allow localhost origin matching our port.
		allowedOrigin := fmt.Sprintf("http://localhost:%d", s.port)
		allowedOrigin2 := fmt.Sprintf("http://127.0.0.1:%d", s.port)
		origin := r.Header.Get("Origin")

		sameOrigin := origin == allowedOrigin || origin == allowedOrigin2
		crossOrigin := origin != "" && !sameOrigin

		// MCP endpoint: enforce strict Origin check to prevent cross-site exfiltration.
		// Requests without Origin header are allowed (CLI tools, MCP SDK clients).
		if r.URL.Path == "/mcp" && crossOrigin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "cross-origin MCP requests are not allowed",
			})
			return
		}

		// REST API endpoints reject ALL browser cross-origin requests (reads
		// AND mutations). CORS alone does not protect GET endpoints: a foreign
		// page can fetch('/api/export/json') and, even if the browser hides the
		// response from JS, certain attack vectors (DNS-rebinding, service
		// workers, img/form-based exfiltration) can still capture data.
		// CLI tools and MCP SDK clients send no Origin header, so they are
		// unaffected by this gate.
		if strings.HasPrefix(r.URL.Path, "/api/") && crossOrigin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "cross-origin API requests are not allowed",
			})
			return
		}

		if sameOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			if crossOrigin && (strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "cross-origin requests are not allowed",
				})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// ── Content-Type Enforcement ──────────────────────────────────
		// Enforce Content-Type: application/json on mutation endpoints.
		// Skip for /mcp — the MCP SDK handler manages its own content negotiation.
		if isUnsafeMethod(r.Method) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				ct := r.Header.Get("Content-Type")
				if hasRequestBody(r) && !strings.HasPrefix(ct, "application/json") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnsupportedMediaType)
					json.NewEncoder(w).Encode(map[string]string{"error": "Content-Type must be application/json"})
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
