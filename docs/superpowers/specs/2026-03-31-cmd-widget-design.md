# Command Widget Design

**Date:** 2026-03-31
**Branch:** cockpit/cmd_widget
**Status:** Approved

## Overview

Full rewrite of `widget-command.svelte` to match the feature quality of `widget-telemetry.svelte`. The widget targets N devices via `config.target`, loads their commands in parallel, presents them in a grouped left panel, and provides a JSON editor + inline response log on the right.

## Layout

```
┌──────────┬────────────────────────────────────┐
│ Commands │  command-name  [VIM] [COPY] [⛶]    │
│ (grouped)│  ┌──────────────────────────────┐  │
│          │  │  JSON editor (CodeMirror)    │  │
│  cmd-a   │  │                              │  │
│  cmd-b ◀ │  └──────────────────────────────┘  │
│  cmd-c   │  [Send] [Dry Run]                   │
│          │  ─────── responses ──────────────   │
│          │  ✓ device-1   42ms   {"ok":true}    │
│          │  ✗ device-2  timeout  error msg     │
└──────────┴────────────────────────────────────┘
```

- Left: `DescriptorPanel` in grouped mode
- Right: `JsonPayloadEditor` (top, flex-1) + response log (bottom, appears after first send)
- Device pills in the widget wrapper header via `onDevicesReady` callback
- Fullscreen: `fixed inset-0 z-50 bg-background`, toggled by button or Escape key

## Architecture

### Data Flow

1. On mount (when `mirStore.mir` is ready), call `listCommands()` in parallel for all resolved device IDs
2. Results are `CommandGroup[]` — each group has `ids[]` (devices sharing that schema) and `descriptors[]`
3. Feed groups to `DescriptorPanel` in grouped mode; label = comma-joined device names
4. Call `onDevicesReady` with flat `{id, name}[]` from all groups
5. User selects a command → set `selectedCommand` + `selectedGroupIdx`
6. Seed `JsonPayloadEditor` with `selectedCommand.template` JSON
7. On send → fan out to all devices in the selected group (see Send Flow)

### Config Type Change

`CommandWidgetConfig` in `dashboard-api.ts` gains one optional field:

```ts
export interface CommandWidgetConfig {
  target: DeviceTargetConfig;
  selectedCommand?: string;  // view state: restored after reload
}
```

### Component Reuse

No new sub-components. Everything reuses existing pieces:

| Component | Usage |
|-----------|-------|
| `DescriptorPanel` | Left panel, grouped mode (`groups`, `onSelectGrouped`, `selectedKey`) |
| `JsonPayloadEditor` | Right panel editor; `deviceValues` + `onSendMulti` for per-device mode |
| `activityStore` | Log every send result (existing pattern) |
| `WidgetDevicePills` | Populated via `onDevicesReady` in `dashboard-grid.svelte` |

## Multi-device Behavior

### Loading

- Resolve device IDs from `config.target` (ids / namespaces / labels) the same way `widget-telemetry.svelte` resolves `deviceInfos` — via a `listCommands()` call with a `DeviceTarget`
- The SDK returns `CommandGroup[]` where each group covers devices that share a command schema
- `onDevicesReady` fires after load with the flat list of all `{id, name}` pairs

### Send Flow

**Broadcast mode** (`onSend` — same payload to all devices in group):
```
Promise.allSettled(
  group.ids.map(id => mir.client().sendCommand().request(target(id), name, text, dryRun))
)
```

**Per-device mode** (`onSendMulti` — Map<deviceId, payload>):
```
Promise.allSettled(
  [...payloads.entries()].map(([id, text]) => mir.client().sendCommand().request(target(id), name, text, dryRun))
)
```

Both flows:
- Clear the previous response log before sending
- Append each result as it settles (success or error)
- Log each result to `activityStore`

## Response Log

Each entry:

```
✓ / ✗  device-name   <duration>ms   <truncated JSON or error>
```

- Icon: green check (success) or red X (error)
- Device name in monospace
- Duration in milliseconds
- Response: truncated to ~80 chars inline; full JSON revealed via `<details>` expand
- Log is hidden before first send; cleared on each new send
- Log is scrollable with `max-h` constraint so it doesn't push the editor off screen

## View State

`selectedCommand?: string` is saved to `CommandWidgetConfig` via `dashboardStore.saveWidgetViewState()` whenever the selected command changes (after initial load, using `untrack` pattern from tlm widget to avoid save loops).

On load, after commands resolve, re-select the matching descriptor by name if `config.selectedCommand` is set.

## Fullscreen

Same pattern as `widget-telemetry.svelte`:
- `let fullscreen = $state(false)`
- Outer div: `class="{fullscreen ? 'fixed inset-0 z-50 bg-background' : 'flex h-full flex-col'} flex flex-col"`
- `<svelte:window onkeydown>` → set `fullscreen = false` on Escape
- Fullscreen toggle button in the right-panel header row (above `JsonPayloadEditor`)

## Files to Change

| File | Change |
|------|--------|
| `internal/ui/web/src/lib/domains/dashboards/components/widget-command.svelte` | Full rewrite |
| `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts` | Add `selectedCommand?` to `CommandWidgetConfig` |
| `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte` | Add `onDevicesReady` prop to `<WidgetCommand>` |

## Out of Scope

- New sub-components (reuse only)
- Changes to `DescriptorPanel`, `JsonPayloadEditor`, or `activityStore`
- Authentication / permissions
- Command history persistence beyond the current session
