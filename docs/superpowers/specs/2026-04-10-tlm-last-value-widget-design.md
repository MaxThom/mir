# Telemetry Last Value Widget — Design Spec

**Date:** 2026-04-10  
**Branch:** `cockpit/widget_tlm_last`

---

## Overview

Add a "Last Value" subtype to the existing Telemetry widget. Instead of a time-series line chart, this subtype shows the most recent reading per device as a grid of tiles — one tile per device, displaying the current value and a color-coded freshness badge. Ideal for fleet-level dashboards where you want to scan current readings at a glance.

---

## Display

**Layout:** Multi-device tile grid. Each tile represents one device.

**Per tile:**
- Device name
- Current value + unit
- Freshness badge (top-right corner):
  - Green `live` / `Xs` — last reading < 1 minute ago
  - Amber `Xm` — last reading 1–5 minutes ago
  - Dim (no badge text change) — last reading 5–60 minutes ago
  - Red `stale` — no data found in the 60-minute lookback window

**Field switcher:** Pill toggles at the top of the widget, one per available field in the measurement. Only one field active at a time. Switching updates all tiles immediately. Active field is persisted as view state.

**Offline / no-data devices:** Tile shown grayed out with `—` as the value.

---

## Data Model

One new optional field added to the existing `TelemetryWidgetConfig`:

```typescript
interface TelemetryWidgetConfig {
  // --- existing fields (unchanged) ---
  target: DeviceTargetConfig;
  measurement: string;
  fields: string[];          // all available fields from the measurement
  timeMinutes: number;       // reused as lookback window (default: 60)
  selectedFields?: string[];
  splitCount?: 1 | 2 | 3 | 4;
  syncFields?: boolean;
  enabledDeviceIds?: string[];

  // --- new ---
  subtype?: 'chart' | 'last-value';  // absent = 'chart' (backwards compatible)
  selectedField?: string;             // active field in last-value display (view state)
}
```

`timeMinutes` doubles as the lookback window: 60 minutes = stale threshold. No new config fields required for the staleness logic.

---

## Wizard Flow

### New step added to the step machine

```typescript
type Step = 'type' | 'tlm-sub' | 'device-sub' | 'target' | 'config';
```

### Routing

```
User clicks "Telemetry"
  → step = 'tlm-sub'   (NEW — mirrors 'device-sub' for the device widget)

User clicks "Line Chart" in tlm-sub
  → selectedSubtype = 'chart'
  → step = 'target'

User clicks "Last Value" in tlm-sub
  → selectedSubtype = 'last-value'
  → step = 'target'

target → config (same as before)
```

### Config step (step 4) for Last Value

1. Load measurements via `mir.client().listTelemetry()` (same SDK call as chart)
2. User picks a measurement from the grouped list (same UI as chart)
3. **New:** User picks a default field from the measurement's `fields[]` list (pill selection)
4. `buildConfig()` writes `subtype: 'last-value'` and `selectedField: chosenField`

### Edit mode

Telemetry widgets with `subtype === 'last-value'` re-enter the wizard at `'tlm-sub'` (same as device widgets re-enter at `'device-sub'`).

---

## Component Architecture

### Dispatch — `dashboard-grid.svelte`

Add a condition where widget components are resolved:

```
type === 'telemetry' && config.subtype === 'last-value'  →  <WidgetTlmLast>
type === 'telemetry' (default / 'chart' / absent)        →  <WidgetTelemetry>   ← unchanged
```

### New component — `widget-tlm-last.svelte`

**Location:** `internal/ui/web/src/lib/domains/dashboards/components/widget-tlm-last.svelte`

**Responsibilities:**
- Accept same props as `widget-telemetry.svelte` (`config: TelemetryWidgetConfig`, `widgetId`, edit mode flags)
- On mount: call `queryTelemetry()` with a 60-min window and no aggregation window, take the last row per device
- Render tile grid (CSS grid, responsive columns)
- Field pill switcher at top — iterates `config.fields`, highlights `selectedField`
- On field toggle: update `selectedField`, save to dashboard view state via `dashboardStore.saveWidgetViewState()`
- Freshness badge: compute age from last row's `_time` timestamp vs `Date.now()`
- Auto-refresh: `setInterval` every 30 seconds, cleared on component destroy
- Grayed-out tile for devices with no rows in the query result

### `widget-telemetry.svelte` — **zero changes**

---

## Data Fetching

Reuses existing `queryTelemetry()` SDK call:

```typescript
const data = await mir.client().queryTelemetry().request(
  target,
  measurement,
  fields,          // all fields — need values for active field + timestamps
  start,           // Date.now() - 60 minutes
  end,             // Date.now()
  undefined        // no aggregation — raw rows, last one is the latest
);
```

Last value per device: take the last row in `data.rows` where `_device === deviceId`. The `_time` field of that row drives the freshness badge.

---

## Verification

1. **Widget creation:** Open dashboard → Add Widget → select Telemetry → confirm "Line Chart / Last Value" step appears → select Last Value → complete wizard → widget appears on grid as tile grid
2. **Field switcher:** Pill toggles switch all tiles to show the selected field; active field persists on page reload
3. **Freshness badges:** Start a device swarm, observe green badges; stop reporting, watch badges go amber then stale after 60 min (or mock by querying with a device that has no recent data)
4. **Edit mode:** Edit the widget → wizard re-enters at subtype step → can switch back to Line Chart
5. **Line chart unaffected:** Create a standard telemetry (line chart) widget, confirm it still works with no regressions
6. **Type check:** Run `npm run check` inside `internal/ui/web/` — must pass (3 pre-existing errors in unrelated files are expected)
