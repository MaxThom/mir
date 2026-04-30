package cockpit_srv

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

func TestCredentialsHandler_NoPassword_NoCreds(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{{Name: "local", Target: "nats://localhost:4222"}},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=local", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestCredentialsHandler_NoPassword_WithCreds(t *testing.T) {
	credsContent := "-----BEGIN NATS USER JWT-----\nfakejwt\n------END NATS USER JWT------\n\n-----BEGIN USER NKEY SEED-----\nfakeseed\n------END USER NKEY SEED------\n"

	tmp, err := os.CreateTemp("", "test-*.creds")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString(credsContent)
	tmp.Close()

	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{{Name: "local", Target: "nats://localhost:4222", Sec: ui.ContextSecurity{Credentials: tmp.Name()}}},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=local", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	var resp CredentialsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Creds != credsContent {
		t.Errorf("expected creds content to match file, got %q", resp.Creds)
	}
}

func TestCredentialsHandler_WithPassword_Correct(t *testing.T) {
	tmp, err := os.CreateTemp("", "test-*.creds")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("fakecreds")
	tmp.Close()

	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "prod",
				Contexts:       []ui.Context{{Name: "prod", Target: "nats://prod:4222", Sec: ui.ContextSecurity{Credentials: tmp.Name(), Password: "secret"}}},
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=prod", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCredentialsHandler_WithPassword_Wrong(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "prod",
				Contexts:       []ui.Context{{Name: "prod", Target: "nats://prod:4222", Sec: ui.ContextSecurity{Password: "secret"}}},
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=prod", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCredentialsHandler_WithPassword_EmptyBody(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "prod",
				Contexts:       []ui.Context{{Name: "prod", Target: "nats://prod:4222", Sec: ui.ContextSecurity{Password: "secret"}}},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=prod", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCredentialsHandler_WithPassword_MissingField(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "prod",
				Contexts:       []ui.Context{{Name: "prod", Target: "nats://prod:4222", Sec: ui.ContextSecurity{Password: "secret"}}},
			},
		},
	}

	// Valid JSON but no password field — treated as empty password
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=prod", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCredentialsHandler_ContextNotFound(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{{Name: "local", Target: "nats://localhost:4222"}},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=nonexistent", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCredentialsHandler_DefaultsToCurrentContext(t *testing.T) {
	tmp, err := os.CreateTemp("", "test-*.creds")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("fakecreds")
	tmp.Close()

	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{{Name: "local", Target: "nats://localhost:4222", Sec: ui.ContextSecurity{Credentials: tmp.Name()}}},
			},
		},
	}

	// No ?context= param — must use currentContext
	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCredentialsHandler_FileNotFound(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts:       []ui.Context{{Name: "local", Target: "nats://localhost:4222", Sec: ui.ContextSecurity{Credentials: "/nonexistent/path.creds"}}},
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credentials?context=local", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestCredentialsHandler_MethodNotAllowed(t *testing.T) {
	server := &CockpitServer{
		log:  zerolog.Nop(),
		opts: Options{Config: ui.Config{}},
	}

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/credentials", nil)
			w := httptest.NewRecorder()
			server.credentialsHandler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s, got %d", method, w.Code)
			}
		})
	}
}
