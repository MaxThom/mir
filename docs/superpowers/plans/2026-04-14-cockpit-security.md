# Cockpit Security Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user authentication to Cockpit so only logged-in users can access the UI and obtain the NATS credentials needed to connect to the message bus.

**Architecture:** The Go cockpit server gains a login/logout/session API backed by a SurrealDB user table and an in-memory session store. A protected `/api/v1/auth/credentials` endpoint reads a pre-generated `.creds` file from disk and returns the JWT + NKey seed to authenticated browsers. The SvelteKit app gains a login page, an auth guard in the root layout, and uses `jwtAuthenticator` from `@nats-io/nats-core` to authenticate the NATS WebSocket connection.

**Tech Stack:** Go `golang.org/x/crypto/bcrypt` (already in go.mod), `crypto/rand` for session IDs, SurrealDB for user storage, `@nats-io/nats-core` `jwtAuthenticator` (already in SDK node_modules), SvelteKit `goto` for redirects.

---

## Scope

This plan intentionally defers:
- Per-user NATS permission scoping (all authenticated users share the same pre-generated `.creds`)
- Password reset, user management UI, OAuth/SSO
- Session persistence across server restarts (in-memory sessions only)

The HTTP session layer provides user identity; the NATS layer provides permission enforcement.

---

## File Map

**New Go files:**
- `internal/servers/cockpit_srv/user_store.go` — User model, bcrypt hashing, SurrealDB CRUD, admin seeding
- `internal/servers/cockpit_srv/session_store.go` — In-memory session map, UUID generation, cookie helpers
- `internal/servers/cockpit_srv/auth_handler.go` — Login, logout, me, credentials HTTP handlers

**Modified Go files:**
- `internal/servers/cockpit_srv/middleware.go` — Add `requireAuth` middleware + context key
- `internal/servers/cockpit_srv/server.go` — Add `userStore`, `sessionStore`, `webCreds` to `CockpitServer`; wire new routes; parse `.creds` file at startup
- `internal/ui/cli/serve_cmd.go` — Add `WebCredentials`, `AdminUsername`, `AdminPassword` fields to `CockpitCfg`

**New SvelteKit files:**
- `internal/ui/web/src/lib/domains/auth/types/types.ts` — `AuthUser`, `LoginRequest` types
- `internal/ui/web/src/lib/domains/auth/services/auth.ts` — `authService` (login, logout, me, credentials)
- `internal/ui/web/src/lib/domains/auth/stores/auth.svelte.ts` — `authStore` (user state, initialize, login, logout)
- `internal/ui/web/src/routes/login/+page.svelte` — Login form page
- `internal/ui/web/src/routes/login/+page.ts` — Disable prerendering for login route

**Modified SvelteKit files:**
- `internal/ui/web/src/routes/+layout.svelte` — Auth guard: redirect to `/login` if unauthenticated
- `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts` — Fetch credentials from API before connecting, pass `jwtAuthenticator`

---

## Task 1: Add security config fields

**Files:**
- Modify: `internal/ui/cli/serve_cmd.go`

- [ ] **Step 1: Add fields to `CockpitCfg`**

Open `internal/ui/cli/serve_cmd.go` and update the `CockpitCfg` struct:

```go
CockpitCfg struct {
    AllowedOrigins []string `help:"CORS allowed origins" yaml:"allowedOrigins"`
    GitHubOwner    string   `help:"GitHub owner for releases feed" default:"MaxThom" yaml:"githubOwner"`
    GitHubRepo     string   `help:"GitHub repo for releases feed" default:"mir" yaml:"githubRepo"`
    WebCredentials string   `help:"Path to .creds file issued to web clients (via 'mir tools security generate-creds')" default:"" yaml:"webCredentials"`
    AdminUsername  string   `help:"Initial admin username, seeded on first start" default:"admin" yaml:"adminUsername"`
    AdminPassword  string   `help:"Initial admin password, seeded on first start" default:"" cfg:"secret" yaml:"adminPassword"`
}
```

- [ ] **Step 2: Update `Options` and the cockpit server instantiation in `run()`**

In `serve_cmd.go`, update the `cockpit_srv.Options` struct literal inside `run()`:

```go
cockpitSrv, err := cockpit_srv.NewCockpit(log, &cockpit_srv.Options{
    AllowedOrigins: cfg.Module.Cockpit.AllowedOrigins,
    WebFS:          webFS,
    Config:         contexts,
    Store:          mngStore,
    GitHub: cockpit_srv.GitHubOptions{
        Owner: cfg.Module.Cockpit.GitHubOwner,
        Repo:  cfg.Module.Cockpit.GitHubRepo,
    },
    Security: cockpit_srv.SecurityOptions{
        WebCredentials: cfg.Module.Cockpit.WebCredentials,
        AdminUsername:  cfg.Module.Cockpit.AdminUsername,
        AdminPassword:  cfg.Module.Cockpit.AdminPassword,
    },
})
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/cli/serve_cmd.go
git commit -m "feat(cockpit): add security config fields to CockpitCfg"
```

---

## Task 2: User store

**Files:**
- Create: `internal/servers/cockpit_srv/user_store.go`
- Create: `internal/servers/cockpit_srv/user_store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/servers/cockpit_srv/user_store_test.go`:

```go
package cockpit_srv

import (
	"testing"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := hashPassword("mysecret")
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}
	if hash == "mysecret" {
		t.Fatal("hash must not equal plaintext")
	}
	if !checkPassword(hash, "mysecret") {
		t.Error("checkPassword returned false for correct password")
	}
	if checkPassword(hash, "wrong") {
		t.Error("checkPassword returned true for wrong password")
	}
}

func TestNewUser_SetsDefaults(t *testing.T) {
	u, err := newUser("alice", "secret", RoleOperator)
	if err != nil {
		t.Fatalf("newUser failed: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", u.Username)
	}
	if u.Role != RoleOperator {
		t.Errorf("expected role '%s', got '%s'", RoleOperator, u.Role)
	}
	if u.PasswordHash == "secret" {
		t.Error("password must be hashed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /path/to/mir.server
go test ./internal/servers/cockpit_srv/... -run TestHashPassword -v
```
Expected: FAIL — `hashPassword`, `checkPassword`, `newUser` not defined.

- [ ] **Step 3: Implement `user_store.go`**

Create `internal/servers/cockpit_srv/user_store.go`:

```go
package cockpit_srv

import (
	"errors"
	"fmt"

	"github.com/maxthom/mir/internal/libs/external/surreal"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Role         string `json:"role"`
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func newUser(username, password, role string) (User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, fmt.Errorf("hashing password: %w", err)
	}
	return User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}, nil
}

type userStore struct {
	db *surreal.AutoReconnDB
}

func newUserStore(db *surreal.AutoReconnDB) (*userStore, error) {
	s := &userStore{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *userStore) initSchema() error {
	q := `DEFINE TABLE IF NOT EXISTS cockpit_users SCHEMALESS;`
	_, err := surreal.Query[any](s.db, q, nil)
	return err
}

func (s *userStore) FindByUsername(username string) (User, error) {
	q := `SELECT * FROM cockpit_users WHERE username = $username LIMIT 1;`
	users, err := surreal.Query[[]User](s.db, q, map[string]any{"username": username})
	if err != nil {
		return User{}, fmt.Errorf("querying user: %w", err)
	}
	if len(users) == 0 {
		return User{}, ErrUserNotFound
	}
	return users[0], nil
}

func (s *userStore) Create(u User) (User, error) {
	created, err := surreal.Create[User](s.db, "cockpit_users", u)
	if err != nil {
		return User{}, fmt.Errorf("creating user: %w", err)
	}
	return *created, nil
}

func (s *userStore) Count() (int, error) {
	q := `SELECT count() FROM cockpit_users GROUP ALL;`
	type countResult struct {
		Count int `json:"count"`
	}
	results, err := surreal.Query[[]countResult](s.db, q, nil)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0].Count, nil
}

// Authenticate validates username/password and returns the user.
func (s *userStore) Authenticate(username, password string) (User, error) {
	u, err := s.FindByUsername(username)
	if errors.Is(err, ErrUserNotFound) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}
	if !checkPassword(u.PasswordHash, password) {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

// SeedAdmin creates the admin user if no users exist.
func (s *userStore) SeedAdmin(username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	count, err := s.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	u, err := newUser(username, password, RoleAdmin)
	if err != nil {
		return err
	}
	_, err = s.Create(u)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/servers/cockpit_srv/... -run "TestHashPassword|TestNewUser" -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/user_store.go internal/servers/cockpit_srv/user_store_test.go
git commit -m "feat(cockpit): add user model with bcrypt hashing and SurrealDB store"
```

---

## Task 3: Session store

**Files:**
- Create: `internal/servers/cockpit_srv/session_store.go`
- Create: `internal/servers/cockpit_srv/session_store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/servers/cockpit_srv/session_store_test.go`:

```go
package cockpit_srv

import (
	"testing"
	"time"
)

func TestSessionStore_CreateAndGet(t *testing.T) {
	ss := newSessionStore()
	sess := ss.create(User{ID: "user:1", Username: "alice", Role: RoleOperator})

	if sess.ID == "" {
		t.Fatal("session ID must not be empty")
	}

	got, ok := ss.get(sess.ID)
	if !ok {
		t.Fatal("expected session to be found")
	}
	if got.Username != "alice" {
		t.Errorf("expected username 'alice', got '%s'", got.Username)
	}
}

func TestSessionStore_Delete(t *testing.T) {
	ss := newSessionStore()
	sess := ss.create(User{ID: "user:1", Username: "bob", Role: RoleViewer})
	ss.delete(sess.ID)

	_, ok := ss.get(sess.ID)
	if ok {
		t.Fatal("session should not be found after delete")
	}
}

func TestSessionStore_ExpiredSession(t *testing.T) {
	ss := newSessionStore()
	sess := ss.create(User{ID: "user:1", Username: "carol", Role: RoleViewer})

	// Manually expire the session
	ss.mu.Lock()
	s := ss.sessions[sess.ID]
	s.ExpiresAt = time.Now().Add(-time.Minute)
	ss.sessions[sess.ID] = s
	ss.mu.Unlock()

	_, ok := ss.get(sess.ID)
	if ok {
		t.Fatal("expired session should not be returned")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/servers/cockpit_srv/... -run TestSessionStore -v
```
Expected: FAIL — `newSessionStore` not defined.

- [ ] **Step 3: Implement `session_store.go`**

Create `internal/servers/cockpit_srv/session_store.go`:

```go
package cockpit_srv

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "mir_session"
	sessionDuration   = 24 * time.Hour
)

type Session struct {
	ID        string
	UserID    string
	Username  string
	Role      string
	ExpiresAt time.Time
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func newSessionStore() *sessionStore {
	ss := &sessionStore{sessions: make(map[string]Session)}
	go ss.cleanupLoop()
	return ss
}

func (ss *sessionStore) create(u User) Session {
	id := generateID()
	sess := Session{
		ID:        id,
		UserID:    u.ID,
		Username:  u.Username,
		Role:      u.Role,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	ss.mu.Lock()
	ss.sessions[id] = sess
	ss.mu.Unlock()
	return sess
}

func (ss *sessionStore) get(id string) (Session, bool) {
	ss.mu.RLock()
	sess, ok := ss.sessions[id]
	ss.mu.RUnlock()
	if !ok || time.Now().After(sess.ExpiresAt) {
		return Session{}, false
	}
	return sess, true
}

func (ss *sessionStore) delete(id string) {
	ss.mu.Lock()
	delete(ss.sessions, id)
	ss.mu.Unlock()
}

func (ss *sessionStore) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		ss.mu.Lock()
		for id, sess := range ss.sessions {
			if now.After(sess.ExpiresAt) {
				delete(ss.sessions, id)
			}
		}
		ss.mu.Unlock()
	}
}

func (ss *sessionStore) setCookie(w http.ResponseWriter, sess Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})
}

func (ss *sessionStore) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/servers/cockpit_srv/... -run TestSessionStore -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/session_store.go internal/servers/cockpit_srv/session_store_test.go
git commit -m "feat(cockpit): add in-memory session store with HttpOnly cookie helpers"
```

---

## Task 4: Auth handlers

**Files:**
- Create: `internal/servers/cockpit_srv/auth_handler.go`
- Create: `internal/servers/cockpit_srv/auth_handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/servers/cockpit_srv/auth_handler_test.go`:

```go
package cockpit_srv

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func makeTestServer(t *testing.T) *CockpitServer {
	t.Helper()
	return &CockpitServer{
		log:          zerolog.Nop(),
		sessionStore: newSessionStore(),
		webCreds:     &webCredentials{JWT: "test-jwt", NKeySeed: "test-seed"},
	}
}

func TestLoginHandler_Success(t *testing.T) {
	s := makeTestServer(t)
	// Inject a user into the server without SurrealDB
	s.loginFn = func(username, password string) (User, error) {
		if username == "admin" && password == "secret" {
			return User{ID: "user:1", Username: "admin", Role: RoleAdmin}, nil
		}
		return User{}, ErrInvalidCredentials
	}

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.loginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Session cookie must be set
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			found = true
			if !c.HttpOnly {
				t.Error("session cookie must be HttpOnly")
			}
		}
	}
	if !found {
		t.Error("expected mir_session cookie to be set")
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	s := makeTestServer(t)
	s.loginFn = func(username, password string) (User, error) {
		return User{}, ErrInvalidCredentials
	}

	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.loginHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCredentialsHandler_RequiresAuth(t *testing.T) {
	s := makeTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/credentials", nil)
	w := httptest.NewRecorder()

	// No session cookie — should be blocked by requireAuth middleware
	// Here we test the handler directly without a session in context
	s.credentialsHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCredentialsHandler_ReturnsCreds(t *testing.T) {
	s := makeTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/credentials", nil)
	// Inject session into context (as requireAuth middleware would do)
	ctx := contextWithSession(req.Context(), Session{ID: "x", Username: "admin", Role: RoleAdmin})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	s.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp credentialsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.JWT == "" || resp.NKeySeed == "" {
		t.Error("expected non-empty JWT and NKeySeed")
	}
}

func TestMeHandler_ReturnsUser(t *testing.T) {
	s := makeTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	ctx := contextWithSession(req.Context(), Session{ID: "x", Username: "admin", Role: RoleAdmin})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	s.meHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["username"] != "admin" {
		t.Errorf("expected username 'admin', got '%s'", resp["username"])
	}
}

func TestLogoutHandler_ClearsCookie(t *testing.T) {
	s := makeTestServer(t)
	sess := s.sessionStore.create(User{ID: "user:1", Username: "admin", Role: RoleAdmin})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	w := httptest.NewRecorder()

	s.logoutHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Session should be gone
	_, ok := s.sessionStore.get(sess.ID)
	if ok {
		t.Error("session should be deleted after logout")
	}
	// Cookie should be cleared
	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookieName && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected mir_session cookie to be cleared (MaxAge < 0)")
	}
}

func TestLoginHandler_MethodNotAllowed(t *testing.T) {
	s := makeTestServer(t)
	s.loginFn = func(_, _ string) (User, error) { return User{}, nil }

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/v1/auth/login", nil)
		w := httptest.NewRecorder()
		s.loginHandler(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", method, w.Code)
		}
	}
}

func TestCredentialsHandler_NotConfigured(t *testing.T) {
	s := makeTestServer(t)
	s.webCreds = nil // not configured

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/credentials", nil)
	ctx := contextWithSession(req.Context(), Session{ID: "x", Username: "admin", Role: RoleAdmin})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	s.credentialsHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not configured") {
		t.Errorf("expected 'not configured' in body, got: %s", w.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/servers/cockpit_srv/... -run "TestLoginHandler|TestCredentialsHandler|TestMeHandler|TestLogoutHandler" -v
```
Expected: FAIL — types and handlers not defined.

- [ ] **Step 3: Implement `auth_handler.go`**

Create `internal/servers/cockpit_srv/auth_handler.go`:

```go
package cockpit_srv

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
)

type contextKey int

const sessionContextKey contextKey = iota

type webCredentials struct {
	JWT      string
	NKeySeed string
}

type credentialsResponse struct {
	JWT      string `json:"jwt"`
	NKeySeed string `json:"nkeySeed"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// contextWithSession stores a Session in a context.
func contextWithSession(ctx context.Context, sess Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, sess)
}

// sessionFromContext retrieves a Session from a context.
func sessionFromContext(ctx context.Context) (Session, bool) {
	sess, ok := ctx.Value(sessionContextKey).(Session)
	return sess, ok
}

// parseCredsFile extracts the JWT and NKey seed from a NATS .creds file.
// Format:
//
//	-----BEGIN NATS USER JWT-----
//	<jwt>
//	------END NATS USER JWT------
//	-----BEGIN USER NKEY SEED-----
//	<seed>
//	------END USER NKEY SEED------
func parseCredsFile(path string) (*webCredentials, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var jwt, seed string
	var inJWT, inSeed bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.Contains(line, "BEGIN NATS USER JWT"):
			inJWT = true
		case strings.Contains(line, "END NATS USER JWT"):
			inJWT = false
		case strings.Contains(line, "BEGIN USER NKEY SEED"):
			inSeed = true
		case strings.Contains(line, "END USER NKEY SEED"):
			inSeed = false
		case inJWT && line != "":
			jwt = line
		case inSeed && line != "":
			seed = line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if jwt == "" || seed == "" {
		return nil, errors.New("invalid .creds file: missing JWT or NKey seed")
	}
	return &webCredentials{JWT: jwt, NKeySeed: seed}, nil
}

// loginHandler handles POST /api/v1/auth/login
func (s *CockpitServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.loginFn(req.Username, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		s.log.Error().Err(err).Msg("login error")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	sess := s.sessionStore.create(user)
	s.sessionStore.setCookie(w, sess)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": user.Username,
		"role":     user.Role,
	})
}

// logoutHandler handles POST /api/v1/auth/logout
func (s *CockpitServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		s.sessionStore.delete(cookie.Value)
	}
	s.sessionStore.clearCookie(w)
	w.WriteHeader(http.StatusOK)
}

// meHandler handles GET /api/v1/auth/me — requires requireAuth middleware
func (s *CockpitServer) meHandler(w http.ResponseWriter, r *http.Request) {
	sess, ok := sessionFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": sess.Username,
		"role":     sess.Role,
	})
}

// credentialsHandler handles GET /api/v1/auth/credentials — requires requireAuth middleware
func (s *CockpitServer) credentialsHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := sessionFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if s.webCreds == nil {
		http.Error(w, "NATS web credentials not configured — set cockpit.webCredentials in mir.yaml", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(credentialsResponse{
		JWT:      s.webCreds.JWT,
		NKeySeed: s.webCreds.NKeySeed,
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/servers/cockpit_srv/... -run "TestLoginHandler|TestCredentialsHandler|TestMeHandler|TestLogoutHandler" -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/auth_handler.go internal/servers/cockpit_srv/auth_handler_test.go
git commit -m "feat(cockpit): add login, logout, me, and credentials HTTP handlers"
```

---

## Task 5: requireAuth middleware

**Files:**
- Modify: `internal/servers/cockpit_srv/middleware.go`
- Create: `internal/servers/cockpit_srv/middleware_test.go` (extend existing if present)

- [ ] **Step 1: Write the failing test**

Add to `internal/servers/cockpit_srv/middleware_test.go` (create if it doesn't exist):

```go
package cockpit_srv

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth_NoCookie(t *testing.T) {
	ss := newSessionStore()
	handler := requireAuth(ss)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_InvalidSession(t *testing.T) {
	ss := newSessionStore()
	handler := requireAuth(ss)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "nonexistent-session"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuth_ValidSession(t *testing.T) {
	ss := newSessionStore()
	sess := ss.create(User{ID: "user:1", Username: "alice", Role: RoleOperator})

	var capturedUsername string
	handler := requireAuth(ss)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, _ := sessionFromContext(r.Context())
		capturedUsername = s.Username
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedUsername != "alice" {
		t.Errorf("expected username 'alice' in context, got '%s'", capturedUsername)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/servers/cockpit_srv/... -run TestRequireAuth -v
```
Expected: FAIL — `requireAuth` not defined.

- [ ] **Step 3: Add `requireAuth` to `middleware.go`**

Append to `internal/servers/cockpit_srv/middleware.go`:

```go
// requireAuth validates the session cookie and injects the Session into the request context.
// Returns 401 if no valid session is found.
func requireAuth(ss *sessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			sess, ok := ss.get(cookie.Value)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(contextWithSession(r.Context(), sess))
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/servers/cockpit_srv/... -run TestRequireAuth -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/middleware.go internal/servers/cockpit_srv/middleware_test.go
git commit -m "feat(cockpit): add requireAuth middleware that validates session cookie"
```

---

## Task 6: Wire everything up in server.go

**Files:**
- Modify: `internal/servers/cockpit_srv/server.go`

- [ ] **Step 1: Update `Options`, `CockpitServer`, and `NewCockpit`**

Replace the contents of `internal/servers/cockpit_srv/server.go`:

```go
package cockpit_srv

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

type CockpitServer struct {
	log          zerolog.Logger
	opts         Options
	store        mng.DashboardStore
	cache        *releasesCache
	userStore    *userStore
	sessionStore *sessionStore
	webCreds     *webCredentials
	// loginFn is the authentication function; swapped in tests to avoid SurrealDB.
	loginFn func(username, password string) (User, error)
}

type Options struct {
	AllowedOrigins []string
	WebFS          fs.FS
	Config         ui.Config
	Store          mng.DashboardStore
	GitHub         GitHubOptions
	Security       SecurityOptions
	DB             *surreal.AutoReconnDB // optional; if nil, auth is disabled
}

type GitHubOptions struct {
	Owner string
	Repo  string
}

type SecurityOptions struct {
	WebCredentials string
	AdminUsername  string
	AdminPassword  string
}

func NewCockpit(logger zerolog.Logger, opts *Options) (*CockpitServer, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.WebFS == nil {
		return nil, fmt.Errorf("webFS cannot be nil")
	}

	s := &CockpitServer{
		log:          logger.With().Str("srv", "cockpit_server").Logger(),
		opts:         *opts,
		store:        opts.Store,
		cache:        &releasesCache{},
		sessionStore: newSessionStore(),
	}

	// Parse web credentials from disk if configured
	if opts.Security.WebCredentials != "" {
		creds, err := parseCredsFile(opts.Security.WebCredentials)
		if err != nil {
			return nil, fmt.Errorf("parsing web credentials file %q: %w", opts.Security.WebCredentials, err)
		}
		s.webCreds = creds
		s.log.Info().Str("path", opts.Security.WebCredentials).Msg("web credentials loaded")
	} else {
		s.log.Warn().Msg("cockpit.webCredentials not configured — /api/v1/auth/credentials will return 503")
	}

	// Setup user store if DB is provided
	if opts.DB != nil {
		us, err := newUserStore(opts.DB)
		if err != nil {
			return nil, fmt.Errorf("initializing user store: %w", err)
		}
		s.userStore = us
		s.loginFn = us.Authenticate

		// Seed admin user on first start
		if err := us.SeedAdmin(opts.Security.AdminUsername, opts.Security.AdminPassword); err != nil {
			s.log.Error().Err(err).Msg("failed to seed admin user")
		} else if opts.Security.AdminUsername != "" {
			s.log.Info().Str("username", opts.Security.AdminUsername).Msg("admin user seeded (no-op if already exists)")
		}
	} else {
		s.log.Warn().Msg("no DB provided to cockpit — auth endpoints will return 503")
		s.loginFn = func(_, _ string) (User, error) {
			return User{}, fmt.Errorf("authentication not configured")
		}
	}

	return s, nil
}

func (s *CockpitServer) RegisterRoutes(mux *http.ServeMux) {
	// Auth routes (login/logout are public; me/credentials are protected)
	s.registerPublicRoute(mux, "POST /api/v1/auth/login", s.loginHandler)
	s.registerPublicRoute(mux, "POST /api/v1/auth/logout", s.logoutHandler)
	s.registerProtectedRoute(mux, "GET /api/v1/auth/me", s.meHandler)
	s.registerProtectedRoute(mux, "GET /api/v1/auth/credentials", s.credentialsHandler)

	// Existing API routes
	s.registerPublicRoute(mux, "/api/v1/contexts", s.configHandler)

	if s.opts.GitHub.Owner != "" {
		s.registerPublicRoute(mux, "/api/v1/releases", s.releasesHandler)
	}

	if s.store != nil {
		if err := SeedWelcomeDashboard(s.log, s.store); err != nil {
			s.log.Error().Err(err).Msg("failed to seed welcome dashboard")
		}
		s.registerDashboardRoute(mux, "/api/v1/dashboards", s.dashboardsHandler)
		s.registerDashboardRoute(mux, "/api/v1/dashboards/", s.dashboardHandler)
	}

	// SPA handler
	spaHandler := createSPAHandler(s.opts.WebFS, s.log)
	handler := metricsMiddleware(spaHandler)
	handler = loggingMiddleware(s.log)(handler)
	handler = securityHeadersMiddleware(handler)
	handler = corsMiddleware(s.opts.AllowedOrigins)(handler)
	mux.Handle("/", handler)

	s.log.Debug().Msg("cockpit routes registered")
}

func (s *CockpitServer) registerPublicRoute(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	h := metricsMiddleware(http.HandlerFunc(handler))
	h = loggingMiddleware(s.log)(h)
	h = securityHeadersMiddleware(h)
	h = corsMiddleware(s.opts.AllowedOrigins)(h)
	mux.Handle(pattern, h)
}

func (s *CockpitServer) registerProtectedRoute(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	h := metricsMiddleware(http.HandlerFunc(handler))
	h = loggingMiddleware(s.log)(h)
	h = requireAuth(s.sessionStore)(h)
	h = securityHeadersMiddleware(h)
	h = corsMiddleware(s.opts.AllowedOrigins)(h)
	mux.Handle(pattern, h)
}

func (s *CockpitServer) registerDashboardRoute(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.registerPublicRoute(mux, pattern, handler)
}
```

- [ ] **Step 2: Pass `db` to cockpit in `serve_cmd.go`**

In `internal/ui/cli/serve_cmd.go`, update the `cockpit_srv.Options` instantiation to include `DB: db`:

```go
cockpitSrv, err := cockpit_srv.NewCockpit(log, &cockpit_srv.Options{
    AllowedOrigins: cfg.Module.Cockpit.AllowedOrigins,
    WebFS:          webFS,
    Config:         contexts,
    Store:          mngStore,
    GitHub: cockpit_srv.GitHubOptions{
        Owner: cfg.Module.Cockpit.GitHubOwner,
        Repo:  cfg.Module.Cockpit.GitHubRepo,
    },
    Security: cockpit_srv.SecurityOptions{
        WebCredentials: cfg.Module.Cockpit.WebCredentials,
        AdminUsername:  cfg.Module.Cockpit.AdminUsername,
        AdminPassword:  cfg.Module.Cockpit.AdminPassword,
    },
    DB: db,
})
```

- [ ] **Step 3: Build to verify no compile errors**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 4: Run all cockpit tests**

```bash
go test ./internal/servers/cockpit_srv/... -v
```
Expected: all tests pass (integration tests that need SurrealDB are skipped or handled separately).

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/server.go internal/ui/cli/serve_cmd.go
git commit -m "feat(cockpit): wire auth stores, credentials, and protected routes into server"
```

---

## Task 7: Frontend auth types, service, and store

**Files:**
- Create: `internal/ui/web/src/lib/domains/auth/types/types.ts`
- Create: `internal/ui/web/src/lib/domains/auth/services/auth.ts`
- Create: `internal/ui/web/src/lib/domains/auth/stores/auth.svelte.ts`

- [ ] **Step 1: Create types**

Create `internal/ui/web/src/lib/domains/auth/types/types.ts`:

```typescript
export type AuthUser = {
	username: string;
	role: string;
};

export type LoginRequest = {
	username: string;
	password: string;
};

export type NatsCredentials = {
	jwt: string;
	nkeySeed: string;
};
```

- [ ] **Step 2: Create auth service**

Create `internal/ui/web/src/lib/domains/auth/services/auth.ts`:

```typescript
import { api } from '$lib/shared/services/api';
import type { AuthUser, LoginRequest, NatsCredentials } from '../types/types';
import type { ApiResponse } from '$lib/shared/types/api';

export const authService = {
	async login(req: LoginRequest): Promise<ApiResponse<AuthUser>> {
		return api.post<AuthUser>('/v1/auth/login', req);
	},

	async logout(): Promise<void> {
		await api.post('/v1/auth/logout');
	},

	async me(): Promise<ApiResponse<AuthUser>> {
		return api.get<AuthUser>('/v1/auth/me');
	},

	async credentials(): Promise<ApiResponse<NatsCredentials>> {
		return api.get<NatsCredentials>('/v1/auth/credentials');
	}
};
```

- [ ] **Step 3: Create auth store**

Create `internal/ui/web/src/lib/domains/auth/stores/auth.svelte.ts`:

```typescript
import { goto } from '$app/navigation';
import { authService } from '../services/auth';
import type { AuthUser, LoginRequest } from '../types/types';

class AuthStore {
	user = $state<AuthUser | null>(null);
	isLoading = $state(false);
	error = $state<string | null>(null);

	get isAuthenticated(): boolean {
		return this.user !== null;
	}

	async initialize(): Promise<void> {
		this.isLoading = true;
		try {
			const resp = await authService.me();
			this.user = resp.data;
		} catch {
			this.user = null;
		} finally {
			this.isLoading = false;
		}
	}

	async login(req: LoginRequest): Promise<boolean> {
		this.isLoading = true;
		this.error = null;
		try {
			const resp = await authService.login(req);
			this.user = resp.data;
			return true;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Login failed';
			return false;
		} finally {
			this.isLoading = false;
		}
	}

	async logout(): Promise<void> {
		try {
			await authService.logout();
		} finally {
			this.user = null;
			goto('/login');
		}
	}
}

export const authStore = new AuthStore();
```

- [ ] **Step 4: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^$" | head -20
```
Expected: no new errors (3 pre-existing errors in unrelated files are acceptable).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/web/src/lib/domains/auth/
git commit -m "feat(cockpit): add auth types, service, and store"
```

---

## Task 8: Login page

**Files:**
- Create: `internal/ui/web/src/routes/login/+page.svelte`
- Create: `internal/ui/web/src/routes/login/+page.ts`

- [ ] **Step 1: Disable prerender for login route**

Create `internal/ui/web/src/routes/login/+page.ts`:

```typescript
export const prerender = false;
```

- [ ] **Step 2: Create the login page**

Create `internal/ui/web/src/routes/login/+page.svelte`:

```svelte
<script lang="ts">
	import { authStore } from '$lib/domains/auth/stores/auth.svelte';
	import { goto } from '$app/navigation';
	import { Button } from '$lib/shared/components/shadcn/button';
	import LogInIcon from '@lucide/svelte/icons/log-in';

	let username = $state('');
	let password = $state('');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		const ok = await authStore.login({ username, password });
		if (ok) {
			goto('/');
		}
	}
</script>

<div class="flex min-h-svh items-center justify-center bg-background">
	<div class="w-full max-w-sm space-y-6 p-8">
		<div class="space-y-2 text-center">
			<h1 class="text-2xl font-semibold tracking-tight">Mir Cockpit</h1>
			<p class="text-sm text-muted-foreground">Sign in to continue</p>
		</div>

		<form onsubmit={handleSubmit} class="space-y-4">
			<div class="space-y-2">
				<label for="username" class="text-sm font-medium">Username</label>
				<input
					id="username"
					type="text"
					bind:value={username}
					autocomplete="username"
					required
					class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
					placeholder="admin"
				/>
			</div>

			<div class="space-y-2">
				<label for="password" class="text-sm font-medium">Password</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					autocomplete="current-password"
					required
					class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
				/>
			</div>

			{#if authStore.error}
				<p class="text-sm text-destructive">{authStore.error}</p>
			{/if}

			<Button type="submit" class="w-full" disabled={authStore.isLoading}>
				<LogInIcon class="mr-2 size-4" />
				{authStore.isLoading ? 'Signing in…' : 'Sign in'}
			</Button>
		</form>
	</div>
</div>
```

- [ ] **Step 3: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^$" | head -20
```
Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/web/src/routes/login/
git commit -m "feat(cockpit): add login page"
```

---

## Task 9: Auth guard in root layout

**Files:**
- Modify: `internal/ui/web/src/routes/+layout.svelte`

- [ ] **Step 1: Add auth initialization and redirect guard**

In `internal/ui/web/src/routes/+layout.svelte`, update the `<script>` block:

Add the import at the top of the existing imports:
```typescript
import { authStore } from '$lib/domains/auth/stores/auth.svelte';
import { goto } from '$app/navigation';
```

Replace the existing `onMount` function:
```typescript
onMount(async () => {
    await authStore.initialize();
    if (!authStore.isAuthenticated && page.url.pathname !== '/login') {
        goto('/login');
        return;
    }
    await contextStore.initialize();
    themeStore.init();
});
```

- [ ] **Step 2: Wrap children render with auth check**

In the template section, wrap `{@render children()}` so the main UI is not shown while auth is loading or when unauthenticated:

```svelte
{:else if !authStore.isLoading && authStore.isAuthenticated}
    {@render children()}
{/if}
```

(Replace the existing `{@render children()}` line in the `else` branch.)

- [ ] **Step 3: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^$" | head -20
```
Expected: no new errors.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/web/src/routes/+layout.svelte
git commit -m "feat(cockpit): add auth guard — redirect to /login if unauthenticated"
```

---

## Task 10: Use NATS credentials in mir.svelte.ts

**Files:**
- Modify: `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts`

- [ ] **Step 1: Update connect to fetch and use NATS credentials**

Replace the contents of `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts`:

```typescript
import { Mir } from '@mir/sdk';
import { jwtAuthenticator } from '@nats-io/nats-core';
import type { Context } from '../../contexts/types/types';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
import { authService } from '$lib/domains/auth/services/auth';

// Converts "nats://host:port" → "ws://host:9222"
function toWsUrl(natsTarget: string): string {
	return natsTarget.replace(/^nats:\/\//, 'ws://').replace(/:\d+$/, ':9222');
}

class MirStore {
	mir = $state<Mir | null>(null);
	isConnecting = $state(false);
	error = $state<string | null>(null);

	private connectionId = 0;

	get isConnected(): boolean {
		return this.mir !== null;
	}

	async connect(ctx: Context) {
		const id = ++this.connectionId;

		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}

		this.isConnecting = true;
		this.error = null;

		try {
			const wsUrl = toWsUrl(ctx.target);

			// Attempt to get NATS credentials from the server.
			// If the endpoint returns 503 (not configured), fall back to no auth
			// so the app still works in development with an open NATS server.
			let connectOpts: Record<string, unknown> = { maxReconnectAttempts: 0 };
			try {
				const credsResp = await authService.credentials();
				const { jwt, nkeySeed } = credsResp.data;
				connectOpts = {
					...connectOpts,
					authenticator: jwtAuthenticator(jwt, new TextEncoder().encode(nkeySeed))
				};
			} catch {
				// credentials endpoint not configured — connect unauthenticated
			}

			const mir = await Mir.connect('cockpit', wsUrl, connectOpts);

			if (id !== this.connectionId) {
				await mir.disconnect();
				return;
			}

			this.mir = mir;
			activityStore.add({
				kind: 'success',
				category: 'Connection',
				title: 'Connected',
				request: { context: ctx.name }
			});
		} catch (err) {
			if (id === this.connectionId) {
				this.error = err instanceof Error ? err.message : 'Connection failed';
			}
			activityStore.add({
				kind: 'error',
				category: 'Connection',
				title: 'Connection Failed',
				error: err instanceof Error ? err.message : String(err)
			});
		} finally {
			if (id === this.connectionId) {
				this.isConnecting = false;
			}
		}
	}

	async disconnect() {
		++this.connectionId;
		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
			activityStore.add({ kind: 'info', category: 'Connection', title: 'Disconnected' });
		}
	}
}

export const mirStore = new MirStore();
```

- [ ] **Step 2: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^$" | head -20
```
Expected: no new errors.

- [ ] **Step 3: Build the web app**

```bash
cd internal/ui/web && npm run build 2>&1 | tail -20
```
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts
git commit -m "feat(cockpit): use NATS jwtAuthenticator with credentials from /api/v1/auth/credentials"
```

---

## Manual Test Checklist

Once the server is built and running with `mir serve --cockpit`, verify end-to-end:

**Setup (one-time):**
```bash
# Generate cockpit web user credentials
mir tools security add client cockpit-web
mir tools security generate-creds cockpit-web --path ~/.config/mir/cockpit-web.creds

# Configure mir.yaml
# cockpit:
#   webCredentials: ~/.config/mir/cockpit-web.creds
#   adminUsername: admin
#   adminPassword: changeme
```

**Browser tests:**
- [ ] Visiting `/` without a session redirects to `/login`
- [ ] Login with wrong password shows "Invalid username or password"
- [ ] Login with `admin` / `changeme` succeeds and redirects to `/`
- [ ] After login, the NATS connection indicator shows "Connected"
- [ ] Refreshing the page stays logged in (session cookie persists)
- [ ] Clicking logout redirects to `/login`
- [ ] After logout, visiting `/` redirects back to `/login`
- [ ] Without `webCredentials` configured, the app still loads (unauthenticated NATS fallback)

---

## Future Work (not in this plan)

- Per-user NATS permission scoping (issue short-lived user-specific JWTs via NSC)
- User management UI (create/delete users from Cockpit)
- Password change flow
- Session persistence across server restarts (move sessions to SurrealDB)
- HTTPS enforcement + `Secure` flag on session cookie
- Rate limiting on `/api/v1/auth/login`
