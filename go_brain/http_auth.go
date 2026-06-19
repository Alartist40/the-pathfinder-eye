// http_auth.go — shared-secret bearer-token gate for the brain's
// HTTP API. Why: the brain exposes /move, /camera, /ai/think,
// /stream on :8080 with no auth at all — anyone on the LAN can drive
// the motors, pivot the camera, or run arbitrary LLM prompts. Cynapse
// uses confirm-gates and reds secret content for similar reasons in
// its shell-tool path. The robot doesn't have a TUI to approve
// confirmations in, so a static bearer token is the next-best thing.
//
// Behavior:
//   - When PATHFINDER_EYE_HTTP_TOKEN is set, every guarded request
//     must carry `Authorization: Bearer ***. Cloud tokens (sk-…,
//     gh[pousr]_, etc.) are rejected for the auth header itself.
//   - When unset, requests are accepted only from loopback hosts
//     (127.0.0.1, ::1). LAN traffic is refused with 403.
//   - All unguarded paths (including /health) remain open.
package main

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// authPathFlags marks which routes require auth even on loopback.
// /health is excluded so heartbeat/dashboards still work freely.
var authPathFlags = map[string]bool{
	"/move":     true,
	"/camera":   true,
	"/ai/think": true,
	"/stream":   true,
}

// authorized reports whether r is allowed to reach its handler.
func authorized(r *http.Request) bool {
	if !authPathFlags[r.URL.Path] {
		// Unguarded route — pass-through.
		return true
	}
	if envToken := os.Getenv("PATHFINDER_EYE_HTTP_TOKEN"); envToken != "" {
		got := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(got, "Bearer ") {
			return false
		}
		// Defense in depth: refuse cloud-shaped tokens in the auth
		// header so they don't accidentally land in logs that the
		// redact pass might miss.
		token := strings.TrimSpace(got[len("Bearer "):])
		if redactOnce(token) != token {
			// Redact altered the token — it's a known-secret shape.
			return false
		}
		return strings.EqualFold(token, envToken)
	}
	// No bearer token configured — only loopback.
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// authWrap returns the wrapped handler that enforces authorized()
// before calling h.
func authWrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}
