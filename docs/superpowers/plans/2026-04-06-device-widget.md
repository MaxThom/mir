# Device Widget Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only `device` dashboard widget that shows either Meta/Spec/Status or Desired/Reported properties for a set of targeted devices, with a per-device pill selector in the content header.

**Architecture:** Single widget type `'device'` with a `view: 'info' | 'properties'` config field chosen in wizard step 3. The widget loads device data via `listDevices()` on mount and on each `refreshTick`. When multiple devices are targeted, clickable pill tabs at the top of the content area switch which device is displayed (`selectedDeviceId` saved via `saveWidgetViewState`).

**Tech Stack:** SvelteKit + Svelte 5 runes, shadcn-svelte, Lucide icons, `@mir/sdk` (`DeviceTarget`, `Device`), TypeScript.

---

## Files

| Action | Path |
|--------|------|
| Modify | `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts` |
| Create | `internal/ui/web/src/lib/domains/dashboards/components/widget-device.svelte` |
| Modify | `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte` |
| Modify | `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte` |

---

## Task 1: Add `DeviceWidgetConfig` to `dashboard-api.ts`

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts`

- [ ] **Step 1: Add the type and extend the union**

In `dashboard-api.ts`, apply these three changes:

**Line 1** — extend `WidgetType`:
```typescript
// before
export type WidgetType = 'telemetry' | 'command' | 'config' | 'events';
// after
export type WidgetType = 'telemetry' | 'command' | 'config' | 'events' | 'device';
```

**After `EventsWidgetConfig`** — add new interface:
```typescript
export interface DeviceWidgetConfig {
	target: DeviceTargetConfig;
	view: 'info' | 'properties';
	selectedDeviceId?: string; // view state — active pill tab
}
```

**`WidgetConfig` union** — add the new type:
```typescript
// before
export type WidgetConfig =
	| TelemetryWidgetConfig
	| CommandWidgetConfig
	| ConfigWidgetConfig
	| EventsWidgetConfig;
// after
export type WidgetConfig =
	| TelemetryWidgetConfig
	| CommandWidgetConfig
	| ConfigWidgetConfig
	| EventsWidgetConfig
	| DeviceWidgetConfig;
```

- [ ] **Step 2: Verify TypeScript is happy**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -i error | head -20
```

Expected: only the 3 pre-existing errors in `nav-section.svelte` and `multi/telemetry`, nothing new.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts
git commit -m "feat(dashboard): add DeviceWidgetConfig type"
```

---

## Task 2: Create `widget-device.svelte`

**Files:**
- Create: `internal/ui/web/src/lib/domains/dashboards/components/widget-device.svelte`

- [ ] **Step 1: Create the file**

Create `internal/ui/web/src/lib/domains/dashboards/components/widget-device.svelte` with the following content:

```svelte
<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { Device } from '@mir/sdk';
	import type { DeviceWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { JsonValue } from '$lib/shared/components/ui/json-value';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { cn } from '$lib/utils';
	import CircleCheckBigIcon from '@lucide/svelte/icons/circle-check-big';

	let {
		config,
		widgetId,
		onDevicesReady,
		refreshTick = 0
	}: {
		config: DeviceWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
		refreshTick?: number;
	} = $props();

	let devices = $state<Device[]>([]);
	let isLoading = $state(false);
	let hasLoaded = $state(false);
	let loadError = $state<string | null>(null);

	const selectedDevice = $derived(
		devices.find((d) => d.spec?.deviceId === config.selectedDeviceId) ?? devices[0] ?? null
	);

	const deviceInfos = $derived(
		devices.map((d, i) => ({
			id: d.spec?.deviceId ?? '',
			name: d.meta?.name ?? '',
			color: CHART_COLORS[i % CHART_COLORS.length]
		}))
	);

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadDevices);
		} else {
			devices = [];
			hasLoaded = false;
		}
	});

	$effect(() => {
		if (refreshTick > 0 && mirStore.mir && hasLoaded) {
			untrack(loadDevices);
		}
	});

	async function loadDevices() {
		const mir = mirStore.mir;
		if (!mir) return;
		if (!hasLoaded) isLoading = true;
		loadError = null;
		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			devices = await mir.client().listDevices().request(target, false);
			onDevicesReady?.(
				devices.map((d, i) => ({
					id: d.spec?.deviceId ?? '',
					name: d.meta?.name ?? '',
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);
			// Default to first device if savedId is gone from the list
			if (
				devices.length > 0 &&
				!devices.some((d) => d.spec?.deviceId === config.selectedDeviceId)
			) {
				selectDevice(devices[0].spec?.deviceId ?? '');
			}
			hasLoaded = true;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load devices';
		} finally {
			isLoading = false;
		}
	}

	function selectDevice(deviceId: string) {
		if (!dashboardStore.activeDashboard) return;
		dashboardStore.saveWidgetViewState(widgetId, { ...config, selectedDeviceId: deviceId });
	}

	function isMatchingDesired(device: Device, key: string, reportedVal: unknown): boolean {
		const desired = (device.properties?.desired ?? {}) as Record<string, unknown>;
		if (!(key in desired)) return false;
		return JSON.stringify(desired[key]) === JSON.stringify(reportedVal);
	}
</script>

<div class="flex h-full flex-col overflow-hidden">
	{#if isLoading}
		<div class="flex flex-1 items-center justify-center">
			<span class="text-xs text-muted-foreground">Loading…</span>
		</div>
	{:else if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if devices.length === 0 && hasLoaded}
		<p class="p-4 text-xs text-muted-foreground">No devices found for this target.</p>
	{:else if selectedDevice}
		<!-- Device pill tabs (only when multiple devices) -->
		{#if devices.length > 1}
			<div class="flex shrink-0 items-center gap-1.5 overflow-x-auto border-b px-3 py-1.5">
				{#each deviceInfos as info (info.id)}
					<button
						onclick={() => selectDevice(info.id)}
						class={cn(
							'shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
							info.id === (selectedDevice.spec?.deviceId ?? '')
								? 'bg-primary text-primary-foreground'
								: 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
						)}
					>
						{info.name}
					</button>
				{/each}
			</div>
		{/if}

		<div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
			{#if config.view === 'info'}
				<!-- ── Info view ────────────────────────────────────────────────── -->
				<div class="space-y-2">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Meta</p>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Name</span>
						<span class="flex-1 text-sm font-medium">{selectedDevice.meta?.name ?? '—'}</span>
					</div>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Namespace</span>
						<span class="flex-1 font-mono text-sm">{selectedDevice.meta?.namespace ?? '—'}</span>
					</div>

					<div class="flex items-start gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Labels</span>
						<div class="flex flex-1 flex-wrap gap-1">
							{#each Object.entries(selectedDevice.meta?.labels ?? {}) as [k, v] (k)}
								<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
							{:else}
								<span class="text-sm text-muted-foreground">—</span>
							{/each}
						</div>
					</div>

					<div class="flex items-start gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Annotations</span>
						<div class="flex flex-1 flex-wrap gap-1">
							{#each Object.entries(selectedDevice.meta?.annotations ?? {}) as [k, v] (k)}
								<span class="text-xs text-muted-foreground">{k}: {v}</span>
							{:else}
								<span class="text-sm text-muted-foreground">—</span>
							{/each}
						</div>
					</div>
				</div>

				<Separator class="my-2" />

				<div class="space-y-2">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Spec</p>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Device ID</span>
						<span class="flex-1 font-mono text-xs text-muted-foreground">
							{selectedDevice.spec?.deviceId ?? '—'}
						</span>
					</div>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Disabled</span>
						{#if selectedDevice.spec?.disabled}
							<Badge variant="destructive" class="text-xs">Yes</Badge>
						{:else}
							<span class="text-sm text-muted-foreground">No</span>
						{/if}
					</div>
				</div>

				<Separator class="my-2" />

				<div class="space-y-2">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Status</p>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Connectivity</span>
						<div class="flex items-center gap-2">
							<span
								class={cn(
									'h-2 w-2 shrink-0 rounded-full',
									selectedDevice.status?.online
										? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
										: 'bg-muted-foreground/30'
								)}
							></span>
							<span
								class={cn(
									'text-sm font-medium',
									selectedDevice.status?.online
										? 'text-emerald-600 dark:text-emerald-400'
										: 'text-muted-foreground'
								)}
							>
								{selectedDevice.status?.online ? 'Online' : 'Offline'}
							</span>
						</div>
					</div>

					<div class="flex items-center gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Last Heartbeat</span>
						<div class="flex-1">
							{#if selectedDevice.status?.lastHearthbeat}
								<TimeTooltip
									timestamp={selectedDevice.status.lastHearthbeat}
									utc={editorPrefs.utc}
									class="text-sm hover:text-foreground"
								/>
							{:else}
								<span class="text-sm text-muted-foreground">—</span>
							{/if}
						</div>
					</div>

					<div class="flex items-start gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Schema</span>
						<div class="flex-1">
							{#if selectedDevice.status?.schema?.packageNames?.length}
								<div class="flex flex-wrap gap-1">
									{#each selectedDevice.status.schema.packageNames as pkg (pkg)}
										<Badge variant="outline" class="font-mono text-xs font-normal">{pkg}</Badge>
									{/each}
								</div>
								{#if selectedDevice.status.schema.lastSchemaFetch}
									<TimeTooltip
										timestamp={selectedDevice.status.schema.lastSchemaFetch}
										utc={editorPrefs.utc}
										prefix="fetched "
										class="mt-0.5 text-xs text-muted-foreground hover:text-foreground"
									/>
								{/if}
							{:else}
								<span class="text-sm text-muted-foreground">Not loaded</span>
							{/if}
						</div>
					</div>
				</div>
			{:else}
				<!-- ── Properties view ──────────────────────────────────────────── -->
				{@const desiredProps = Object.entries(selectedDevice.properties?.desired ?? {}).sort(([a], [b]) => a.localeCompare(b))}
				{@const reportedProps = Object.entries(selectedDevice.properties?.reported ?? {}).sort(([a], [b]) => a.localeCompare(b))}

				{#if desiredProps.length === 0 && reportedProps.length === 0}
					<p class="text-sm text-muted-foreground">No properties configured.</p>
				{:else}
					<div class="space-y-3">
						<div>
							<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
								Desired
							</p>
							{#if desiredProps.length === 0}
								<p class="text-xs text-muted-foreground">—</p>
							{:else}
								<div class="space-y-1.5">
									{#each desiredProps as [k, v] (k)}
										<div class="flex flex-col">
											<div class="flex items-center gap-1.5">
												<span class="font-mono text-xs text-muted-foreground">{k}</span>
												{#if selectedDevice.status?.properties?.desired?.[k]}
													<TimeTooltip
														timestamp={selectedDevice.status.properties.desired[k]}
														utc={editorPrefs.utc}
														class="text-[10px] text-muted-foreground/60"
													/>
												{/if}
											</div>
											<JsonValue value={v} />
										</div>
									{/each}
								</div>
							{/if}
						</div>

						<Separator />

						<div>
							<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
								Reported
							</p>
							{#if reportedProps.length === 0}
								<p class="text-xs text-muted-foreground">—</p>
							{:else}
								<div class="space-y-1.5">
									{#each reportedProps as [k, v] (k)}
										<div class="flex flex-col">
											<div class="flex items-center gap-1.5">
												<span class="font-mono text-xs text-muted-foreground">{k}</span>
												{#if isMatchingDesired(selectedDevice, k, v)}
													<CircleCheckBigIcon class="size-3 text-emerald-500" />
												{/if}
												{#if selectedDevice.status?.properties?.reported?.[k]}
													<TimeTooltip
														timestamp={selectedDevice.status.properties.reported[k]}
														utc={editorPrefs.utc}
														class="text-[10px] text-muted-foreground/60"
													/>
												{/if}
											</div>
											<JsonValue value={v} />
										</div>
									{/each}
								</div>
							{/if}
						</div>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>
```

- [ ] **Step 2: Check types**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -i error | head -20
```

Expected: only the 3 pre-existing errors, nothing new.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/widget-device.svelte
git commit -m "feat(dashboard): add widget-device component (info + properties views)"
```

---

## Task 3: Register widget in `dashboard-grid.svelte`

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte`

- [ ] **Step 1: Add import**

At the top of the `<script>` block, after the `WidgetEvents` import, add:

```typescript
import WidgetDevice from './widget-device.svelte';
import type { TelemetryWidgetConfig, CommandWidgetConfig, ConfigWidgetConfig, EventsWidgetConfig, DeviceWidgetConfig } from '../api/dashboard-api';
```

(Replace the existing `import type { TelemetryWidgetConfig, CommandWidgetConfig, ConfigWidgetConfig, EventsWidgetConfig }` line with the one above that adds `DeviceWidgetConfig`.)

- [ ] **Step 2: Add rendering case**

In the widget rendering block, after the `{:else if widget.type === 'events'}` block and before `{/if}`, add:

```svelte
{:else if widget.type === 'device'}
    <WidgetDevice
        widgetId={widget.id}
        config={widget.config as DeviceWidgetConfig}
        {refreshTick}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
```

The full updated block looks like:
```svelte
{#if widget.type === 'telemetry'}
    <WidgetTelemetry ... />
{:else if widget.type === 'command'}
    <WidgetCommand ... />
{:else if widget.type === 'config'}
    <WidgetConfig ... />
{:else if widget.type === 'events'}
    <WidgetEvents config={widget.config as EventsWidgetConfig} {refreshTick} />
{:else if widget.type === 'device'}
    <WidgetDevice
        widgetId={widget.id}
        config={widget.config as DeviceWidgetConfig}
        {refreshTick}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
{/if}
```

- [ ] **Step 3: Check types**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -i error | head -20
```

Expected: only the 3 pre-existing errors.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte
git commit -m "feat(dashboard): register device widget in grid"
```

---

## Task 4: Add device type to `add-widget-dialog.svelte`

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte`

- [ ] **Step 1: Add imports**

At the top of the `<script>` block, add these imports alongside the existing ones:

```typescript
import CpuIcon from '@lucide/svelte/icons/cpu';
import type { DeviceWidgetConfig } from '../api/dashboard-api';
```

Also add `DeviceWidgetConfig` to the existing named import from `'../api/dashboard-api'`:
```typescript
import type {
    Widget,
    WidgetType,
    DeviceTargetConfig,
    TelemetryWidgetConfig,
    EventsWidgetConfig,
    CommandWidgetConfig,
    ConfigWidgetConfig,
    DeviceWidgetConfig   // ← add this
} from '../api/dashboard-api';
```

- [ ] **Step 2: Add `selectedDeviceView` state**

After the `let eventLimit = $state(50);` line, add:

```typescript
// Device config
let selectedDeviceView = $state<'info' | 'properties'>('info');
```

- [ ] **Step 3: Update `reset()`**

In the `reset()` function, after `eventLimit = 50;`, add:

```typescript
selectedDeviceView = 'info';
```

- [ ] **Step 4: Update `typeLabel()`**

In the `typeLabel()` function, add the `device` case before the closing brace:

```typescript
case 'device':
    return 'Device';
```

- [ ] **Step 5: Update edit-mode population**

In the `$effect` that pre-populates state when `editWidget` is set, add after the `else if (editWidget.type === 'events')` block:

```typescript
} else if (editWidget.type === 'device') {
    const c = editWidget.config as DeviceWidgetConfig;
    selectedDeviceView = c.view ?? 'info';
}
```

- [ ] **Step 6: Update `buildConfig()`**

In `buildConfig()`, update the return type and add the `device` case. First, update the return type signature:

```typescript
function buildConfig(): TelemetryWidgetConfig | CommandWidgetConfig | ConfigWidgetConfig | EventsWidgetConfig | DeviceWidgetConfig {
```

Then add the case before the closing brace of the switch:

```typescript
case 'device':
    return { target, view: selectedDeviceView } satisfies DeviceWidgetConfig;
```

- [ ] **Step 7: Update step 1 type grid**

Find the `<div class="grid w-full max-w-4xl grid-cols-4 gap-4">` line and change `grid-cols-4` to `grid-cols-5`.

In the `{#each [...] as item}` array, add the device entry as the 5th element:

```typescript
{ type: 'device' as WidgetType, icon: CpuIcon, label: 'Device', desc: 'View device meta, status and properties' }
```

The full updated array passed to `{#each}`:
```typescript
[
    { type: 'telemetry' as WidgetType, icon: ActivityIcon, label: 'Telemetry', desc: 'Visualize time-series data from device sensors' },
    { type: 'command' as WidgetType, icon: TerminalIcon, label: 'Command', desc: 'Send commands and view responses from devices' },
    { type: 'config' as WidgetType, icon: SlidersHorizontalIcon, label: 'Configuration', desc: 'Manage and push configuration to devices' },
    { type: 'events' as WidgetType, icon: CalendarClockIcon, label: 'Events', desc: 'Monitor events and audit logs from the fleet' },
    { type: 'device' as WidgetType, icon: CpuIcon, label: 'Device', desc: 'View device meta, status and properties' }
]
```

- [ ] **Step 8: Add step 3 view picker for `device` type**

In the step 3 block (`{#if step === 'config'}`), find the `{:else if selectedType === 'events'}` section. After its closing `{/if}` (but still before the outer `{:else}` fallthrough), add:

```svelte
{:else if selectedType === 'device'}
    <div class="space-y-1">
        <p class="text-sm font-medium">View</p>
        <div class="grid grid-cols-2 gap-3">
            <button
                onclick={() => (selectedDeviceView = 'info')}
                class="flex flex-col items-center gap-2 rounded-xl border p-6 text-center transition-colors hover:border-primary hover:bg-accent {selectedDeviceView === 'info' ? 'border-primary bg-accent' : 'border-border'}"
            >
                <InfoIcon class="h-8 w-8 text-muted-foreground" />
                <div>
                    <p class="font-semibold">Info</p>
                    <p class="mt-0.5 text-xs text-muted-foreground">Meta, Spec &amp; Status</p>
                </div>
            </button>
            <button
                onclick={() => (selectedDeviceView = 'properties')}
                class="flex flex-col items-center gap-2 rounded-xl border p-6 text-center transition-colors hover:border-primary hover:bg-accent {selectedDeviceView === 'properties' ? 'border-primary bg-accent' : 'border-border'}"
            >
                <SlidersHorizontalIcon class="h-8 w-8 text-muted-foreground" />
                <div>
                    <p class="font-semibold">Properties</p>
                    <p class="mt-0.5 text-xs text-muted-foreground">Desired &amp; Reported</p>
                </div>
            </button>
        </div>
    </div>
```

Note: `InfoIcon` is already imported. `SlidersHorizontalIcon` is already imported.

- [ ] **Step 9: Check types**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -i error | head -20
```

Expected: only the 3 pre-existing errors.

- [ ] **Step 10: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte
git commit -m "feat(dashboard): add device widget to add-widget dialog"
```

---

## Task 5: End-to-end verification

- [ ] **Step 1: Start infrastructure and dev server**

```bash
# Terminal 1
just infra

# Terminal 2
mir serve

# Terminal 3
cd internal/ui/web && npm run dev
```

- [ ] **Step 2: Add a Device Info widget**

1. Open dashboard in browser
2. Enter edit mode → Add Widget
3. Step 1: click **Device** → step 2: pick 2+ devices → step 3: click **Info** → Add Widget
4. Verify: widget renders with Meta/Spec/Status sections for first device
5. Click a different device pill → verify content switches to that device
6. Verify `selectedDeviceId` persists after page refresh

- [ ] **Step 3: Add a Device Properties widget**

1. Add Widget → Device → same devices → **Properties** → Add Widget
2. Verify: Desired and Reported sections render with `JsonValue` for each key
3. Verify: ✓ icon appears on Reported keys that match Desired
4. Verify: timestamps appear next to keys that have them

- [ ] **Step 4: Verify auto-refresh**

1. Enable dashboard auto-refresh (e.g. 30s)
2. Verify widget reloads data on tick (no errors in browser console)

- [ ] **Step 5: Verify single-device target**

1. Add Device widget targeting exactly 1 device
2. Verify: no pill tabs shown (only one device, nothing to switch)

- [ ] **Step 6: Final type check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -i error | head -20
```

Expected: only the 3 pre-existing errors.

- [ ] **Step 7: Final commit**

```bash
git add -p  # stage any remaining tweaks
git commit -m "feat(dashboard): device widget — info and properties views"
```
