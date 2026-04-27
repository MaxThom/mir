package cockpit_srv

import (
	"encoding/json"
	"net/http"
)

// ConfigResponse represents the public-facing API response for configuration
type ConfigResponse struct {
	CurrentContext string            `json:"currentContext"`
	Contexts       []ContextResponse `json:"contexts"`
}

// ContextResponse represents a sanitized context without sensitive fields
// Excludes: Credentials, RootCA, TlsCert, TlsKey, Password
type ContextResponse struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	WebTarget string `json:"webTarget,omitempty"`
	Grafana   string `json:"grafana"`
	Secured   bool   `json:"secured"`
}

// configHandler handles GET /api/config requests
// Returns the cockpit configuration with sensitive fields filtered out
func (s *CockpitServer) configHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build sanitized response
	response := ConfigResponse{
		CurrentContext: s.opts.Config.CurrentContext,
		Contexts:       make([]ContextResponse, len(s.opts.Config.Contexts)),
	}

	// Map contexts to response DTOs, filtering out sensitive fields
	for i, ctx := range s.opts.Config.Contexts {
		response.Contexts[i] = ContextResponse{
			Name:      ctx.Name,
			Target:    ctx.Target,
			WebTarget: ctx.WebTarget,
			Grafana:   ctx.Grafana,
			Secured:   ctx.Password != "",
		}
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error().Err(err).Msg("failed to encode config response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
