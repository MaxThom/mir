package cockpit_srv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

func TestConfigHandler_Success(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "production",
				Contexts: []ui.Context{
					{
						Name:    "local",
						Target:  "nats://localhost:4222",
						Grafana: "localhost:3000",
						Sec: ui.ContextSecurity{
							Credentials: "/path/to/creds",
							RootCA:      "/path/to/ca",
							TlsCert:     "/path/to/cert",
							TlsKey:      "/path/to/key",
						},
					},
					{
						Name:    "production",
						Target:  "nats://prod.example.com:4222",
						Grafana: "grafana.example.com",
					},
				},
			},
		},
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	// Execute
	server.configHandler(w, req)

	// Assert response code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Assert content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Parse response
	var response ConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Assert current context
	if response.CurrentContext != "production" {
		t.Errorf("expected currentContext 'production', got '%s'", response.CurrentContext)
	}

	// Assert contexts count
	if len(response.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(response.Contexts))
	}

	// Assert first context
	if response.Contexts[0].Name != "local" {
		t.Errorf("expected first context name 'local', got '%s'", response.Contexts[0].Name)
	}
	if response.Contexts[0].Target != "ws://localhost:9222" {
		t.Errorf("expected first context target 'ws://localhost:9222', got '%s'", response.Contexts[0].Target)
	}
	if response.Contexts[0].Grafana != "localhost:3000" {
		t.Errorf("expected first context grafana 'localhost:3000', got '%s'", response.Contexts[0].Grafana)
	}
}

func TestConfigHandler_SensitiveFieldsFiltered(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts: []ui.Context{
					{
						Name:    "local",
						Target:  "nats://localhost:4222",
						Grafana: "localhost:3000",
						Sec: ui.ContextSecurity{
							Credentials: "/secret/credentials",
							RootCA:      "/secret/ca.pem",
							TlsCert:     "/secret/cert.pem",
							TlsKey:      "/secret/key.pem",
						},
					},
				},
			},
		},
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	// Execute
	server.configHandler(w, req)

	// Parse raw JSON to check for sensitive fields
	var rawResponse map[string]any
	if err := json.NewDecoder(w.Body).Decode(&rawResponse); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check contexts array
	contexts, ok := rawResponse["contexts"].([]any)
	if !ok || len(contexts) == 0 {
		t.Fatal("expected contexts array in response")
	}

	// Check first context
	firstContext, ok := contexts[0].(map[string]any)
	if !ok {
		t.Fatal("expected context to be an object")
	}

	// Assert sensitive fields are NOT present
	sensitiveFields := []string{"credentials", "rootCA", "tlsCert", "tlsKey"}
	for _, field := range sensitiveFields {
		if _, exists := firstContext[field]; exists {
			t.Errorf("sensitive field '%s' should not be present in response", field)
		}
	}

	// Assert public fields ARE present
	publicFields := []string{"name", "target", "grafana"}
	for _, field := range publicFields {
		if _, exists := firstContext[field]; !exists {
			t.Errorf("public field '%s' should be present in response", field)
		}
	}
}

func TestConfigHandler_MethodNotAllowed(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{},
			},
		},
	}

	// Test various HTTP methods
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/config", nil)
			w := httptest.NewRecorder()

			server.configHandler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestConfigHandler_EmptyContexts(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "",
				Contexts:       []ui.Context{},
			},
		},
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	// Execute
	server.configHandler(w, req)

	// Assert response code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var response ConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Assert empty contexts
	if len(response.Contexts) != 0 {
		t.Errorf("expected 0 contexts, got %d", len(response.Contexts))
	}

	// Assert empty current context
	if response.CurrentContext != "" {
		t.Errorf("expected empty currentContext, got '%s'", response.CurrentContext)
	}
}

func TestConfigHandler_MultipleContexts(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "staging",
				Contexts: []ui.Context{
					{
						Name:    "local",
						Target:  "nats://localhost:4222",
						Grafana: "localhost:3000",
					},
					{
						Name:    "staging",
						Target:  "nats://staging.example.com:4222",
						Grafana: "staging-grafana.example.com",
					},
					{
						Name:    "production",
						Target:  "nats://prod.example.com:4222",
						Grafana: "prod-grafana.example.com",
					},
				},
			},
		},
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	// Execute
	server.configHandler(w, req)

	// Assert response code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Parse response
	var response ConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Assert current context
	if response.CurrentContext != "staging" {
		t.Errorf("expected currentContext 'staging', got '%s'", response.CurrentContext)
	}

	// Assert all contexts are present
	if len(response.Contexts) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(response.Contexts))
	}

	// Assert context order is preserved
	expectedNames := []string{"local", "staging", "production"}
	for i, ctx := range response.Contexts {
		if ctx.Name != expectedNames[i] {
			t.Errorf("expected context[%d].name '%s', got '%s'", i, expectedNames[i], ctx.Name)
		}
	}
}

func TestToWebSocketTarget(t *testing.T) {
	cases := []struct {
		nats      string
		webTarget string
		want      string
	}{
		// Derived from natsTarget (no webTarget)
		{"nats://localhost:4222", "", "ws://localhost:9222"},
		{"nats://prod.example.com:4222", "", "ws://prod.example.com:9222"},
		{"nats://host:1234", "", "ws://host:9222"},
		{"tls://host:4222", "", "wss://host:9222"},
		{"nats+tls://host:4222", "", "wss://host:9222"},
		// webTarget already a WebSocket URL — used as-is
		{"nats://host:4222", "wss://host:9222", "wss://host:9222"},
		{"nats://host:4222", "ws://custom:8080", "ws://custom:8080"},
		// webTarget uses a nats:// variant — converted
		{"nats://host:4222", "nats://myhost:9222", "ws://myhost:9222"},
		{"nats://host:4222", "tls://myhost:9222", "wss://myhost:9222"},
		{"nats://host:4222", "nats+tls://myhost:9222", "wss://myhost:9222"},
	}
	for _, c := range cases {
		got := toWebSocketTarget(c.nats, c.webTarget)
		if got != c.want {
			t.Errorf("toWebSocketTarget(%q, %q) = %q, want %q", c.nats, c.webTarget, got, c.want)
		}
	}
}

func TestConfigHandler_JSONStructure(t *testing.T) {
	// Setup
	log := zerolog.Nop()
	server := &CockpitServer{
		log: log,
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts: []ui.Context{
					{
						Name:    "local",
						Target:  "nats://localhost:4222",
						Grafana: "localhost:3000",
					},
				},
			},
		},
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	// Execute
	server.configHandler(w, req)

	// Parse response as raw JSON
	var rawResponse map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&rawResponse); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify top-level structure
	if _, exists := rawResponse["currentContext"]; !exists {
		t.Error("response missing 'currentContext' field")
	}

	if _, exists := rawResponse["contexts"]; !exists {
		t.Error("response missing 'contexts' field")
	}

	// Verify only expected fields are present
	expectedFields := map[string]bool{"currentContext": true, "contexts": true}
	for field := range rawResponse {
		if !expectedFields[field] {
			t.Errorf("unexpected field '%s' in response", field)
		}
	}
}
