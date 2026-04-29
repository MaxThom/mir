# Echo — AI Integration for Mir

**Date:** 2026-04-28
**Status:** Approved for implementation

## Overview

Echo is an MCP (Model Context Protocol) server embedded in `mir serve`. It exposes Mir's IoT capabilities to AI clients (Claude Desktop, Cursor, etc.) via HTTP/SSE, enabling two use cases from a single tool:

1. **AI operator** — the AI manages the Mir fleet directly (list devices, send commands, read telemetry, update configs, query events)
2. **Developer assistant** — the AI helps developers build with Mir (query docs, explain CLI commands, scaffold projects, generate schemas)

## Architecture

### Placement

Echo is a new `internal/servers/echo_srv/` package, following the exact same pattern as `cockpit_srv`. It registers routes on the shared HTTP mux in `cmds/cockpit/main.go` — no new port, no new binary.

```go
// Existing
cockpit := cockpit_srv.NewCockpit(logger, &cockpitOpts)
cockpit.RegisterRoutes(mux)

// New
echo := echo_srv.NewEcho(logger, &echo_srv.Options{
    NatsConn: natsConn,
    Config:   uiConfig,
})
echo.RegisterRoutes(mux) // adds /mcp/sse and /mcp/message
```

### Package structure

```
internal/servers/echo_srv/
  ├── server.go              // EchoServer struct, Options, RegisterRoutes
  ├── tools.go               // tool registry, handler dispatch
  ├── tools_operational.go   // list_devices, get_device, send_command, …
  ├── tools_devassist.go     // query_docs, explain_cli_command, scaffold_device_project, …
  ├── docs_embed.go          // embeds book/src/** at build time
  └── sse.go                 // HTTP/SSE transport wiring
```

### Request flow

```
AI client (Claude / Cursor)
  → GET /mcp/sse            (open SSE stream)
  → POST /mcp/message       (tool call JSON-RPC)
  → echo_srv tool registry
      ├── operational tools  → NATS (existing connection)
      ├── doc tools          → embedded book/src FS
      └── CLI tools          → exec mir --help
```

## Transport

**Library:** `github.com/modelcontextprotocol/go-sdk` (official Anthropic + Google SDK, v1.x stable)

**Why:** v1 stable API, auto-schema inference from typed Go structs, maintained in sync with the MCP spec, OpenSSF security review, integrates as a plain `http.Handler`.

```go
s := mcp.NewServer(&mcp.Implementation{Name: "Echo", Version: "1.0.0"}, nil)
// ... register tools ...
handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server { return s }, nil)
mux.Handle("/mcp/", handler)
```

**Client config** (what the user adds to Claude Desktop / Cursor):
```json
{
  "mcpServers": {
    "mir": {
      "url": "http://your-mir-server/mcp/sse",
      "headers": { "Authorization": "Bearer <password>" }
    }
  }
}
```
Omit `headers` when running locally with no password configured.

## Tools

All tool inputs are typed Go structs; the SDK infers JSON schemas automatically.

### Operational tools (`tools_operational.go`)

| Tool | Inputs | Output |
|------|--------|--------|
| `list_devices` | `namespace?`, `name?` | `[]Device` |
| `get_device` | `name`, `namespace` | `Device` (full digital twin) |
| `create_device` | `name`, `namespace`, `schema?` | `Device` |
| `delete_device` | `name`, `namespace` | ok |
| `send_command` | `name`, `namespace`, `command`, `payload_json?` | response |
| `update_config` | `name`, `namespace`, `config`, `payload_json` | ok |
| `get_telemetry` | `name`, `namespace`, `field?`, `limit?` | `[]TlmPoint` |
| `list_events` | `namespace?`, `type?`, `limit?` | `[]Event` |

All operational tools use the existing NATS connection passed via `Options.NatsConn`. They call the same NATS subjects the CLI uses — no new protocol.

### Developer assistant tools (`tools_devassist.go`)

| Tool | Inputs | Mechanism |
|------|--------|-----------|
| `query_docs` | `query` | keyword search over embedded `book/src/` → returns top-3 matching sections |
| `list_cli_commands` | — | `exec mir --help` → returns command tree |
| `explain_cli_command` | `command` | `exec mir <command> --help` → returns live output |
| `scaffold_device_project` | `module_name` | `exec mir tools generate device-template <name>` |
| `generate_schema` | `device_name` | `exec mir tools generate mir-schema <name>` |

### Typed struct example

```go
type SendCommandInput struct {
    Name      string `json:"name"      jsonschema:"required,Device name"`
    Namespace string `json:"namespace"  jsonschema:"required,Namespace"`
    Command   string `json:"command"    jsonschema:"required,Command name"`
    Payload   string `json:"payload"    jsonschema:"JSON payload string"`
}

func (e *EchoServer) handleSendCommand(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input SendCommandInput,
) (*mcp.CallToolResult, any, error) {
    // ...
}
```

## Developer Assistant Knowledge Base

### Embedded docs

`book/src/**/*.md` is compiled into the binary at build time. Following the same pattern as `ui.CockpitBuildFS`, a new `book/embed.go` file at the repo root exports the FS:

```go
// book/embed.go
package book

import "embed"

//go:embed src
var DocsFS embed.FS
```

`DocsFS` is passed to echo_srv via `Options.DocsFS fs.FS`, the same way `WebFS` is passed to cockpit_srv. In `cmds/cockpit/main.go`:

```go
echo_srv.Options{
    DocsFS: book.DocsFS,
    // ...
}
```

`query_docs` walks `DocsFS`, scores each `.md` file by word overlap with the query, and returns the top-3 matching sections as raw markdown. No LLM call, no external service — fast and works offline.

### Live CLI introspection

`explain_cli_command` and `list_cli_commands` exec the `mir` binary at runtime. This guarantees the output always reflects the actually installed version — no stale docs problem.

```go
out, err := exec.CommandContext(ctx, "mir", append(strings.Fields(input.Command), "--help")...).Output()
```

## Security

Echo reuses the `Password` field already present on `ui.Config.Context` — no new config.

### Auth middleware

An `authMiddleware` wraps all `/mcp/` routes before the MCP handler:

```go
func authMiddleware(password string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if password == "" {
            // local mode — no password configured, pass through
            next.ServeHTTP(w, r)
            return
        }
        token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        if token != password {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Behaviour

| Context password | Request has valid token | Result |
|-----------------|------------------------|--------|
| empty (local) | — | allowed (local dev, no auth needed) |
| set | yes | allowed |
| set | no / missing | 401 Unauthorized |

### Client configuration

Claude Desktop and Cursor support `headers` at the MCP server level — the token is sent automatically on both the SSE connection and every tool call:

```json
{
  "mcpServers": {
    "mir": {
      "url": "http://your-mir-server/mcp/sse",
      "headers": { "Authorization": "Bearer <your-password>" }
    }
  }
}
```

## Options struct

```go
type Options struct {
    NatsConn *nats.Conn  // existing connection from mir serve
    Config   ui.Config   // CLI config with contexts (Password field used for auth)
    DocsFS   fs.FS       // embedded book/src FS, from book.DocsFS
}
```

Echo is disabled if `NatsConn` is nil (e.g., when running Cockpit standalone without NATS).

## Out of scope (v1)

- Built-in chat UI (future: embedded in Cockpit)
- Streaming telemetry subscriptions (future: MCP resources/subscriptions)
- Vector search over docs (future: replace keyword search with embeddings)
