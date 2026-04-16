# Telemetry Units Display in Cockpit

**Date:** 2026-04-16  
**Branch:** `cockpit/deploy`  
**Status:** Approved

---

## Context

The backend telemetry pipeline already returns a `units` array alongside `headers` in the `QueryTelemetry` proto response (field `repeated string units = 2`). The TypeScript SDK strips this field during response transformation — `QueryData` only carries `headers` and `rows`. As a result, no unit information reaches the UI.

The goal is to surface units in two places in every telemetry view:
- **Tooltip**: value + unit inline (`25.3 °C`)
- **Field toggle pills**: field name + unit in parentheses (`temperature (°C)`)

For the last-value widget, units appear as a small label beneath the large numeric value.

---

## Architecture

Three layers, each with a single responsibility:

```
Proto response (units[])
        ↓
1. SDK layer   — expose fieldUnits in QueryData
        ↓
2. Config layer — unit stored on ChartConfig entry
        ↓
3. Display layer — tooltip, pills, last-value tiles
```

---

## Layer 1 — SDK (`pkgs/web/src/server_tlm.ts`)

Extend `QueryData`:

```ts
export type QueryData = {
  headers: string[];
  fieldUnits: Record<string, string>;  // { "temperature": "°C", "humidity": "%" }
  rows: QueryRow[];
};
```

In `QueryTelemetry.request()`, build the map alongside `headers`:

```ts
return {
  headers: qt.headers,
  fieldUnits: Object.fromEntries(qt.headers.map((h, i) => [h, qt.units[i] ?? ''])),
  rows: qt.rows.map(...)
};
```

Empty-unit fields map to `""` — treated as "no unit" downstream. Error return path also gets `fieldUnits: {}`.

---

## Layer 2 — Config (`chart-utils.ts`)

Add `unit?: string` to the `ChartConfig` entry type:

```ts
export type ChartConfig = {
  [k in string]: {
    label?: string;
    icon?: Component;
    unit?: string;           // e.g. "°C", "%", "" = no unit
  } & ( ... color/theme ... );
};
```

All existing chartConfig objects stay valid — field is optional.

---

## Layer 3 — Display

### Tooltip (`chart-tooltip.svelte`)

Append unit after the formatted value:

```svelte
{item.value.toLocaleString()}{itemConfig?.unit ? ' ' + itemConfig.unit : ''}
```

Covers all chart variants (`tlm-single-slot-chart`, `tlm-slot-chart`) since both render via `TlmChart` → `ChartTooltip`.

### Field toggle pills (`tlm-field-toggles.svelte`)

Add `fieldUnits?: Record<string, string>` prop. Display `temperature (°C)` when unit non-empty, `temperature` otherwise. The off-screen measurement div (used for overflow width calculation) must also include the suffix so pill widths are measured accurately.

### Last-value tiles (`widget-tlm-last.svelte`)

Unit displayed as a small subdued label beneath the large value — keeps the tile readable:

```svelte
<p class="text-2xl font-bold ...">{formatValue(val)}</p>
{#if fieldUnits[selectedField]}
  <p class="mt-0.5 font-mono text-xs text-muted-foreground">{fieldUnits[selectedField]}</p>
{/if}
```

---

## Setting Units into ChartConfig

### `widget-telemetry.svelte` (dashboard chart widget)

During the per-device merge loop:

```ts
chartConfig[`${deviceName}_${field}`] = {
  label: `${field} - ${deviceName}`,
  color: getDeviceFieldColor(fieldIdx, devIdx),
  unit: queryData.fieldUnits[field] ?? ''
};
```

`fieldUnits` is keyed by raw field name (e.g. `"temperature"`) — correct before merge prefixes device names.

### `[deviceId]/telemetry/+page.svelte` (device telemetry page)

```ts
chartConfig[field] = {
  label: field,
  color: CHART_COLORS[idx % CHART_COLORS.length],
  unit: telemetryStore.queryData?.fieldUnits[field] ?? ''
};
```

No telemetry store changes needed — it already holds `QueryData`.

### `widget-tlm-last.svelte` (last-value widget)

No chartConfig. Stores `fieldUnits` directly as `$state`:

```ts
let fieldUnits = $state<Record<string, string>>({});
// After query:
fieldUnits = results[0]?.data.fieldUnits ?? {};
```

---

## Wiring `fieldUnits` to TlmFieldToggles

| Component | Source of `fieldUnits` |
|---|---|
| `tlm-single-slot-chart.svelte` | Derived from `chartConfig[field]?.unit` |
| `tlm-slot-chart.svelte` | Derived from `chartConfig[devices[0].name + '_' + field]?.unit` |
| `widget-tlm-last.svelte` | `$state` from `results[0].data.fieldUnits` |

---

## Files to Modify

| File | Change |
|---|---|
| `pkgs/web/src/server_tlm.ts` | Add `fieldUnits` to `QueryData`; build from proto in `QueryTelemetry.request()` |
| `internal/ui/web/src/lib/shared/components/shadcn/chart/chart-utils.ts` | Add `unit?: string` to `ChartConfig` |
| `internal/ui/web/src/lib/shared/components/shadcn/chart/chart-tooltip.svelte` | Append unit after value |
| `internal/ui/web/src/lib/domains/devices/components/telemetry/tlm-field-toggles.svelte` | Add `fieldUnits?` prop; show `field (unit)`; update off-screen measurement div |
| `internal/ui/web/src/lib/domains/devices/components/telemetry/tlm-single-slot-chart.svelte` | Derive `fieldUnits` from `chartConfig`; pass to `TlmFieldToggles` |
| `internal/ui/web/src/lib/domains/devices/components/telemetry/tlm-slot-chart.svelte` | Derive `fieldUnits` from `chartConfig` + `devices[0]`; pass to `TlmFieldToggles` |
| `internal/ui/web/src/lib/domains/dashboards/components/widget-telemetry.svelte` | Set `unit` on chartConfig entries during merge |
| `internal/ui/web/src/lib/domains/dashboards/components/widget-tlm-last.svelte` | Store `fieldUnits` state; show unit in tiles; pass to `TlmFieldToggles` |
| `internal/ui/web/src/routes/devices/[deviceId]/telemetry/+page.svelte` | Set `unit` on chartConfig entries from `telemetryStore.queryData` |

---

## Verification

1. Run `npm run check` inside `internal/ui/web/` — confirm 0 new type errors (3 pre-existing errors in unrelated files are expected)
2. Start infrastructure (`just infra`) and server (`mir serve`)
3. Start a swarm: `mir swarm --ids=power,weather`
4. Open the device telemetry page for a swarm device:
   - Field pills show `value (unit)` format
   - Hover tooltip shows `25.3 °C` (unit after value)
5. Open the dashboard, add a telemetry chart widget:
   - Same pill and tooltip behavior
   - Multi-device view: each `deviceName_field` pill shows the correct unit
6. Add a last-value widget:
   - Field pills show `field (unit)`
   - Each device tile shows the value with unit beneath it
7. Fields with no unit (empty string from proto) display with no unit suffix — no trailing space or empty parens
