# Cockpit Security & TLS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Secure the browser↔NATS connection in Cockpit with pre-configured JWT/nkey credentials, HTTPS on the Cockpit server, and WSS on the NATS WebSocket port.

**Architecture:** Cockpit reads a pre-configured `.creds` file per context (already a field in `ui.Context`) and exposes `GET /api/v1/credentials?context=<name>`; the browser fetches those creds before connecting and passes them as a `credsAuthenticator` to `Mir.connect`. Cockpit HTTP server switches to `ListenAndServeTLS` when `tlsCert`/`tlsKey` are set. `toWsUrl` auto-detects `https:` to emit `wss://`. NATS config documents a TLS websocket block for production. k8s values expose port 9222 via the NATS service.

**Tech Stack:** Go (net/http), SvelteKit 5 / Svelte runes, `@nats-io/nats-core` (`credsAuthenticator`), NATS server config, Helm

---

## File Map

**Create:**
- `internal/servers/cockpit_srv/credentials_handler.go` — reads .creds file for a named context, returns content as JSON
- `internal/servers/cockpit_srv/credentials_handler_test.go` — unit tests for the handler

**Modify:**
- `internal/servers/cockpit_srv/server.go` — register `/api/v1/credentials` route
- `cmds/cockpit/main.go` — add `TlsCert`/`TlsKey` to `HttpServer` config struct; conditional `ListenAndServeTLS`
- `pkgs/web/src/index.ts` — re-export `credsAuthenticator` from `@nats-io/nats-core`
- `internal/ui/web/src/lib/shared/types/api.ts` — add `CredentialsResponse` type
- `internal/ui/web/src/lib/domains/contexts/services/contexts.ts` — add `getCredentials()` method
- `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts` — fetch creds, update `toWsUrl` for `wss://`, pass `authenticator` to `Mir.connect`
- `infra/compose/natsio/config.conf` — document TLS websocket block for production
- `infra/k8s/charts/mir/values.yaml` — expose NATS websocket port 9222

---

## Task 1: Credentials Handler (Go)

**Files:**
- Create: `internal/servers/cockpit_srv/credentials_handler.go`
- Create: `internal/servers/cockpit_srv/credentials_handler_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/servers/cockpit_srv/credentials_handler_test.go`:

```go
package cockpit_srv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

func TestCredentialsHandler_Success(t *testing.T) {
	credsContent := "-----BEGIN NATS USER JWT-----\neyJmYWtlIjoidGVzdCJ9\n------END NATS USER JWT------\n\n-----BEGIN USER NKEY SEED-----\nSUAFAKESEEDFORTESTING\n------END USER NKEY SEED------\n"

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
				Contexts: []ui.Context{
					{Name: "local", Target: "nats://localhost:4222", Credentials: tmp.Name()},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials?context=local", nil)
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	var resp CredentialsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Creds != credsContent {
		t.Errorf("expected creds content to match file, got %q", resp.Creds)
	}
}

func TestCredentialsHandler_NoCredentialsConfigured(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts: []ui.Context{
					{Name: "local", Target: "nats://localhost:4222"},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials?context=local", nil)
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
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
				Contexts: []ui.Context{
					{Name: "local", Target: "nats://localhost:4222", Credentials: tmp.Name()},
				},
			},
		},
	}

	// No ?context= param — should default to currentContext
	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials", nil)
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials?context=nonexistent", nil)
	w := httptest.NewRecorder()
	server.credentialsHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCredentialsHandler_FileNotFound(t *testing.T) {
	server := &CockpitServer{
		log: zerolog.Nop(),
		opts: Options{
			Config: ui.Config{
				CurrentContext: "local",
				Contexts: []ui.Context{
					{Name: "local", Target: "nats://localhost:4222", Credentials: "/nonexistent/path.creds"},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/credentials?context=local", nil)
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

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
go test ./internal/servers/cockpit_srv/... -run TestCredentials -v
```

Expected: compile error — `credentialsHandler` and `CredentialsResponse` undefined.

- [ ] **Step 3: Implement the credentials handler**

Create `internal/servers/cockpit_srv/credentials_handler.go`:

```go
package cockpit_srv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// CredentialsResponse holds the raw .creds file content for a context.
// The browser passes this directly to credsAuthenticator from @nats-io/nats-core.
type CredentialsResponse struct {
	Creds string `json:"creds"`
}

// credentialsHandler handles GET /api/v1/credentials?context=<name>
// Returns the .creds file content for the named context, or 204 if no credentials are configured.
func (s *CockpitServer) credentialsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contextName := r.URL.Query().Get("context")
	if contextName == "" {
		contextName = s.opts.Config.CurrentContext
	}

	var credPath string
	found := false
	for _, ctx := range s.opts.Config.Contexts {
		if ctx.Name == contextName {
			credPath = ctx.Credentials
			found = true
			break
		}
	}

	if !found {
		http.Error(w, fmt.Sprintf("context %q not found", contextName), http.StatusNotFound)
		return
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
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/servers/cockpit_srv/... -run TestCredentials -v
```

Expected: all 6 `TestCredentials*` tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/servers/cockpit_srv/credentials_handler.go internal/servers/cockpit_srv/credentials_handler_test.go
git commit -m "feat(cockpit): add credentials handler endpoint"
```

---

## Task 2: Register Credentials Route

**Files:**
- Modify: `internal/servers/cockpit_srv/server.go`

- [ ] **Step 1: Register the route in `RegisterRoutes`**

In `internal/servers/cockpit_srv/server.go`, after the `/api/v1/contexts` registration (line 58), add:

```go
// Credentials endpoint — serves .creds file content for a named context
credsHandler := metricsMiddleware(http.HandlerFunc(s.credentialsHandler))
credsHandler = loggingMiddleware(s.log)(credsHandler)
credsHandler = securityHeadersMiddleware(credsHandler)
credsHandler = corsMiddleware(s.opts.AllowedOrigins)(credsHandler)
mux.Handle("/api/v1/credentials", credsHandler)
```

The full updated `RegisterRoutes` block around that section looks like:

```go
func (s *CockpitServer) RegisterRoutes(mux *http.ServeMux) {
	apiHandler := metricsMiddleware(http.HandlerFunc(s.configHandler))
	apiHandler = loggingMiddleware(s.log)(apiHandler)
	apiHandler = securityHeadersMiddleware(apiHandler)
	apiHandler = corsMiddleware(s.opts.AllowedOrigins)(apiHandler)
	mux.Handle("/api/v1/contexts", apiHandler)

	credsHandler := metricsMiddleware(http.HandlerFunc(s.credentialsHandler))
	credsHandler = loggingMiddleware(s.log)(credsHandler)
	credsHandler = securityHeadersMiddleware(credsHandler)
	credsHandler = corsMiddleware(s.opts.AllowedOrigins)(credsHandler)
	mux.Handle("/api/v1/credentials", credsHandler)

	// ... rest of existing code unchanged
```

- [ ] **Step 2: Build to confirm no errors**

```bash
go build ./internal/servers/cockpit_srv/...
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add internal/servers/cockpit_srv/server.go
git commit -m "feat(cockpit): register /api/v1/credentials route"
```

---

## Task 3: TLS for Cockpit HTTP Server

**Files:**
- Modify: `cmds/cockpit/main.go`

- [ ] **Step 1: Add `TlsCert` and `TlsKey` to `HttpServer` config struct**

In `cmds/cockpit/main.go`, update the `HttpServer` struct (currently around line 48):

```go
HttpServer struct {
    Port           int
    AllowedOrigins []string `yaml:"allowedOrigins"`
    TlsCert        string   `yaml:"tlsCert"`
    TlsKey         string   `yaml:"tlsKey"`
}
```

- [ ] **Step 2: Switch to conditional `ListenAndServeTLS` in `run()`**

Replace the goroutine that starts the HTTP server (around line 232–244) with:

```go
wg.Add(1)
go func() {
    defer wg.Done()
    health.SetComponentReady(health.ComponentHttp)
    log.Info().Int("port", cfg.HttpServer.Port).Msg("starting cockpit web server")
    var serveErr error
    if cfg.HttpServer.TlsCert != "" && cfg.HttpServer.TlsKey != "" {
        log.Info().Str("cert", cfg.HttpServer.TlsCert).Str("key", cfg.HttpServer.TlsKey).Msg("TLS enabled")
        serveErr = server.ListenAndServeTLS(cfg.HttpServer.TlsCert, cfg.HttpServer.TlsKey)
    } else {
        serveErr = server.ListenAndServe()
    }
    if serveErr != nil && serveErr != http.ErrServerClosed {
        health.SetComponentUnready(health.ComponentHttp)
        log.Error().Err(serveErr).Msg("http server error")
        health.SetUnready()
        mir_signals.Shutdown()
    }
    log.Debug().Msg("http server shutdown")
}()
```

- [ ] **Step 3: Build to confirm no errors**

```bash
go build ./cmds/cockpit/...
```

Expected: exits 0, no output.

- [ ] **Step 4: Commit**

```bash
git add cmds/cockpit/main.go
git commit -m "feat(cockpit): add optional TLS to HTTP server"
```

---

## Task 4: SDK — Export `credsAuthenticator`

**Files:**
- Modify: `pkgs/web/src/index.ts`

- [ ] **Step 1: Add re-export to SDK index**

In `pkgs/web/src/index.ts`, add one line after the existing exports:

```typescript
/**
 * @mir/web-sdk
 *
 * TypeScript SDK for Mir IoT Hub - Real-time device communication via NATS
 */

export { Mir } from "./mir";
export type { MirOptions } from "./mir";

export * from "./models";
export * from "./server_core";
export * from "./server_event";
export * from "./server_cmd";
export * from "./server_cfg";
export * from "./server_tlm";
export * from "./client";

export { credsAuthenticator } from "@nats-io/nats-core";

// Package version
export const VERSION = "0.1.0";
```

- [ ] **Step 2: Rebuild the SDK**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server/pkgs/web
npm run build
```

Expected: outputs files to `dist/`, exits 0.

- [ ] **Step 3: Confirm `credsAuthenticator` is in the dist**

```bash
grep -l "credsAuthenticator" dist/index.js dist/index.d.ts
```

Expected: both files listed.

- [ ] **Step 4: Commit**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
git add pkgs/web/src/index.ts pkgs/web/dist/
git commit -m "feat(sdk): export credsAuthenticator from @nats-io/nats-core"
```

---

## Task 5: Frontend — Credentials Type and Service

**Files:**
- Modify: `internal/ui/web/src/lib/shared/types/api.ts`
- Modify: `internal/ui/web/src/lib/domains/contexts/services/contexts.ts`

- [ ] **Step 1: Add `CredentialsResponse` type**

In `internal/ui/web/src/lib/shared/types/api.ts`, append:

```typescript
export type CredentialsResponse = {
	creds: string;
};
```

- [ ] **Step 2: Add `getCredentials` to the context service**

In `internal/ui/web/src/lib/domains/contexts/services/contexts.ts`, update to:

```typescript
import { api } from '../../../shared/services/api';
import type { ApiResponse, ContextsResponse, CredentialsResponse } from '../../../shared/types/api';

export const contextService = {
	async getAll(): Promise<ApiResponse<ContextsResponse>> {
		return api.get<ContextsResponse>('/v1/contexts');
	},

	async getCredentials(contextName: string): Promise<string | null> {
		const response = await fetch(
			`/api/v1/credentials?context=${encodeURIComponent(contextName)}`
		);
		if (response.status === 204) return null;
		if (!response.ok) throw new Error(`credentials request failed: ${response.status}`);
		const data: CredentialsResponse = await response.json();
		return data.creds;
	}
};
```

- [ ] **Step 3: Type-check**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server/internal/ui/web
npm run check 2>&1 | grep -v "node_modules"
```

Expected: no new errors beyond the 3 pre-existing ones in `nav-section.svelte` and `multi/telemetry`.

- [ ] **Step 4: Commit**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
git add internal/ui/web/src/lib/shared/types/api.ts \
        internal/ui/web/src/lib/domains/contexts/services/contexts.ts
git commit -m "feat(cockpit): add credentials type and service method"
```

---

## Task 6: Frontend — Wire Credentials and WSS into `mir.svelte.ts`

**Files:**
- Modify: `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts`

- [ ] **Step 1: Rewrite `mir.svelte.ts`**

Replace the full file content:

```typescript
import { Mir, credsAuthenticator } from '@mir/sdk';
import type { MirOptions } from '@mir/sdk';
import type { Context } from '../../contexts/types/types';
import { contextService } from '../../contexts/services/contexts';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

// Converts "nats://host:port" → "ws://host:9222" or "wss://host:9222"
// Uses wss:// automatically when the page is served over https.
function toWsUrl(natsTarget: string): string {
	const scheme = window.location.protocol === 'https:' ? 'wss' : 'ws';
	return natsTarget.replace(/^nats:\/\//, `${scheme}://`).replace(/:\d+$/, ':9222');
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

			const opts: MirOptions = { maxReconnectAttempts: 0 };

			const creds = await contextService.getCredentials(ctx.name);
			if (creds !== null) {
				opts.authenticator = credsAuthenticator(new TextEncoder().encode(creds));
			}

			const mir = await Mir.connect('cockpit', wsUrl, opts);

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
cd /home/maxthom/code/mir-ecosystem/mir.server/internal/ui/web
npm run check 2>&1 | grep -v "node_modules"
```

Expected: no new errors beyond the 3 pre-existing ones.

- [ ] **Step 3: Commit**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
git add internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts
git commit -m "feat(cockpit): fetch NATS credentials and use wss:// when on https"
```

---

## Task 7: NATS Config — Document WSS for Production

**Files:**
- Modify: `infra/compose/natsio/config.conf`

- [ ] **Step 1: Update the websocket section with production TLS block**

Replace the existing `# WebSocket configuration (for browser clients)` section in `infra/compose/natsio/config.conf`:

```
# WebSocket configuration (for browser clients)
# Development — plain ws:// (no TLS)
websocket: {
  listen: 0.0.0.0:9222
  no_tls: true
  compress: true
}

# Production — replace the block above with this (wss://)
# Requires mounting certs into the container at /etc/nats/certs/
# websocket: {
#   listen: 0.0.0.0:9222
#   tls: {
#     cert_file: "/etc/nats/certs/tls.crt"
#     key_file:  "/etc/nats/certs/tls.key"
#   }
#   compress: true
# }
```

- [ ] **Step 2: Update the compose.yaml to show cert mount option**

In `infra/compose/natsio/compose.yaml`, add the cert volume as a commented option:

```yaml
services:
  nats:
    image: nats:2.12.4-alpine
    command: -c /etc/nats/config.conf
    restart: unless-stopped
    volumes:
      - ./config.conf:/etc/nats/config.conf:ro
      - ./resolver.conf:/etc/nats/resolver.conf:ro
      - ./certs:/etc/nats/certs   # mount TLS certs here for production
      - nats_data:/data
    ports:
      - 4222:4222
      - 8222:8222
      - 9222:9222
  nats-exporter:
    image: natsio/prometheus-nats-exporter:0.19.1
    ports:
      - "7777:7777"
    command: -varz -connz -routez -subz -prefix=nats -jsz all "http://nats:8222"
    restart: unless-stopped
    depends_on:
      - nats

volumes:
  nats_data:
```

- [ ] **Step 3: Commit**

```bash
git add infra/compose/natsio/config.conf infra/compose/natsio/compose.yaml
git commit -m "docs(nats): document production TLS websocket config"
```

---

## Task 8: Kubernetes — Expose NATS WebSocket Port

**Files:**
- Modify: `infra/k8s/charts/mir/values.yaml`

- [ ] **Step 1: Add websocket port to NATS service and config**

In `infra/k8s/charts/mir/values.yaml`, update the `nats:` section. Find the `service.merge.spec.ports` array (currently has only the `nats` port on 4222) and add port 9222:

```yaml
nats:
  enabled: true
  service:
    merge:
      spec:
        type: LoadBalancer
        ports:
          - appProtocol: tcp
            name: nats
            nodePort: 31422
            port: 4222
            protocol: TCP
            targetPort: nats
          - appProtocol: tcp
            name: websocket
            nodePort: 31922
            port: 9222
            protocol: TCP
            targetPort: 9222
  config:
    merge:
      max_payload: << 8MB >>
      websocket:
        port: 9222
        no_tls: true
```

> **Note:** For production k8s with TLS, replace `no_tls: true` with a `tls:` block referencing a k8s Secret mounted as `/etc/nats/certs/`. See `infra/k8s/charts/mir/secret/nats-tls.secret.yaml` for the secret template.

- [ ] **Step 2: Verify YAML is valid**

```bash
helm template infra/k8s/charts/mir -f infra/k8s/charts/mir/values.yaml > /dev/null
```

Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add infra/k8s/charts/mir/values.yaml
git commit -m "feat(k8s): expose NATS websocket port 9222"
```

---

## Verification Checklist

After all tasks, verify end-to-end with a local dev setup (no TLS — plain ws/http):

- [ ] `go test ./internal/servers/cockpit_srv/... -v` — all tests pass
- [ ] `go build ./cmds/cockpit/...` — builds clean
- [ ] Start infra: `just infra`
- [ ] Start cockpit: `go run ./cmds/cockpit/`
- [ ] Browser opens `http://localhost:3021` — Cockpit loads
- [ ] `/api/v1/credentials?context=local` returns 204 (no creds configured in dev)
- [ ] Connection succeeds without credentials (NATS has no auth in dev)

Then verify with credentials:
- [ ] Generate a test creds file: `mir tools security user add --name cockpit-web --scope client`
- [ ] Add `credentials: /path/to/cockpit-web.creds` to the context in cockpit config
- [ ] `/api/v1/credentials?context=local` now returns 200 with `{ "creds": "..." }`
- [ ] NATS connection succeeds with credentials

For TLS (requires certs):
- [ ] Generate dev certs with `mkcert localhost` or `openssl req -x509 ...`
- [ ] Add `tlsCert` / `tlsKey` to cockpit `httpServer` config
- [ ] Enable NATS websocket TLS block in `config.conf`
- [ ] Open `https://localhost:3021` — browser shows valid cert
- [ ] NATS connects on `wss://localhost:9222`

---

## Config Reference

Full `cockpit.yaml` example with security enabled:

```yaml
logLevel: info
httpServer:
  port: 3021
  tlsCert: /etc/mir/certs/cockpit.crt
  tlsKey:  /etc/mir/certs/cockpit.key
  allowedOrigins:
    - https://localhost:3021
contexts:
  currentContext: local
  contexts:
    - name: local
      target: nats://localhost:4222
      grafana: localhost:3000
      credentials: /etc/mir/creds/cockpit-web.creds
      rootCA: /etc/mir/certs/ca.crt
      username: admin           # optional — enables login gate for this context
      password: changeme
```

---

---

## Phase 2: Per-Context Authentication (implement last)

> Complete Tasks 1–8 first. These tasks build the login gate on top of the working credential + TLS stack.

**Design:** Each context in the cockpit config can have an optional `username`/`password`. The context list API exposes a `secured: bool` flag (no credentials exposed). When the SPA activates a secured context without a valid session token, it redirects to `/login?context=<name>`. The login page POSTs to `/api/v1/auth`, which validates credentials and returns a short-lived token. The token is stored in a Svelte store (memory only) and passed as `X-Session-Token` on every `/api/v1/credentials` call.

---

### Task 9: Backend Authentication

**Files:**
- Modify: `internal/ui/config.go` — add `Username`, `Password` to `Context`
- Modify: `internal/servers/cockpit_srv/config_handler.go` — add `Secured bool` to `ContextResponse`
- Modify: `internal/servers/cockpit_srv/credentials_handler.go` — validate session token when context has auth
- Create: `internal/servers/cockpit_srv/auth_handler.go` — session store + login endpoint
- Create: `internal/servers/cockpit_srv/auth_handler_test.go` — tests
- Modify: `internal/servers/cockpit_srv/server.go` — add `sessions` field to `CockpitServer`, register `/api/v1/auth`

- [ ] **Step 1: Add `Username` and `Password` to `ui.Context`**

In `internal/ui/config.go`, update the `Context` struct:

```go
type Context struct {
	Name        string `yaml:"name"`
	Target      string `yaml:"target"`
	Grafana     string `yaml:"grafana"`
	Credentials string `yaml:"credentials"`
	RootCA      string `yaml:"rootCA"`
	TlsCert     string `yaml:"tlsCert"`
	TlsKey      string `yaml:"tlsKey"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}
```

- [ ] **Step 2: Expose `Secured` flag in context response**

In `internal/servers/cockpit_srv/config_handler.go`, update `ContextResponse` and its mapping:

```go
type ContextResponse struct {
	Name    string `json:"name"`
	Target  string `json:"target"`
	Grafana string `json:"grafana"`
	Secured bool   `json:"secured"`
}
```

In `configHandler`, update the mapping loop:

```go
for i, ctx := range s.opts.Config.Contexts {
	response.Contexts[i] = ContextResponse{
		Name:    ctx.Name,
		Target:  ctx.Target,
		Grafana: ctx.Grafana,
		Secured: ctx.Username != "",
	}
}
```

- [ ] **Step 3: Write auth handler tests**

Create `internal/servers/cockpit_srv/auth_handler_test.go`:

```go
package cockpit_srv

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxthom/mir/internal/ui"
	"github.com/rs/zerolog"
)

func TestAuthHandler_Success(t *testing.T) {
	server := &CockpitServer{
		log:      zerolog.Nop(),
		sessions: newSessionStore(),
		opts: Options{
			Config: ui.Config{
				Contexts: []ui.Context{
					{Name: "prod", Username: "admin", Password: "secret"},
				},
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"context": "prod", "username": "admin", "password": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.authHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp loginResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthHandler_InvalidPassword(t *testing.T) {
	server := &CockpitServer{
		log:      zerolog.Nop(),
		sessions: newSessionStore(),
		opts: Options{
			Config: ui.Config{
				Contexts: []ui.Context{
					{Name: "prod", Username: "admin", Password: "secret"},
				},
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"context": "prod", "username": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.authHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_ContextNotFound(t *testing.T) {
	server := &CockpitServer{
		log:      zerolog.Nop(),
		sessions: newSessionStore(),
		opts:     Options{Config: ui.Config{Contexts: []ui.Context{}}},
	}

	body, _ := json.Marshal(map[string]string{"context": "missing", "username": "a", "password": "b"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.authHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAuthHandler_NoAuthRequired(t *testing.T) {
	server := &CockpitServer{
		log:      zerolog.Nop(),
		sessions: newSessionStore(),
		opts: Options{
			Config: ui.Config{
				Contexts: []ui.Context{
					{Name: "local", Target: "nats://localhost:4222"},
				},
			},
		},
	}

	body, _ := json.Marshal(map[string]string{"context": "local", "username": "", "password": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.authHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_MethodNotAllowed(t *testing.T) {
	server := &CockpitServer{log: zerolog.Nop(), sessions: newSessionStore(), opts: Options{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth", nil)
	w := httptest.NewRecorder()
	server.authHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestSessionStore_ValidateSuccess(t *testing.T) {
	s := newSessionStore()
	token := s.create("prod")
	if !s.validate(token, "prod") {
		t.Error("expected token to be valid")
	}
}

func TestSessionStore_WrongContext(t *testing.T) {
	s := newSessionStore()
	token := s.create("prod")
	if s.validate(token, "staging") {
		t.Error("expected token to be invalid for different context")
	}
}

func TestSessionStore_InvalidToken(t *testing.T) {
	s := newSessionStore()
	if s.validate("notareal token", "prod") {
		t.Error("expected invalid token to fail")
	}
}

func TestSessionStore_EmptyToken(t *testing.T) {
	s := newSessionStore()
	if s.validate("", "prod") {
		t.Error("expected empty token to fail")
	}
}
```

- [ ] **Step 4: Run tests to confirm they fail**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
go test ./internal/servers/cockpit_srv/... -run "TestAuth|TestSession" -v
```

Expected: compile error — `authHandler`, `loginResponse`, `newSessionStore`, `sessions` field undefined.

- [ ] **Step 5: Implement `auth_handler.go`**

Create `internal/servers/cockpit_srv/auth_handler.go`:

```go
package cockpit_srv

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const sessionTTL = 8 * time.Hour

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]sessionEntry
}

type sessionEntry struct {
	context   string
	expiresAt time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]sessionEntry)}
}

func (s *sessionStore) create(contextName string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.sessions {
		if now.After(v.expiresAt) {
			delete(s.sessions, k)
		}
	}
	s.sessions[token] = sessionEntry{context: contextName, expiresAt: now.Add(sessionTTL)}
	return token
}

func (s *sessionStore) validate(token, contextName string) bool {
	if token == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	return ok && sess.context == contextName && time.Now().Before(sess.expiresAt)
}

type loginRequest struct {
	Context  string `json:"context"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (s *CockpitServer) authHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var username, password string
	found := false
	for _, ctx := range s.opts.Config.Contexts {
		if ctx.Name == req.Context {
			username = ctx.Username
			password = ctx.Password
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "context not found", http.StatusNotFound)
		return
	}

	if username == "" {
		http.Error(w, "context does not require authentication", http.StatusBadRequest)
		return
	}

	if req.Username != username || req.Password != password {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := s.sessions.create(req.Context)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(loginResponse{Token: token}); err != nil {
		s.log.Error().Err(err).Msg("failed to encode auth response")
	}
}
```

- [ ] **Step 6: Add `sessions` field to `CockpitServer` and initialize it**

In `internal/servers/cockpit_srv/server.go`, update the `CockpitServer` struct:

```go
type CockpitServer struct {
	log      zerolog.Logger
	opts     Options
	store    mng.DashboardStore
	cache    *releasesCache
	sessions *sessionStore
}
```

Update `NewCockpit` to initialize `sessions`:

```go
return &CockpitServer{
	log:      logger.With().Str("srv", "cockpit_server").Logger(),
	opts:     *opts,
	store:    opts.Store,
	cache:    &releasesCache{},
	sessions: newSessionStore(),
}, nil
```

Register the auth route in `RegisterRoutes` (after the credentials route):

```go
authHandler := metricsMiddleware(http.HandlerFunc(s.authHandler))
authHandler = loggingMiddleware(s.log)(authHandler)
authHandler = securityHeadersMiddleware(authHandler)
authHandler = corsMiddleware(s.opts.AllowedOrigins)(authHandler)
mux.Handle("/api/v1/auth", authHandler)
```

- [ ] **Step 7: Add session validation to `credentialsHandler`**

In `internal/servers/cockpit_srv/credentials_handler.go`, update the context lookup to also capture `username`, then validate before reading the file:

```go
func (s *CockpitServer) credentialsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contextName := r.URL.Query().Get("context")
	if contextName == "" {
		contextName = s.opts.Config.CurrentContext
	}

	var credPath, username string
	found := false
	for _, ctx := range s.opts.Config.Contexts {
		if ctx.Name == contextName {
			credPath = ctx.Credentials
			username = ctx.Username
			found = true
			break
		}
	}

	if !found {
		http.Error(w, fmt.Sprintf("context %q not found", contextName), http.StatusNotFound)
		return
	}

	if username != "" {
		token := r.Header.Get("X-Session-Token")
		if !s.sessions.validate(token, contextName) {
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
```

- [ ] **Step 8: Run all cockpit tests**

```bash
go test ./internal/servers/cockpit_srv/... -v
```

Expected: all tests pass including `TestAuth*`, `TestSession*`, `TestCredentials*`, `TestConfig*`.

- [ ] **Step 9: Build to confirm no errors**

```bash
go build ./cmds/cockpit/... && go build ./internal/servers/cockpit_srv/...
```

Expected: exits 0.

- [ ] **Step 10: Commit**

```bash
git add internal/ui/config.go \
        internal/servers/cockpit_srv/auth_handler.go \
        internal/servers/cockpit_srv/auth_handler_test.go \
        internal/servers/cockpit_srv/config_handler.go \
        internal/servers/cockpit_srv/credentials_handler.go \
        internal/servers/cockpit_srv/server.go
git commit -m "feat(cockpit): per-context basic auth with session tokens"
```

---

### Task 10: Frontend Login Flow

**Files:**
- Modify: `internal/ui/web/src/lib/domains/contexts/types/types.ts` — add `secured`
- Modify: `internal/ui/web/src/lib/shared/types/api.ts` — add auth types
- Create: `internal/ui/web/src/lib/domains/auth/stores/auth.svelte.ts` — per-context token store
- Create: `internal/ui/web/src/lib/domains/auth/services/auth.ts` — login service
- Create: `internal/ui/web/src/routes/login/+page.svelte` — login page
- Modify: `internal/ui/web/src/routes/+layout.svelte` — redirect secured contexts to login
- Modify: `internal/ui/web/src/lib/domains/contexts/services/contexts.ts` — pass token in `getCredentials`
- Modify: `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts` — pass token from auth store

- [ ] **Step 1: Add `secured` to Context type**

In `internal/ui/web/src/lib/domains/contexts/types/types.ts`:

```typescript
export type Context = {
	name: string;
	target: string;
	grafana: string;
	secured: boolean;
};
```

- [ ] **Step 2: Add auth types to `api.ts`**

In `internal/ui/web/src/lib/shared/types/api.ts`, append:

```typescript
export type AuthRequest = {
	context: string;
	username: string;
	password: string;
};

export type AuthResponse = {
	token: string;
};
```

- [ ] **Step 3: Create auth store**

Create `internal/ui/web/src/lib/domains/auth/stores/auth.svelte.ts`:

```typescript
class AuthStore {
	private tokens = $state<Record<string, string>>({});

	getToken(contextName: string): string | null {
		return this.tokens[contextName] ?? null;
	}

	setToken(contextName: string, token: string) {
		this.tokens = { ...this.tokens, [contextName]: token };
	}

	clearToken(contextName: string) {
		const { [contextName]: _, ...rest } = this.tokens;
		this.tokens = rest;
	}
}

export const authStore = new AuthStore();
```

- [ ] **Step 4: Create auth service**

Create `internal/ui/web/src/lib/domains/auth/services/auth.ts`:

```typescript
export const authService = {
	async login(context: string, username: string, password: string): Promise<string> {
		const response = await fetch('/api/v1/auth', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ context, username, password })
		});
		if (!response.ok) throw new Error('Invalid username or password');
		const data: { token: string } = await response.json();
		return data.token;
	}
};
```

- [ ] **Step 5: Create login page**

Create `internal/ui/web/src/routes/login/+page.svelte`:

```svelte
<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { authService } from '$lib/domains/auth/services/auth';
	import { authStore } from '$lib/domains/auth/stores/auth.svelte';
	import { Button } from '$lib/shared/components/shadcn/button/index.js';
	import { Input } from '$lib/shared/components/shadcn/input/index.js';
	import { Label } from '$lib/shared/components/shadcn/label/index.js';

	const contextName = $derived(page.url.searchParams.get('context') ?? '');
	let username = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let isLoading = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		isLoading = true;
		try {
			const token = await authService.login(contextName, username, password);
			authStore.setToken(contextName, token);
			goto('/');
		} catch {
			error = 'Invalid username or password';
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-background">
	<div class="w-full max-w-sm space-y-6 p-6">
		<div class="space-y-1">
			<h1 class="text-2xl font-semibold tracking-tight">Sign in</h1>
			<p class="text-sm text-muted-foreground">
				Connecting to <span class="font-mono font-medium">{contextName}</span>
			</p>
		</div>
		<form onsubmit={handleSubmit} class="space-y-4">
			<div class="space-y-2">
				<Label for="username">Username</Label>
				<Input
					id="username"
					type="text"
					bind:value={username}
					required
					autocomplete="username"
					disabled={isLoading}
				/>
			</div>
			<div class="space-y-2">
				<Label for="password">Password</Label>
				<Input
					id="password"
					type="password"
					bind:value={password}
					required
					autocomplete="current-password"
					disabled={isLoading}
				/>
			</div>
			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}
			<Button type="submit" class="w-full" disabled={isLoading}>
				{isLoading ? 'Signing in…' : 'Sign in'}
			</Button>
		</form>
	</div>
</div>
```

- [ ] **Step 6: Update layout to redirect secured contexts**

In `internal/ui/web/src/routes/+layout.svelte`, add the import and update the `$effect`:

Add to the script imports:
```typescript
import { authStore } from '$lib/domains/auth/stores/auth.svelte';
import { goto } from '$app/navigation';
```

Replace the existing `$effect` block (lines 71–76):

```typescript
$effect(() => {
	const ctx = contextStore.activeContext;
	if (!ctx) return;

	// Don't run connection logic on the login page itself (prevents redirect loop)
	if (page.url.pathname === '/login') return;

	if (ctx.secured && !authStore.getToken(ctx.name)) {
		untrack(() => goto(`/login?context=${encodeURIComponent(ctx.name)}`));
		return;
	}

	untrack(() => mirStore.connect(ctx));
});
```

- [ ] **Step 7: Pass token in `getCredentials`**

In `internal/ui/web/src/lib/domains/contexts/services/contexts.ts`, update `getCredentials` signature:

```typescript
import { api } from '../../../shared/services/api';
import type { ApiResponse, ContextsResponse } from '../../../shared/types/api';

export const contextService = {
	async getAll(): Promise<ApiResponse<ContextsResponse>> {
		return api.get<ContextsResponse>('/v1/contexts');
	},

	async getCredentials(contextName: string, token?: string | null): Promise<string | null> {
		const headers: HeadersInit = {};
		if (token) (headers as Record<string, string>)['X-Session-Token'] = token;
		const response = await fetch(
			`/api/v1/credentials?context=${encodeURIComponent(contextName)}`,
			{ headers }
		);
		if (response.status === 204) return null;
		if (!response.ok) throw new Error(`credentials request failed: ${response.status}`);
		const data: { creds: string } = await response.json();
		return data.creds;
	}
};
```

- [ ] **Step 8: Pass auth token from store in `mir.svelte.ts`**

In `internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts`, add the import and update the `getCredentials` call:

Add import:
```typescript
import { authStore } from '$lib/domains/auth/stores/auth.svelte';
```

Update the credentials line inside `connect()`:
```typescript
const creds = await contextService.getCredentials(ctx.name, authStore.getToken(ctx.name));
```

The full updated `connect()` method:

```typescript
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

		const opts: MirOptions = { maxReconnectAttempts: 0 };

		const creds = await contextService.getCredentials(ctx.name, authStore.getToken(ctx.name));
		if (creds !== null) {
			opts.authenticator = credsAuthenticator(new TextEncoder().encode(creds));
		}

		const mir = await Mir.connect('cockpit', wsUrl, opts);

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
```

- [ ] **Step 9: Type-check**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server/internal/ui/web
npm run check 2>&1 | grep -v "node_modules"
```

Expected: no new errors beyond the 3 pre-existing ones.

- [ ] **Step 10: Commit**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server
git add internal/ui/web/src/lib/domains/contexts/types/types.ts \
        internal/ui/web/src/lib/shared/types/api.ts \
        internal/ui/web/src/lib/domains/auth/ \
        internal/ui/web/src/routes/login/ \
        internal/ui/web/src/routes/+layout.svelte \
        internal/ui/web/src/lib/domains/contexts/services/contexts.ts \
        internal/ui/web/src/lib/domains/mir/stores/mir.svelte.ts
git commit -m "feat(cockpit): per-context login page and session token flow"
```

---

## Phase 2 Verification

- [ ] Add `username: admin` / `password: secret` to a context in `cockpit.yaml`
- [ ] Open `http://localhost:3021` — selecting the secured context redirects to `/login?context=<name>`
- [ ] Submit wrong password → error message shown, no redirect
- [ ] Submit correct password → redirected to `/`, NATS connects with credentials
- [ ] `GET /api/v1/credentials?context=<name>` without `X-Session-Token` header → 401
- [ ] `GET /api/v1/credentials?context=<name>` with valid token → 200 with creds
- [ ] Context without `username` set → no redirect, connects directly (backward compatible)
