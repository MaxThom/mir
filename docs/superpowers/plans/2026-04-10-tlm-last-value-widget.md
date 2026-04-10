# Telemetry Last Value Widget — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Last Value" subtype to the telemetry widget that shows the most recent reading per device as a responsive tile grid with freshness badges.

**Architecture:** A new `'tlm-sub'` wizard step routes between "Line Chart" (existing) and "Last Value" (new). The subtype is encoded in `TelemetryWidgetConfig.subtype`. `dashboard-grid.svelte` dispatches to the new `widget-tlm-last.svelte` when `subtype === 'last-value'`; the original `widget-telemetry.svelte` is unchanged.

**Tech Stack:** SvelteKit + Svelte 5 (runes), TypeScript, Tailwind CSS, existing `@mir/sdk` `queryTelemetry()`, `shadcn-svelte` components.

---

## File Map

| Action | File |
|--------|------|
| Modify | `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts` |
| Modify | `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte` |
| Modify | `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte` |
| Create | `internal/ui/web/src/lib/domains/dashboards/components/widget-tlm-last.svelte` |

---

## Task 1: Extend TelemetryWidgetConfig

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts:10-20`

- [ ] **Step 1: Add `subtype` and `selectedField` to `TelemetryWidgetConfig`**

Replace the existing interface (lines 10–20):

```typescript
export interface TelemetryWidgetConfig {
	target: DeviceTargetConfig;
	measurement: string;
	fields: string[];
	timeMinutes: number;
	// View state (optional — absent in older saved dashboards, defaults applied on load)
	selectedFields?: string[];
	splitCount?: 1 | 2 | 3 | 4;
	syncFields?: boolean;
	enabledDeviceIds?: string[];
	subtype?: 'chart' | 'last-value';   // absent = 'chart' (backwards compatible)
	selectedField?: string;              // active field for last-value display (view state)
}
```

- [ ] **Step 2: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: no new errors (3 pre-existing errors in `nav-section.svelte` and `multi/telemetry` are expected).

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts
git commit -m "feat(dashboard): extend TelemetryWidgetConfig with subtype and selectedField"
```

---

## Task 2: Add tlm-sub wizard step

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte`

All edits are in the `<script>` block of this file.

- [ ] **Step 1: Update the `Step` type (line 78)**

Replace:
```typescript
type Step = 'type' | 'device-sub' | 'target' | 'config';
```
With:
```typescript
type Step = 'type' | 'tlm-sub' | 'device-sub' | 'target' | 'config';
```

- [ ] **Step 2: Add state vars for telemetry subtype (after the `// Device config` block, around line 106)**

After `let selectedDeviceView = $state<'info' | 'properties' | 'status'>('info');` add:

```typescript
// Telemetry subtype config
let selectedTlmSubtype = $state<'chart' | 'last-value'>('chart');
let selectedLastValueField = $state('');
```

- [ ] **Step 3: Pre-populate subtype state in edit mode**

In the `$effect(() => { if (open && editWidget) { ... } })` block (around line 51–76), the telemetry case currently reads:

```typescript
if (editWidget.type === 'telemetry') {
    const c = editWidget.config as TelemetryWidgetConfig;
    selectedMeasurement = c.measurement;
}
```

Replace it with:

```typescript
if (editWidget.type === 'telemetry') {
    const c = editWidget.config as TelemetryWidgetConfig;
    selectedMeasurement = c.measurement;
    selectedTlmSubtype = c.subtype ?? 'chart';
    selectedLastValueField = c.selectedField ?? '';
}
```

- [ ] **Step 4: Reset the new state vars in `reset()`**

In the `reset()` function (around line 355–380), after `selectedDeviceView = 'info';` add:

```typescript
selectedTlmSubtype = 'chart';
selectedLastValueField = '';
```

- [ ] **Step 5: Route telemetry through `tlm-sub` in `selectType()`**

Replace the `selectType` function (around line 382–386):

```typescript
function selectType(t: WidgetType) {
	selectedType = t;
	title = typeLabel(t);
	step = t === 'device' ? 'device-sub' : t === 'telemetry' ? 'tlm-sub' : 'target';
}
```

- [ ] **Step 6: Include subtype in `buildConfig()` for telemetry**

In `buildConfig()`, replace the `'telemetry'` case (around line 413–422):

```typescript
case 'telemetry': {
    const descriptor = measurementGroups
        .flatMap((g) => g.measurements)
        .find((m) => m.name === selectedMeasurement);
    return {
        target,
        measurement: selectedMeasurement,
        fields: descriptor?.fields ?? (editWidget?.config as TelemetryWidgetConfig)?.fields ?? [],
        timeMinutes: 60,
        subtype: selectedTlmSubtype,
        ...(selectedTlmSubtype === 'last-value' ? { selectedField: selectedLastValueField } : {})
    } satisfies TelemetryWidgetConfig;
}
```

- [ ] **Step 7: Add `Grid2x2Icon` import at the top of the script imports**

After the existing icon imports add:

```typescript
import Grid2x2Icon from '@lucide/svelte/icons/grid-2x2';
```

- [ ] **Step 8: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: no new errors.

- [ ] **Step 9: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte
git commit -m "feat(dashboard): add tlm-sub wizard step script — subtype state and routing"
```

---

## Task 3: Add tlm-sub wizard template

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte`

All edits are in the template (HTML) section.

- [ ] **Step 1: Update step description to include `tlm-sub`**

The description block is around line 469–472. Replace the entire `<Dialog.Description>` content:

```svelte
<Dialog.Description>
    {#if step === 'type'}Step 1 of {selectedType === 'text' ? '2' : '3'} — Choose widget type{/if}
    {#if step === 'tlm-sub'}Step 2 of 4 — Choose display type{/if}
    {#if step === 'device-sub'}Step 2 of 3 — Configure widget{/if}
    {#if step === 'target'}{editWidget ? 'Step 1 of 2' : (selectedType === 'text' || selectedType === 'events') ? 'Step 2 of 2' : selectedType === 'telemetry' ? 'Step 3 of 4' : 'Step 3 of 3'} — {selectedType === 'text' ? 'Name your widget' : 'Select devices'}{/if}
    {#if step === 'config'}{editWidget ? 'Step 2 of 2' : selectedType === 'telemetry' ? 'Step 4 of 4' : 'Step 3 of 3'} — Configure widget{/if}
</Dialog.Description>
```

- [ ] **Step 2: Add the `tlm-sub` step block after the `device-sub` block**

After the closing `{/if}` of the `{#if step === 'device-sub'}` block (around line 546), insert:

```svelte
<!-- Step 2 (telemetry only): Subtype picker -->
{#if step === 'tlm-sub'}
    <div class="flex flex-1 items-start justify-center">
        <div class="grid w-full max-w-2xl grid-cols-2 gap-4">
            <button
                onclick={() => { selectedTlmSubtype = 'chart'; title = 'Telemetry'; step = 'target'; }}
                class="flex flex-col items-center gap-4 rounded-xl border border-border p-10 text-center transition-colors hover:border-primary hover:bg-accent"
            >
                <ActivityIcon class="h-12 w-12 text-muted-foreground" />
                <div>
                    <p class="font-semibold">Line Chart</p>
                    <p class="mt-1 text-xs text-muted-foreground">Time-series visualization with field toggles</p>
                </div>
            </button>
            <button
                onclick={() => { selectedTlmSubtype = 'last-value'; title = 'Last Value'; step = 'target'; }}
                class="flex flex-col items-center gap-4 rounded-xl border border-border p-10 text-center transition-colors hover:border-primary hover:bg-accent"
            >
                <Grid2x2Icon class="h-12 w-12 text-muted-foreground" />
                <div>
                    <p class="font-semibold">Last Value</p>
                    <p class="mt-1 text-xs text-muted-foreground">Current reading per device as tiles</p>
                </div>
            </button>
        </div>
    </div>
    <div class="flex gap-2">
        <Button variant="outline" onclick={() => (step = 'type')}>Back</Button>
    </div>
{/if}
```

- [ ] **Step 3: Update the Back button in the target step to navigate to `tlm-sub` for telemetry**

In the target step's Back button (around line 636–640), replace:

```typescript
onclick={() => {
    if (editWidget) { open = false; reset(); }
    else if (selectedType === 'device' || selectedType === 'device-list') step = 'device-sub';
    else step = 'type';
}}
```

With:

```typescript
onclick={() => {
    if (editWidget) { open = false; reset(); }
    else if (selectedType === 'device' || selectedType === 'device-list') step = 'device-sub';
    else if (selectedType === 'telemetry') step = 'tlm-sub';
    else step = 'type';
}}
```

- [ ] **Step 4: Add field picker in the config step for last-value**

In the config step, after the measurement list section ends (the closing `{/if}` of the `{#if measurementGroups.length > 1}` info block, around line 719), and before the `{:else if selectedType === 'command'}` block, insert:

```svelte
{#if selectedTlmSubtype === 'last-value' && selectedMeasurement}
    {@const measFields = measurementGroups.flatMap((g) => g.measurements).find((m) => m.name === selectedMeasurement)?.fields ?? []}
    {#if measFields.length > 0}
        <div class="space-y-1">
            <p class="text-sm font-medium">Default field</p>
            <div class="flex flex-wrap gap-2">
                {#each measFields as field (field)}
                    <button
                        onclick={() => (selectedLastValueField = field)}
                        class="rounded-md border px-3 py-1.5 font-mono text-sm transition-colors
                            {selectedLastValueField === field
                            ? 'border-primary bg-primary text-primary-foreground'
                            : 'border-border hover:bg-accent'}"
                    >
                        {field}
                    </button>
                {/each}
            </div>
        </div>
    {/if}
{/if}
```

- [ ] **Step 5: Update the Save button disabled condition**

Replace the existing disabled condition on the Save button (around line 825–828):

```svelte
disabled={dashboardStore.isSaving ||
    (selectedType === 'telemetry' && !selectedMeasurement) ||
    (selectedType === 'telemetry' && selectedTlmSubtype === 'last-value' && !selectedLastValueField) ||
    (selectedType === 'command' && !selectedCommandName) ||
    (selectedType === 'config' && !selectedConfigName)}
```

- [ ] **Step 6: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: no new errors.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte
git commit -m "feat(dashboard): add tlm-sub wizard template — subtype step and last-value field picker"
```

---

## Task 4: Create widget-tlm-last.svelte

**Files:**
- Create: `internal/ui/web/src/lib/domains/dashboards/components/widget-tlm-last.svelte`

- [ ] **Step 1: Create the file**

```svelte
<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import type { TelemetryWidgetConfig } from '../api/dashboard-api';

	let {
		config,
		widgetId,
		refreshTick = 0,
		onDevicesReady
	}: {
		config: TelemetryWidgetConfig;
		widgetId: string;
		refreshTick?: number;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	// ─── State ────────────────────────────────────────────────────────────────

	let deviceInfos = $state<{ id: string; name: string }[]>([]);
	// Last row per device: deviceId → { values, timestamp }
	let lastRows = $state<Map<string, { values: Record<string, unknown>; timestamp: Date | null }>>(
		new Map()
	);
	let selectedField = $state(untrack(() => config.selectedField ?? config.fields[0] ?? ''));
	let hasLoaded = $state(false);
	let loadError = $state<string | null>(null);
	let generation = 0;

	// ─── Startup ─────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadAndQuery);
		} else {
			lastRows = new Map();
		}
	});

	// ─── Refresh tick ─────────────────────────────────────────────────────────

	$effect(() => {
		if (refreshTick > 0 && untrack(() => hasLoaded)) {
			untrack(query);
		}
	});

	// ─── Auto-refresh every 30 s ─────────────────────────────────────────────

	$effect(() => {
		if (!hasLoaded) return;
		const id = setInterval(() => query(), 30_000);
		return () => clearInterval(id);
	});

	// ─── Auto-save selectedField view state ───────────────────────────────────

	$effect(() => {
		if (!hasLoaded || !dashboardStore.editMode) return;
		const field = selectedField;
		untrack(() => {
			dashboardStore.saveWidgetViewState(widgetId, { ...config, selectedField: field });
		});
	});

	// ─── Flush view state before create() snapshots ───────────────────────────

	$effect(() => {
		if (!dashboardStore.isSaving || !dashboardStore.isCreatingNew || !hasLoaded) return;
		untrack(() => {
			dashboardStore.saveWidgetViewState(widgetId, { ...config, selectedField });
		});
	});

	// ─── Data ─────────────────────────────────────────────────────────────────

	async function loadAndQuery() {
		const mir = mirStore.mir;
		if (!mir || !config.measurement) return;
		loadError = null;
		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			const groups = await mir.client().listTelemetry().request(target);
			const group = groups.find((g) => g.descriptors.some((d) => d.name === config.measurement));
			deviceInfos = group?.ids ?? (config.target.ids ?? []).map((id) => ({ id, name: id }));
		} catch {
			deviceInfos = (config.target.ids ?? []).map((id) => ({ id, name: id }));
		}
		onDevicesReady?.(
			deviceInfos.map((dev, devIdx) => ({
				...dev,
				color: CHART_COLORS[devIdx % CHART_COLORS.length]
			}))
		);
		hasLoaded = true;
		await query();
	}

	async function query() {
		const mir = mirStore.mir;
		if (!mir || !config.measurement || deviceInfos.length === 0) return;
		const myGen = ++generation;

		const end = new Date();
		const start = new Date(end.getTime() - (config.timeMinutes ?? 60) * 60 * 1000);

		try {
			const results = await Promise.all(
				deviceInfos.map((dev) =>
					mir
						.client()
						.queryTelemetry()
						.request(
							new DeviceTarget({ ids: [dev.id] }),
							config.measurement,
							config.fields,
							start,
							end,
							undefined // no aggregation — raw rows, last one is the latest
						)
						.then((data) => ({ deviceId: dev.id, data }))
				)
			);
			if (myGen !== generation) return;

			const newMap = new Map<
				string,
				{ values: Record<string, unknown>; timestamp: Date | null }
			>();
			for (const { deviceId, data } of results) {
				if (data.rows.length === 0) {
					newMap.set(deviceId, { values: {}, timestamp: null });
					continue;
				}
				const lastRow = data.rows[data.rows.length - 1];
				const ts = lastRow.values['_time'];
				newMap.set(deviceId, {
					values: lastRow.values as Record<string, unknown>,
					timestamp: ts instanceof Date ? ts : null
				});
			}
			lastRows = newMap;
		} catch {
			// silently ignore — stale data remains displayed
		}
	}

	// ─── Helpers ──────────────────────────────────────────────────────────────

	function getFreshness(timestamp: Date | null): { label: string; cls: string } {
		if (!timestamp) return { label: 'stale', cls: 'text-destructive' };
		const ageSec = (Date.now() - timestamp.getTime()) / 1000;
		if (ageSec < 5) return { label: 'live', cls: 'text-emerald-500' };
		if (ageSec < 60) return { label: `${Math.round(ageSec)}s`, cls: 'text-emerald-500' };
		const ageMin = ageSec / 60;
		if (ageMin < 5) return { label: `${Math.round(ageMin)}m`, cls: 'text-amber-500' };
		if (ageMin < 60) return { label: `${Math.round(ageMin)}m`, cls: 'text-muted-foreground' };
		return { label: 'stale', cls: 'text-destructive' };
	}

	function formatValue(val: unknown): string {
		if (val === null || val === undefined) return '—';
		if (typeof val === 'number') return Number.isInteger(val) ? String(val) : val.toFixed(2);
		if (typeof val === 'boolean') return val ? 'true' : 'false';
		return String(val);
	}
</script>

<div class="flex h-full flex-col">
	<!-- Field switcher (hidden when only one field) -->
	{#if config.fields.length > 1}
		<div class="flex shrink-0 flex-wrap gap-1 border-b px-3 py-2">
			{#each config.fields as field (field)}
				<button
					onclick={() => (selectedField = field)}
					class="rounded-md border px-2 py-0.5 font-mono text-xs transition-colors
						{selectedField === field
						? 'border-primary bg-primary text-primary-foreground'
						: 'border-border hover:bg-accent'}"
				>
					{field}
				</button>
			{/each}
		</div>
	{/if}

	{#if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if deviceInfos.length === 0 && hasLoaded}
		<p class="px-4 py-2 text-xs text-muted-foreground">No devices found.</p>
	{:else}
		<!-- Device tile grid -->
		<div class="flex-1 overflow-auto p-3">
			<div
				class="grid gap-2"
				style="grid-template-columns: repeat({Math.min(deviceInfos.length, 2)}, 1fr)"
			>
				{#each deviceInfos as dev (dev.id)}
					{@const row = lastRows.get(dev.id)}
					{@const val = row?.values[selectedField] ?? null}
					{@const freshness = getFreshness(row?.timestamp ?? null)}
					{@const noData = !row || row.timestamp === null}
					<div
						class="relative rounded-lg border border-border bg-card px-3 py-3 text-center transition-opacity
							{noData ? 'opacity-40' : ''}"
					>
						<!-- Freshness badge -->
						<span
							class="absolute top-1.5 right-1.5 rounded px-1 py-px font-mono text-[9px] leading-tight {freshness.cls}"
						>
							{freshness.label}
						</span>

						<!-- Device name -->
						<p class="mb-1.5 truncate font-mono text-[11px] text-muted-foreground">{dev.name}</p>

						<!-- Value -->
						<p class="text-2xl font-bold leading-none tracking-tight text-foreground">
							{formatValue(val)}
						</p>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
```

- [ ] **Step 2: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/widget-tlm-last.svelte
git commit -m "feat(dashboard): add widget-tlm-last component — device tile grid with freshness badges"
```

---

## Task 5: Wire dispatch in dashboard-grid

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte`

- [ ] **Step 1: Import `WidgetTlmLast` and update the type import**

At the top of the script (after existing imports around line 8–16), add:

```typescript
import WidgetTlmLast from './widget-tlm-last.svelte';
```

Also update the type import to include `TelemetryWidgetConfig` if not already present (it is already imported on line 16).

- [ ] **Step 2: Add dispatch branch for last-value**

In the template, replace the existing telemetry branch (around line 104–110):

```svelte
{#if widget.type === 'telemetry'}
    <WidgetTelemetry
        widgetId={widget.id}
        config={widget.config as TelemetryWidgetConfig}
        {refreshTick}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
```

With:

```svelte
{#if widget.type === 'telemetry' && (widget.config as TelemetryWidgetConfig).subtype === 'last-value'}
    <WidgetTlmLast
        widgetId={widget.id}
        config={widget.config as TelemetryWidgetConfig}
        {refreshTick}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
{:else if widget.type === 'telemetry'}
    <WidgetTelemetry
        widgetId={widget.id}
        config={widget.config as TelemetryWidgetConfig}
        {refreshTick}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
```

- [ ] **Step 3: Type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: no new errors (3 pre-existing errors only).

- [ ] **Step 4: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte
git commit -m "feat(dashboard): dispatch widget-tlm-last for telemetry last-value subtype"
```

---

## Task 6: End-to-end verification

**Prerequisites:** Mir infra running (`mir infra up`) + Mir server running (`mir serve`) + a device swarm active (`mir swarm --ids=power,weather,sensor-3`).

- [ ] **Step 1: Start the dev server**

```bash
cd internal/ui/web && npm run dev
```

Open http://localhost:5173 (or the dev server URL shown).

- [ ] **Step 2: Verify wizard flow — Line Chart (regression)**

1. Open a dashboard → Add Widget
2. Click **Telemetry**
3. Confirm a new step 2 appears with "Line Chart" and "Last Value" cards
4. Click **Line Chart** → confirm it proceeds to target (device) selection
5. Complete wizard → confirm a normal line chart widget appears
6. Confirm chart renders data

- [ ] **Step 3: Verify wizard flow — Last Value (new)**

1. Add Widget → Telemetry → **Last Value**
2. Select target devices
3. In config step: select a measurement → confirm field picker appears
4. Select a field → confirm **Add Widget** becomes enabled
5. Add the widget → confirm tile grid appears on dashboard

- [ ] **Step 4: Verify tile grid behaviour**

1. If measurement has >1 field: confirm pill switcher appears at top of widget
2. Click a different field pill → tiles update immediately, no reload
3. Confirm freshness badge shows `live` or small seconds count for active devices
4. Stop one device → wait ~60s → confirm that device tile goes to `stale`

- [ ] **Step 5: Verify auto-refresh**

Wait 30 seconds with the widget visible. Confirm the freshness badge timestamps update (values re-fetched without user action).

- [ ] **Step 6: Verify edit mode**

1. Edit the Last Value widget → wizard opens at target step with pre-selected measurement and field
2. Confirm `selectedTlmSubtype` is `'last-value'` (visible if you go Back to `tlm-sub` step)
3. Save → widget re-renders correctly

- [ ] **Step 7: Final type-check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -v "^src/lib/shared/components/shadcn/nav-section\|multi/telemetry"
```

Expected: zero new errors.
