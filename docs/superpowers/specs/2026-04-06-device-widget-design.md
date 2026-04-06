# Device Widget Design

**Date:** 2026-04-06  
**Branch:** cockpit/widget_cfg  
**Status:** Approved

## Context

The Mir cockpit dashboard already has telemetry, command, config, and events widgets. The device Overview page (single device view) shows rich device data: meta (name, namespace, labels, annotations), spec (device ID, disabled flag), status (online, last heartbeat, schema), and desired/reported properties with drift detection. This widget brings that same data into the dashboard for multi-device monitoring — read-only, auto-refreshing, and switchable per device via header pill tabs.

## Widget Type

Single new widget type: `'device'`.  
One Svelte component: `widget-device.svelte`.  
Step 3 in the add-widget wizard lets the user pick which view to display.

## Config Shape

```typescript
interface DeviceWidgetConfig {
  target: DeviceTargetConfig   // same as all other widgets
  view: 'info' | 'properties'  // chosen in wizard step 3
  selectedDeviceId?: string    // view state — active pill tab
}
```

`selectedDeviceId` is persisted via `saveWidgetViewState()` (same pattern as `enabledDeviceIds` in the telemetry widget). Defaults to the first device in the resolved list on first load.

## Wizard

- **Step 1:** Add a "Device" button to the existing 4-button type grid (making it 5).
- **Step 2:** Existing `device-target-builder.svelte` — no changes.
- **Step 3 (device only):** Two large option cards:
  - **Info** — Meta / Spec / Status
  - **Properties** — Desired / Reported

## Component: `widget-device.svelte`

**Props** (same signature as all other widgets):
```typescript
{
  config: DeviceWidgetConfig
  widgetId: string
  onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void
  refreshTick?: number
}
```

**Data loading:**  
On mount and on each `refreshTick` change, call `mir.client().listDevices()` with the target. All required fields (meta, spec, status, properties) are returned in that single call — no extra requests needed.

**Header extra (via `widget-wrapper`'s `headerExtra` snippet):**  
Clickable device name pills rendered inline. Active device: indigo background. Inactive: gray. Clicking a pill sets `selectedDeviceId` and saves view state. If the target resolves to only one device, pills are hidden.

**Content — `view === 'info'` (Info view):**  
Compact always-visible key-value groups. No collapsing.

| Section | Fields |
|---------|--------|
| META | Name, Namespace, Labels (badges), Annotations (badges) |
| SPEC | Device ID (monospace, truncated), Disabled |
| STATUS | Online indicator (green/gray dot), Last heartbeat (`TimeTooltip`), Schema package names + last fetch time |

Section headers use the same `text-xs font-medium tracking-wide text-muted-foreground uppercase` style from the Overview page. Key column is fixed-width, value wraps.

**Content — `view === 'properties'` (Properties view):**  
Exact rendering logic from `device-properties-card.svelte`, without the Card wrapper:
- **Desired** section: each key in monospace + `TimeTooltip` for last-set timestamp + `JsonValue` for the value
- Separator
- **Reported** section: each key in monospace + `CircleCheckBigIcon` if value matches desired + `TimeTooltip` + `JsonValue`
- Empty state: "No properties configured."

Both views are scrollable within the widget content area.

**Auto-refresh:**  
Responds to `refreshTick` prop (same as telemetry widget). No manual refresh button needed — the dashboard-level refresh handles it.

**`editorPrefs.utc`:**  
Passed through to `TimeTooltip` calls for consistent UTC toggle behaviour.

## Registration Points

| File | Change |
|------|--------|
| `dashboard-api.ts` | Add `'device'` to `WidgetType`; add `DeviceWidgetConfig` interface |
| `dashboard-grid.svelte` | Add `{:else if widget.type === 'device'}<WidgetDevice .../>{/if}` |
| `add-widget-dialog.svelte` | Add "Device" button in step 1 type grid; add step 3 view-picker for `device` type |

## Reused Components & Utilities

- `widget-wrapper.svelte` — card shell, edit/remove buttons, `headerExtra` snippet
- `device-properties-card.svelte` — Properties view logic (copy rendering, drop Card wrapper)
- `TimeTooltip` — relative timestamps with UTC toggle
- `JsonValue` — property value rendering
- `CircleCheckBigIcon` — desired/reported match indicator
- `editorPrefs` store — UTC preference
- `saveWidgetViewState()` — persist `selectedDeviceId` without triggering full re-sync
- `DeviceTarget` / `mir.client().listDevices()` — data fetching

## Out of Scope

- Editing device data (read-only by design)
- Showing device events in this widget (separate `evt` widget planned)
- Inline schema/protobuf inspection

## Verification

1. Add a Device widget (Info view) targeting 2+ devices → pills appear, clicking switches the displayed device
2. Add a Device widget (Properties view) → desired/reported render correctly, ✓ icon shows when values match
3. Dashboard auto-refresh enabled → data updates without manual action
4. Single-device target → no pills shown
5. Dynamic target (namespace filter) → all matching devices appear as pills
6. `npm run check` inside `internal/ui/web/` passes with no new errors
