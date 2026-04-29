package cockpit_srv

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ConfigResponse represents the public-facing API response for configuration
type ConfigResponse struct {
	CurrentContext string            `json:"currentContext"`
	Contexts       []ContextResponse `json:"contexts"`
}

// ContextResponse represents a sanitized context without sensitive fields.
// Target is always a WebSocket URL (ws:// or wss://) ready for browser use.
// Excludes: Credentials, RootCA, TlsCert, TlsKey, Password
type ContextResponse struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Grafana string `json:"grafana"`
	Secured bool   `json:"secured"`
}

// toWebSocketTarget returns a WebSocket URL (ws:// or wss://) for a context.
// Uses webTarget if set; otherwise falls back to natsTarget with port 9222.
// In both cases, nats:// → ws://, tls:// → wss://, nats+tls:// → wss://.
func toWebSocketTarget(natsTarget, webTarget string) string {
	if webTarget != "" {
		scheme, host, ok := parseNatsURL(webTarget)
		if !ok {
			return webTarget // already ws:// or wss:// — use as-is
		}
		return scheme + "://" + host
	}
	scheme, host, _ := parseNatsURL(natsTarget)
	if i := strings.LastIndex(host, ":"); i != -1 {
		host = host[:i]
	}
	return scheme + "://" + host + ":9222"
}

// parseNatsURL extracts the WebSocket scheme and host from a NATS URL.
// Returns (wsScheme, hostWithPort, true) when the URL uses a nats:// variant,
// or ("", "", false) when it is already a WebSocket URL.
func parseNatsURL(url string) (scheme, host string, isNats bool) {
	switch {
	case strings.HasPrefix(url, "nats+tls://"):
		return "wss", strings.TrimPrefix(url, "nats+tls://"), true
	case strings.HasPrefix(url, "tls://"):
		return "wss", strings.TrimPrefix(url, "tls://"), true
	case strings.HasPrefix(url, "nats://"):
		return "ws", strings.TrimPrefix(url, "nats://"), true
	default:
		return "", "", false
	}
}

// configHandler handles GET /api/v1/contexts requests
// Returns the cockpit configuration with sensitive fields filtered out
func (s *CockpitServer) configHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := ConfigResponse{
		CurrentContext: s.opts.Config.CurrentContext,
		Contexts:       make([]ContextResponse, len(s.opts.Config.Contexts)),
	}

	for i, ctx := range s.opts.Config.Contexts {
		response.Contexts[i] = ContextResponse{
			Name:    ctx.Name,
			Target:  toWebSocketTarget(ctx.Target, ctx.WebTarget),
			Grafana: ctx.Grafana,
			Secured: ctx.Password != "",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error().Err(err).Msg("failed to encode config response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
