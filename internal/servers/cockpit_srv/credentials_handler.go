package cockpit_srv

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type CredentialsResponse struct {
	Creds string `json:"creds"`
}

type credentialsRequest struct {
	Password string `json:"password"`
}

// credentialsHandler handles POST /api/v1/credentials?context=<name>
// Returns the raw .creds file content for the named context.
// If the context has a password configured, the POST body must contain { "password": "..." }.
func (s *CockpitServer) credentialsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contextName := r.URL.Query().Get("context")
	if contextName == "" {
		contextName = s.opts.Config.CurrentContext
	}

	var credPath, password string
	found := false
	for _, ctx := range s.opts.Config.Contexts {
		if ctx.Name == contextName {
			credPath = ctx.Credentials
			password = ctx.Password
			found = true
			break
		}
	}

	if !found {
		http.Error(w, fmt.Sprintf("context %q not found", contextName), http.StatusNotFound)
		return
	}

	if password != "" {
		var req credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if subtle.ConstantTimeCompare([]byte(req.Password), []byte(password)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if credPath == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		s.log.Error().Err(err).Str("path", credPath).Msg("failed to read credentials file")
		http.Error(w, "failed to read credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(CredentialsResponse{Creds: string(data)}); err != nil {
		s.log.Error().Err(err).Msg("failed to encode credentials response")
	}
}
