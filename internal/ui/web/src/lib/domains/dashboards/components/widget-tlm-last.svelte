<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { CHART_COLORS, getTimeRange } from '$lib/domains/devices/utils/tlm-time';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import type { GlobalTimeFilter } from '$lib/shared/stores/editor-prefs.svelte';
	import TlmFieldToggles from '$lib/domains/devices/components/telemetry/tlm-field-toggles.svelte';
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
	let isInRefresh = false;
	let generation = 0;
	let timeFilter = $state<GlobalTimeFilter>(editorPrefs.timeFilter);

	// ─── Global time change ────────────────────────────────────────────────────

	$effect(() => {
		const globalFilter = editorPrefs.timeFilter;
		if (untrack(() => !hasLoaded)) return;
		untrack(() => {
			timeFilter = globalFilter;
			query();
		});
	});

	// ─── Startup ─────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadAndQuery);
		} else {
			lastRows = new Map();
		}
	});

	// Stable key for structural config — triggers reload when measurement/fields/target change.
	const configKey = $derived(
		`${config.measurement}|${(config.fields ?? []).join('\0')}|${JSON.stringify(config.target ?? {})}`
	);

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		configKey;
		if (mirStore.mir && untrack(() => hasLoaded)) {
			untrack(loadAndQuery);
		}
	});

	// ─── Refresh tick ─────────────────────────────────────────────────────────

	$effect(() => {
		if (refreshTick > 0) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			if (untrack(() => deviceInfos).length === 0) untrack(loadAndQuery);
			else untrack(query);
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
		// Reset selectedField if it no longer exists in the (possibly updated) fields list
		if (!config.fields.includes(selectedField)) {
			selectedField = config.selectedField ?? config.fields[0] ?? '';
		}
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

		const { start, end } = getTimeRange(timeFilter);

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
		} finally {
			if (myGen === generation && isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
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

	// Sort: devices with a real timestamp first, stale (no timestamp) last.
	const sortedDeviceInfos = $derived(
		[...deviceInfos].sort((a, b) => {
			const aStale = !lastRows.get(a.id)?.timestamp;
			const bStale = !lastRows.get(b.id)?.timestamp;
			if (aStale === bStale) return 0;
			return aStale ? 1 : -1;
		})
	);

	function formatValue(val: unknown): string {
		if (val === null || val === undefined) return '—';
		if (typeof val === 'number') return Number.isInteger(val) ? String(val) : val.toFixed(2);
		if (typeof val === 'boolean') return val ? 'true' : 'false';
		return String(val);
	}
</script>

<div class="flex h-full flex-col">
	<!-- Separator (with breathing room above) -->
	<div class="mt-2 shrink-0 border-b"></div>

	<!-- Field switcher (hidden when only one field) -->
	{#if config.fields.length > 1}
		<TlmFieldToggles
			fields={config.fields}
			selectedFields={[selectedField]}
			ontoggle={(field) => (selectedField = field)}
		/>
	{/if}

	{#if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if deviceInfos.length === 0 && hasLoaded}
		<p class="px-4 py-2 text-xs text-muted-foreground">No devices found.</p>
	{:else}
		<!-- Device tile grid -->
		<div class="min-h-0 flex-1 overflow-auto p-3">
			<div
				class="grid h-full gap-2"
				style="grid-template-columns: repeat({Math.min(deviceInfos.length, 2)}, 1fr); grid-template-rows: repeat({Math.ceil(deviceInfos.length / 2)}, 1fr)"
			>
				{#each sortedDeviceInfos as dev (dev.id)}
					{@const row = lastRows.get(dev.id)}
					{@const val = row?.values[selectedField] ?? null}
					{@const freshness = getFreshness(row?.timestamp ?? null)}
					{@const noData = !row || row.timestamp === null}
					<div
						class="relative flex flex-col items-center justify-center rounded-lg border border-border bg-card px-3 py-3 text-center transition-opacity
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
