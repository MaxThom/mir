# Web UI Implementation Plan for Mir IoT Hub

## Overview

Add a SvelteKit-based web interface that connects directly to NATS via WebSocket, with a minimal Go HTTP API for authentication and NATS credential issuance.

## Architecture

```
Browser (SvelteKit SPA)
    ├─→ TypeScript SDK (pkgs/web/)
    └─→ NATS.js WebSocket
         ↓
    ws://localhost:9222 (NATS WebSocket - already configured!)
         ↓
    NATS Message Bus
         ↓
    Mir Services (core, prototlm, protocmd, etc.)

Authentication Flow:
Browser → HTTP /api/credentials → Go WebAPI → NSC → NATS JWT+nkey → Browser
```

## Key Design Decisions

1. **Direct NATS Connection**: Browser connects to NATS WebSocket (port 9222 already configured in infra)
2. **Static Site**: SvelteKit with `adapter-static` - no Node.js runtime needed
3. **Security**: Go API issues short-lived (15min) NATS credentials, no app-level roles (NATS ACLs only)
4. **TypeScript SDK**: Mirrors Go ModuleSDK with fluent API (`mir.client().createDevice()`)
5. **Compression**: Implement zstd support for parity with Go SDK
6. **UI**: Shadcn-svelte components with TailwindCSS
7. **User Management**: Deferred - manual credential creation via NSC for now

## Project Structure

```
mir.server/
├── cmds/webapi/                    # NEW: HTTP API server
│   ├── main.go                     # Server entry + routes
│   ├── config.go                   # Configuration
│   └── handlers/
│       ├── credentials.go          # NATS credential issuance
│       ├── static.go              # Serve SvelteKit build
│       └── health.go              # Health check
│
├── web/                           # NEW: SvelteKit app
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.svelte     # Root layout
│   │   │   ├── +page.svelte       # Dashboard
│   │   │   ├── devices/           # Device management
│   │   │   ├── telemetry/         # Telemetry viewer
│   │   │   ├── commands/          # Command sender
│   │   │   └── events/            # Event log
│   │   └── lib/
│   │       ├── components/        # Shadcn-svelte components
│   │       ├── stores/            # Svelte stores
│   │       │   ├── connection.ts  # NATS connection state
│   │       │   └── devices.ts     # Device cache
│   │       └── api/
│   │           ├── http.ts        # HTTP client
│   │           └── nats.ts        # NATS wrapper
│   ├── svelte.config.js           # adapter-static
│   └── package.json
│
├── pkgs/web/                      # NEW: TypeScript SDK
│   ├── src/
│   │   ├── mir.ts                 # Core Mir class
│   │   ├── client.ts              # Client routes
│   │   ├── device.ts              # Device routes
│   │   ├── event.ts               # Event routes
│   │   ├── compression.ts         # zstd support
│   │   ├── headers.ts             # NATS header constants
│   │   └── proto/                 # Generated protobuf types
│   ├── package.json
│   └── tsconfig.json
│
├── internal/webapi/               # NEW: Shared API code
│   ├── credentials/
│   │   ├── service.go             # Credential issuance
│   │   ├── nsc.go                 # NSC wrapper
│   │   └── cache.go               # Credential cache
│   └── middleware/
│       ├── cors.go                # CORS middleware
│       └── ratelimit.go           # Rate limiting
│
├── buf.gen.web.yaml              # NEW: Protobuf gen for TS
└── justfile                      # UPDATE: Add web commands
```

## Implementation Steps

### Phase 1: TypeScript SDK Foundation

**1.1 Setup TypeScript Project**
- Create `pkgs/web/` directory
- Initialize package.json with dependencies: nats.ws, @bufbuild/protobuf, zstd-wasm
- Setup tsconfig.json for ES modules
- Create buf.gen.web.yaml for protobuf generation

**Files to create:**
- `pkgs/web/package.json`
- `pkgs/web/tsconfig.json`
- `buf.gen.web.yaml`

**1.2 Generate Protobuf Types**
```bash
# buf.gen.web.yaml uses @bufbuild/protobuf plugin
buf generate --template buf.gen.web.yaml
```
Outputs TypeScript types to `pkgs/web/src/proto/mir_api/v1/`

**1.3 Implement Core Mir Class**

File: `pkgs/web/src/mir.ts`
```typescript
export class Mir {
  private conn: NatsConnection
  private instanceName: string

  static async connect(name: string, servers: string[], jwt: string, nkeySeed: string): Promise<Mir>

  client(): ClientRoutes
  device(): DeviceRoutes
  event(): EventRoutes

  async disconnect(): Promise<void>
}
```

**1.4 Implement Header Management**

File: `pkgs/web/src/headers.ts`
- Constants: `MIR_TRIGGER_CHAIN`, `MIR_MSG`, `MIR_CONTENT_ENCODING`, `MIR_TIME`
- Helper functions to parse/set headers

**1.5 Implement Compression**

File: `pkgs/web/src/compression.ts`
- Use `zstd-wasm` or `@oneidentity/zstd-js` for browser
- `compressData(data: Uint8Array): Uint8Array`
- `decompressData(data: Uint8Array): Uint8Array`
- Auto-detect compressed responses via header

**1.6 Implement Client Routes**

File: `pkgs/web/src/client.ts`

Mirror Go ModuleSDK patterns:
```typescript
export class ClientRoutes {
  async createDevice(req: CreateDeviceRequest): Promise<Device>
  async listDevices(req: ListDevicesRequest): Promise<Device[]>
  async updateDevice(req: UpdateDeviceRequest): Promise<Device>
  async deleteDevice(req: DeleteDeviceRequest): Promise<void>

  async listTelemetry(req: ListTelemetryRequest): Promise<TelemetryItem[]>

  async sendCommand(req: SendCommandRequest): Promise<CommandResponse>
  async listCommands(req: ListCommandsRequest): Promise<CommandSpec[]>

  async sendConfig(req: SendConfigRequest): Promise<ConfigResponse>
  async listConfigs(req: ListConfigsRequest): Promise<ConfigSpec[]>

  async listEvents(req: ListEventsRequest): Promise<Event[]>
}
```

Subject patterns match Go SDK:
- `client.{instanceId}.core.v1alpha.{function}`
- `client.{instanceId}.tlm.v1alpha.{function}`
- `client.{instanceId}.cmd.v1alpha.{function}`
- `client.{instanceId}.cfg.v1alpha.{function}`
- `client.{instanceId}.evt.v1alpha.{function}`

**1.7 Implement Device & Event Routes**

File: `pkgs/web/src/device.ts`
```typescript
export class DeviceRoutes {
  subscribeHeartbeat(deviceId: string, handler: HeartbeatHandler): Subscription
  subscribeTelemetry(deviceId: string, handler: TelemetryHandler): Subscription
  subscribeReportedProperties(deviceId: string, handler: PropertiesHandler): Subscription
}
```

File: `pkgs/web/src/event.ts`
```typescript
export class EventRoutes {
  subscribeAll(handler: EventHandler): Subscription
  subscribeDeviceOnline(handler: DeviceEventHandler): Subscription
  subscribeDeviceOffline(handler: DeviceEventHandler): Subscription
  subscribeDeviceCreated(handler: DeviceEventHandler): Subscription
  subscribeDeviceUpdated(handler: DeviceEventHandler): Subscription
  subscribeDeviceDeleted(handler: DeviceEventHandler): Subscription
  subscribeCommand(handler: CommandEventHandler): Subscription
  subscribeDesiredProperties(handler: PropertyEventHandler): Subscription
  subscribeReportedProperties(handler: PropertyEventHandler): Subscription
}
```

### Phase 2: Go WebAPI Server

**2.1 Create WebAPI Server**

File: `cmds/webapi/main.go`
```go
package main

func main() {
    cfg := loadConfig()

    // Setup HTTP server
    mux := http.NewServeMux()

    // Health endpoint
    mux.HandleFunc("/api/health", healthHandler)

    // Credential endpoint
    credSvc := credentials.NewService(cfg.NSC)
    mux.HandleFunc("POST /api/credentials", credentialHandler(credSvc))

    // Serve static SvelteKit build
    mux.Handle("/", serveStatic())

    // Add middleware
    handler := middleware.CORS(mux)
    handler = middleware.RateLimit(handler)

    // HTTP/2 h2c server (match existing pattern)
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", cfg.Port),
        Handler: h2c.NewHandler(handler, &http2.Server{}),
    }

    server.ListenAndServe()
}
```

File: `cmds/webapi/config.go`
```go
type Config struct {
    Port        int
    NSC         NSCConfig
    RateLimit   RateLimitConfig
    AllowOrigins []string
}

type NSCConfig struct {
    Operator    string
    Account     string
    StoreDir    string
}
```

**2.2 Implement Credential Issuance**

File: `internal/webapi/credentials/nsc.go`
```go
type Service struct {
    operator string
    account  string
    storeDir string
    cache    *CredentialCache
}

type Credentials struct {
    JWT       string    `json:"jwt"`
    NKeySeed  string    `json:"nkey_seed"`
    ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) IssueWebClientCredentials(username string) (*Credentials, error) {
    // 1. Check cache
    if cached := s.cache.Get(username); cached != nil {
        return cached, nil
    }

    // 2. Generate credentials via NSC
    // Execute: nsc add user -a mir -n {username} --expiry 15m --allow-pubsub _INBOX.> --allow-pub client.*.core.v1alpha.* ...
    cmd := exec.Command("nsc", "add", "user",
        "-a", s.account,
        "-n", username,
        "--expiry", "15m",
        s.getWebClientScopes()...)

    // 3. Generate .creds file
    credsCmd := exec.Command("nsc", "generate", "creds", "-a", s.account, "-n", username)
    credsData, err := credsCmd.CombinedOutput()
    if err != nil {
        return nil, err
    }

    // 4. Parse .creds (contains JWT + nkey seed)
    creds := parseCredsFile(credsData)

    // 5. Cache with TTL
    s.cache.Set(username, creds, 14*time.Minute) // Refresh 1 min before expiry

    return creds, nil
}

func (s *Service) getWebClientScopes() []string {
    return []string{
        "--allow-pubsub", "_INBOX.>",
        "--allow-pub", "client.*.core.v1alpha.*",
        "--allow-pub", "client.*.tlm.v1alpha.*",
        "--allow-pub", "client.*.cmd.v1alpha.*",
        "--allow-pub", "client.*.cfg.v1alpha.*",
        "--allow-pub", "client.*.evt.v1alpha.*",
        "--allow-sub", "event.*.core.v1alpha.>",
        "--allow-sub", "event.*.cmd.v1alpha.>",
        "--allow-sub", "event.*.cfg.v1alpha.>",
        "--allow-sub", "device.*.tlm.v1alpha.proto",
        "--allow-sub", "device.*.cfg.v1alpha.proto",
        "--allow-sub", "device.*.core.v1alpha.hearthbeat",
    }
}
```

File: `internal/webapi/credentials/cache.go`
```go
type CredentialCache struct {
    mu    sync.RWMutex
    items map[string]*cacheItem
}

type cacheItem struct {
    creds     *Credentials
    expiresAt time.Time
}

func (c *CredentialCache) Get(key string) *Credentials
func (c *CredentialCache) Set(key string, creds *Credentials, ttl time.Duration)
func (c *CredentialCache) cleanup() // Background goroutine
```

**2.3 Implement HTTP Handler**

File: `cmds/webapi/handlers/credentials.go`
```go
func credentialHandler(svc *credentials.Service) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // For now, no auth - just generate credentials for a generic "web-user"
        // TODO: Add authentication layer later

        username := "web-user-" + generateRandomID()

        creds, err := svc.IssueWebClientCredentials(username)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(creds)
    }
}
```

**2.4 Static File Serving**

File: `cmds/webapi/handlers/static.go`
```go
import "embed"

//go:embed ../../web/build/*
var staticFiles embed.FS

func serveStatic() http.Handler {
    fs, _ := fs.Sub(staticFiles, "web/build")
    fileServer := http.FileServer(http.FS(fs))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // SPA fallback: serve index.html for non-API routes
        if !strings.HasPrefix(r.URL.Path, "/api/") {
            if _, err := fs.Open(strings.TrimPrefix(r.URL.Path, "/")); err != nil {
                r.URL.Path = "/"
            }
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

**2.5 Middleware**

File: `internal/webapi/middleware/cors.go`
```go
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if contains(allowedOrigins, origin) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
            }

            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusOK)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

File: `internal/webapi/middleware/ratelimit.go`
```go
type RateLimiter struct {
    mu      sync.Mutex
    clients map[string]*rate.Limiter
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)
        limiter := rl.getLimiter(ip)

        if !limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Phase 3: SvelteKit Application

**3.1 Initialize SvelteKit**

```bash
cd mir.server
npm create svelte@latest web
# Choose: Skeleton project, TypeScript, ESLint, Prettier

cd web
npm install @sveltejs/adapter-static
npx shadcn-svelte@latest init
```

File: `web/svelte.config.js`
```javascript
import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: true
    })
  }
};
```

File: `web/vite.config.ts`
```typescript
export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:3020',
        changeOrigin: true
      }
    }
  }
});
```

**3.2 NATS Connection Store**

File: `web/src/lib/stores/connection.ts`
```typescript
import { writable } from 'svelte/store';
import { Mir } from '@mir/web-sdk';

type ConnectionState =
  | { status: 'disconnected' }
  | { status: 'connecting' }
  | { status: 'connected'; mir: Mir }
  | { status: 'error'; error: string };

export const connection = writable<ConnectionState>({ status: 'disconnected' });

let refreshTimer: number;

export async function connectToMir() {
  connection.set({ status: 'connecting' });

  try {
    // Get NATS credentials from HTTP API
    const resp = await fetch('/api/credentials', { method: 'POST' });
    const { jwt, nkey_seed, expires_at } = await resp.json();

    // Connect to NATS WebSocket
    const mir = await Mir.connect(
      'web-client',
      ['ws://localhost:9222'],
      jwt,
      nkey_seed
    );

    connection.set({ status: 'connected', mir });

    // Auto-refresh credentials before expiry
    const expiresAt = new Date(expires_at).getTime();
    const refreshIn = expiresAt - Date.now() - 60000; // 1 min before expiry
    refreshTimer = setTimeout(() => reconnectWithNewCredentials(mir), refreshIn);

  } catch (error) {
    connection.set({ status: 'error', error: error.message });
  }
}

async function reconnectWithNewCredentials(oldMir: Mir) {
  await oldMir.disconnect();
  await connectToMir();
}

export function disconnect() {
  clearTimeout(refreshTimer);
  connection.update(state => {
    if (state.status === 'connected') {
      state.mir.disconnect();
    }
    return { status: 'disconnected' };
  });
}
```

**3.3 Device Store**

File: `web/src/lib/stores/devices.ts`
```typescript
import { writable, derived } from 'svelte/store';
import { connection } from './connection';
import type { Device } from '@mir/web-sdk/proto';

export const devices = writable<Device[]>([]);

export async function loadDevices() {
  const conn = get(connection);
  if (conn.status !== 'connected') return;

  const deviceList = await conn.mir.client().listDevices({
    targets: { ids: [], labels: {}, annotations: {} }
  });

  devices.set(deviceList);
}

export async function createDevice(device: Device) {
  const conn = get(connection);
  if (conn.status !== 'connected') return;

  const created = await conn.mir.client().createDevice({ device });
  devices.update(list => [...list, created]);
}

// Subscribe to device events for live updates
export function subscribeToDeviceEvents() {
  const conn = get(connection);
  if (conn.status !== 'connected') return;

  conn.mir.event().subscribeDeviceCreated((msg, event) => {
    devices.update(list => [...list, event.device]);
  });

  conn.mir.event().subscribeDeviceDeleted((msg, event) => {
    devices.update(list => list.filter(d => d.id !== event.deviceId));
  });

  conn.mir.event().subscribeDeviceUpdated((msg, event) => {
    devices.update(list => list.map(d =>
      d.id === event.device.id ? event.device : d
    ));
  });
}
```

**3.4 Root Layout**

File: `web/src/routes/+layout.svelte`
```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { connection, connectToMir } from '$lib/stores/connection';
  import '../app.css';

  let mounted = false;

  onMount(() => {
    mounted = true;
    connectToMir();
  });
</script>

{#if $connection.status === 'connected'}
  <div class="app">
    <nav>
      <a href="/">Dashboard</a>
      <a href="/devices">Devices</a>
      <a href="/telemetry">Telemetry</a>
      <a href="/commands">Commands</a>
      <a href="/events">Events</a>
    </nav>

    <main>
      <slot />
    </main>
  </div>
{:else if $connection.status === 'connecting'}
  <div class="loading">Connecting to Mir...</div>
{:else if $connection.status === 'error'}
  <div class="error">
    Error: {$connection.error}
    <button on:click={connectToMir}>Retry</button>
  </div>
{/if}
```

**3.5 Device List Page**

File: `web/src/routes/devices/+page.svelte`
```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { devices, loadDevices, subscribeToDeviceEvents } from '$lib/stores/devices';
  import DeviceCard from '$lib/components/DeviceCard.svelte';
  import { Button } from '$lib/components/ui/button';

  onMount(() => {
    loadDevices();
    subscribeToDeviceEvents();
  });
</script>

<div class="devices-page">
  <header>
    <h1>Devices</h1>
    <Button on:click={() => goto('/devices/create')}>Create Device</Button>
  </header>

  <div class="device-grid">
    {#each $devices as device (device.id)}
      <DeviceCard {device} />
    {/each}
  </div>
</div>
```

**3.6 Telemetry Viewer**

File: `web/src/routes/telemetry/+page.svelte`
```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { connection } from '$lib/stores/connection';
  import TelemetryChart from '$lib/components/TelemetryChart.svelte';
  import { writable } from 'svelte/store';

  const telemetryData = writable([]);
  let selectedDevice = null;

  onMount(() => {
    const conn = get(connection);
    if (conn.status !== 'connected') return;

    // Subscribe to live telemetry
    conn.mir.device().subscribeTelemetry('*', (msg, deviceId, data) => {
      telemetryData.update(items => [...items, { deviceId, data, timestamp: Date.now() }]);
    });
  });
</script>

<div class="telemetry-page">
  <h1>Telemetry</h1>
  <TelemetryChart data={$telemetryData} />
</div>
```

**3.7 Event Log**

File: `web/src/routes/events/+page.svelte`
```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { connection } from '$lib/stores/connection';
  import EventLog from '$lib/components/EventLog.svelte';
  import { writable } from 'svelte/store';

  const events = writable([]);

  onMount(() => {
    const conn = get(connection);
    if (conn.status !== 'connected') return;

    // Subscribe to all events
    conn.mir.event().subscribeAll((msg, eventId, event) => {
      events.update(list => [event, ...list].slice(0, 100)); // Keep last 100
    });
  });
</script>

<div class="events-page">
  <h1>System Events</h1>
  <EventLog {events} />
</div>
```

### Phase 4: Build Integration

**4.1 Update Justfile**

File: `justfile` (append these targets)
```makefile
# TypeScript SDK
web-sdk-install:
    cd pkgs/web && npm install

web-sdk-build: web-sdk-install
    cd pkgs/web && npm run build

# SvelteKit
web-install:
    cd web && npm install

web-dev: web-install
    cd web && npm run dev

web-build: web-install
    cd web && npm run build

# WebAPI
build-webapi:
    go build -ldflags="{{ld_flags}}" -o bin/webapi cmds/webapi/main.go

run-webapi:
    go run cmds/webapi/main.go serve

# Full web stack
web-stack: infra
    @echo "Starting Mir services..."
    just run-core &
    just run-prototlm &
    just run-protocmd &
    just run-protocfg &
    just run-eventstore &
    sleep 2
    @echo "Starting WebAPI..."
    just run-webapi &
    sleep 1
    @echo "Starting SvelteKit dev server..."
    just web-dev

# Production build
build-web-prod: web-sdk-build web-build
    go build -ldflags="{{ld_flags}} -s -w" -o bin/webapi cmds/webapi/main.go
```

**4.2 Create Dockerfile**

File: `Dockerfile.webapi`
```dockerfile
# Stage 1: Build TypeScript SDK
FROM node:20-alpine AS sdk-builder
WORKDIR /build/pkgs/web
COPY pkgs/web/package*.json ./
RUN npm ci
COPY pkgs/web/ ./
RUN npm run build

# Stage 2: Build SvelteKit
FROM node:20-alpine AS web-builder
WORKDIR /build/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
COPY --from=sdk-builder /build/pkgs/web/dist /build/pkgs/web/dist
RUN npm run build

# Stage 3: Build Go binary
FROM golang:1.24.5-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /build/web/build ./web/build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o webapi cmds/webapi/main.go

# Stage 4: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates nsc
COPY --from=go-builder /build/webapi /usr/local/bin/webapi
EXPOSE 3020
ENTRYPOINT ["webapi"]
CMD ["serve"]
```

**4.3 Docker Compose for Development**

File: `infra/compose/web/compose.yaml`
```yaml
services:
  webapi:
    build:
      context: ../../..
      dockerfile: Dockerfile.webapi
    ports:
      - "3020:3020"
    environment:
      - MIR_WEBAPI_PORT=3020
      - MIR_WEBAPI_NSC_OPERATOR=mir
      - MIR_WEBAPI_NSC_ACCOUNT=mir
      - MIR_WEBAPI_ALLOW_ORIGINS=http://localhost:5173
    volumes:
      - ~/.nsc:/root/.nsc:ro
    depends_on:
      - nats
```

### Phase 5: Testing & Documentation

**5.1 Unit Tests for TypeScript SDK**

File: `pkgs/web/src/mir.test.ts`
```typescript
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { Mir } from './mir';

describe('Mir SDK', () => {
  let mir: Mir;

  beforeAll(async () => {
    mir = await Mir.connect('test-client', ['ws://localhost:9222'], jwt, nkey);
  });

  afterAll(async () => {
    await mir.disconnect();
  });

  it('should list devices', async () => {
    const devices = await mir.client().listDevices({});
    expect(devices).toBeInstanceOf(Array);
  });

  it('should create device', async () => {
    const device = await mir.client().createDevice({
      device: { meta: { name: 'test-device', namespace: 'default' } }
    });
    expect(device.meta.name).toBe('test-device');
  });
});
```

**5.2 Integration Tests for WebAPI**

File: `cmds/webapi/handlers/credentials_test.go`
```go
func TestCredentialIssuance(t *testing.T) {
    svc := credentials.NewService(credentials.Config{
        Operator: "mir",
        Account:  "mir",
        StoreDir: "/tmp/nsc-test",
    })

    creds, err := svc.IssueWebClientCredentials("test-user")
    require.NoError(t, err)
    require.NotEmpty(t, creds.JWT)
    require.NotEmpty(t, creds.NKeySeed)

    // Verify creds work with NATS
    nc, err := nats.Connect("nats://localhost:4222", nats.UserJWT(
        func() (string, error) { return creds.JWT, nil },
        func(nonce []byte) ([]byte, error) {
            return signWithSeed(creds.NKeySeed, nonce)
        },
    ))
    require.NoError(t, err)
    defer nc.Close()
}
```

**5.3 Documentation**

File: `book/src/web_ui/overview.md`
```markdown
# Web UI

Mir provides a web-based interface for managing your IoT devices.

## Features
- Real-time device monitoring
- Telemetry visualization
- Command execution
- Configuration management
- Event logging

## Architecture
The web UI connects directly to NATS via WebSocket, enabling real-time updates without polling.

## Getting Started

### Development
\`\`\`bash
just web-stack
\`\`\`

Open http://localhost:5173

### Production
\`\`\`bash
just build-web-prod
./bin/webapi serve
\`\`\`

## TypeScript SDK
See [TypeScript SDK Guide](./typescript_sdk.md) for using the SDK in your own applications.
```

File: `pkgs/web/README.md`
```markdown
# Mir TypeScript SDK

Browser-compatible SDK for interacting with Mir IoT Hub via NATS.

## Installation
\`\`\`bash
npm install @mir/web-sdk
\`\`\`

## Usage
\`\`\`typescript
import { Mir } from '@mir/web-sdk';

// Connect
const mir = await Mir.connect('my-app', ['ws://localhost:9222'], jwt, nkey);

// List devices
const devices = await mir.client().listDevices({});

// Subscribe to events
mir.event().subscribeDeviceOnline((msg, event) => {
  console.log('Device online:', event.deviceId);
});

// Disconnect
await mir.disconnect();
\`\`\`

## API Reference
See [full documentation](../../book/src/web_ui/typescript_sdk.md).
```

## Critical Files

### Core Implementation Files:
1. **`pkgs/web/src/mir.ts`** - TypeScript SDK core, mirrors Go ModuleSDK
2. **`internal/webapi/credentials/nsc.go`** - NATS credential generation via NSC
3. **`web/src/lib/stores/connection.ts`** - NATS connection management with auto-refresh
4. **`cmds/webapi/main.go`** - HTTP server entry point and routing
5. **`cmds/webapi/handlers/static.go`** - Static file serving with SPA fallback
6. **`pkgs/web/src/compression.ts`** - zstd compression/decompression

### Configuration Files:
7. **`buf.gen.web.yaml`** - Protobuf generation for TypeScript
8. **`web/svelte.config.js`** - SvelteKit adapter-static configuration
9. **`Dockerfile.webapi`** - Multi-stage build for production

### Infrastructure:
10. **`justfile`** - Build commands for web stack

## Verification & Testing

### Step 1: Verify TypeScript SDK
```bash
cd pkgs/web
npm install
npm test
npm run build
```
Expected: All tests pass, dist/ directory created with compiled SDK

### Step 2: Verify WebAPI Server
```bash
# Start infrastructure
just infra

# Start WebAPI
just run-webapi
```
Expected: Server starts on :3020, health endpoint returns 200

Test credential endpoint:
```bash
curl -X POST http://localhost:3020/api/credentials
```
Expected: JSON response with `jwt`, `nkey_seed`, `expires_at`

### Step 3: Verify SvelteKit Build
```bash
cd web
npm install
npm run build
```
Expected: build/ directory created with index.html and assets

### Step 4: Integration Test
```bash
# Start full stack
just web-stack
```

Open http://localhost:5173 and verify:
- [ ] Connection status shows "connected"
- [ ] Device list loads
- [ ] Can create a device
- [ ] Real-time events appear in event log
- [ ] Telemetry streams update in real-time

### Step 5: Production Build
```bash
just build-web-prod
./bin/webapi serve
```

Open http://localhost:3020 and verify same functionality.

### Step 6: Docker Build
```bash
docker build -f Dockerfile.webapi -t mir-webapi .
docker run -p 3020:3020 mir-webapi
```

### Step 7: Load Test
Use swarm to create 100+ devices and verify:
- [ ] UI remains responsive
- [ ] NATS connection stable
- [ ] Credential refresh works
- [ ] No memory leaks

## Security Checklist

- [ ] NATS credentials expire after 15 minutes
- [ ] Credential auto-refresh 1 min before expiry
- [ ] Rate limiting on /api/credentials (20 req/min per IP)
- [ ] CORS configured for allowed origins only
- [ ] NSC permissions scope web users correctly
- [ ] No sensitive data in browser localStorage
- [ ] Credentials only in memory
- [ ] HTTPS enforced in production
- [ ] Security headers set (CSP, X-Frame-Options, etc.)

## Trade-offs & Future Work

### Current Limitations:
1. **No user authentication** - Deferred, all users get same NATS permissions
2. **No user management UI** - Manual NSC setup required
3. **Single-user mode** - All web sessions share permissions
4. **No persistent state** - Device list reloaded on refresh

### Future Enhancements:
1. Add proper user auth (username/password, OAuth, SAML)
2. User management UI with role assignment
3. Persist connection state across refreshes
4. Add more visualizations (graphs, maps, alerts)
5. Mobile-responsive design
6. Offline support with service workers
7. Export/import device configurations

## Development Timeline

- **Week 1**: TypeScript SDK + protobuf generation
- **Week 2**: Go WebAPI + credential issuance
- **Week 3-4**: SvelteKit UI (devices, telemetry, events, commands)
- **Week 5**: Build integration, Docker, testing
- **Week 6**: Documentation, polish, security review

Total: ~6 weeks for full implementation
