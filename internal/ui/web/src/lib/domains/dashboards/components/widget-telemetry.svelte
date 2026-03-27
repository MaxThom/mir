<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import { SvelteMap } from 'svelte/reactivity';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import type { QueryData, QueryRow } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
	import TlmToolbar from '$lib/domains/devices/components/telemetry/tlm-toolbar.svelte';
	import TlmFieldToggles from '$lib/domains/devices/components/telemetry/tlm-field-toggles.svelte';
	import TlmSlotChart from '$lib/domains/devices/components/telemetry/tlm-slot-chart.svelte';
	import {
		CHART_COLORS,
		getDeviceFieldColor,
		type TimeFilter,
		getTimeRange,
		getAggregationWindow
	} from '$lib/domains/devices/utils/tlm-time';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import Columns2Icon from '@lucide/svelte/icons/columns-2';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import Grid2x2Icon from '@lucide/svelte/icons/grid-2x2';
	import SquareIcon from '@lucide/svelte/icons/square';
	import LockIcon from '@lucide/svelte/icons/lock';
	import LockOpenIcon from '@lucide/svelte/icons/lock-open';
	import type { TelemetryWidgetConfig } from '../api/dashboard-api';

	let {
		config,
		widgetId,
		refreshTick = 0
	}: { config: TelemetryWidgetConfig; widgetId: string; refreshTick?: number } = $props();

	// ─── State ────────────────────────────────────────────────────────────────

	let deviceInfos = $state<{ id: string; name: string }[]>([]);
	let loadError = $state<string | null>(null);
	let queryError = $state<string | null>(null);
	let mergedData = $state<QueryData | null>(null);
	let mergedFields = $state<string[]>([]);
	let chartConfig = $state<ChartConfig>({});
	let queryStart = $state<Date | null>(null);
	let queryEnd = $state<Date | null>(null);

	let timeFilter = $state<TimeFilter>(editorPrefs.timeFilter);
	let splitCount = $state<1 | 2 | 3 | 4>(untrack(() => (config.splitCount ?? 1) as 1 | 2 | 3 | 4));
	let syncFields = $state(untrack(() => config.syncFields ?? false));
	let hasLoaded = $state(false);

	let selectedFields = $state<string[]>([]);
	let enabledDeviceIds = $state<string[]>([]);
	let syncSelectedFields = $state<string[]>([]);

	let fullscreen = $state(false);
	let widgetEl: HTMLDivElement | undefined;
	let widgetWidth = $state(9999);

	const compact = $derived(widgetWidth < 420);

	let generation = 0;
	let isInRefresh = false;

	// ─── Global time change ────────────────────────────────────────────────────

	$effect(() => {
		const globalFilter = editorPrefs.timeFilter;
		if (untrack(() => !hasLoaded)) return;
		untrack(() => {
			timeFilter = globalFilter;
			query();
		});
	});

	// ─── Refresh tick ─────────────────────────────────────────────────────────

	$effect(() => {
		if (refreshTick > 0) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			if (untrack(() => deviceInfos).length === 0) loadAndQuery();
			else query();
		}
	});

	// ─── Auto-save view settings ──────────────────────────────────────────────

	$effect(() => {
		if (!hasLoaded) return;

		// Capture user-controlled state (these are the tracked dependencies)
		const currentSelectedFields = selectedFields;
		const currentSplitCount = splitCount;
		const currentSyncFields = syncFields;
		const currentEnabledDeviceIds = enabledDeviceIds;

		// Use saveWidgetViewState (no immediate _syncDashboard) to avoid
		// cascading prop changes that would re-trigger a data query.
		untrack(() => {
			dashboardStore.saveWidgetViewState(widgetId, {
				...config,
				selectedFields: currentSelectedFields,
				splitCount: currentSplitCount,
				syncFields: currentSyncFields,
				enabledDeviceIds: currentEnabledDeviceIds
			});
		});
	});

	// Track widget width for compact mode
	$effect(() => {
		if (!widgetEl) return;
		const ro = new ResizeObserver(([entry]) => {
			widgetWidth = entry.contentRect.width;
		});
		ro.observe(widgetEl);
		return () => ro.disconnect();
	});

	// ─── Startup ──────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			// untrack so config.* reads inside loadAndQuery don't become
			// dependencies — saveWidgetConfig creates a new config object each
			// time and would otherwise re-trigger this effect in a tight loop.
			untrack(loadAndQuery);
		} else {
			mergedData = null;
		}
	});

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
			// Fall back to raw IDs as names
			deviceInfos = (config.target.ids ?? []).map((id) => ({ id, name: id }));
		}

		enabledDeviceIds =
			config.enabledDeviceIds?.filter((id) => deviceInfos.some((d) => d.id === id)) ??
			deviceInfos.map((d) => d.id);

		const validSelectedFields =
			config.selectedFields?.filter((f) => config.fields.includes(f)) ?? [];
		selectedFields =
			validSelectedFields.length > 0 ? validSelectedFields : config.fields.slice(0, 1);
		syncSelectedFields = selectedFields;
		hasLoaded = true;
		query();
	}

	// ─── Query and merge ──────────────────────────────────────────────────────

	async function query() {
		const mir = mirStore.mir;
		if (!mir || !config.measurement || deviceInfos.length === 0) return;

		const myGen = ++generation;
		queryError = null;

		try {
			const { start, end } = getTimeRange(timeFilter);
			queryStart = start;
			queryEnd = end;
			const aggWindow = getAggregationWindow(start, end) ?? '10s';

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
							aggWindow
						)
						.then((data) => ({ deviceId: dev.id, deviceName: dev.name, data }))
				)
			);

			if (myGen !== generation) return;

			const newMergedFields: string[] = [];
			const newChartConfig: ChartConfig = {};

			config.fields.forEach((field, fieldIdx) => {
				results.forEach(({ deviceName, data }) => {
					if (!data.headers.includes(field)) return;
					const key = `${deviceName}_${field}`;
					newMergedFields.push(key);
					newChartConfig[key] = {
						label: `${field} - ${deviceName}`,
						color: CHART_COLORS[fieldIdx % CHART_COLORS.length]
					};
				});
			});

			const timeMap = new SvelteMap<number, QueryRow>();
			results.forEach(({ deviceName, data }) => {
				data.rows.forEach((row) => {
					const t = row.values['_time'] as Date;
					const key = t instanceof Date ? t.getTime() : 0;
					if (!timeMap.has(key)) timeMap.set(key, { values: { _time: t } });
					const mergedRow = timeMap.get(key)!;
					data.headers
						.filter((h) => !h.startsWith('_'))
						.forEach((field) => {
							mergedRow.values[`${deviceName}_${field}`] = row.values[field];
						});
				});
			});

			const sortedRows = Array.from(timeMap.values()).sort((a, b) => {
				const ta = a.values['_time'] as Date;
				const tb = b.values['_time'] as Date;
				return ta.getTime() - tb.getTime();
			});

			if (myGen !== generation) return;

			mergedData = { headers: ['_time', ...newMergedFields], rows: sortedRows };
			mergedFields = newMergedFields;
			chartConfig = newChartConfig;
		} catch (err) {
			if (myGen !== generation) return;
			queryError = err instanceof Error ? err.message : 'Query failed';
		} finally {
			if (myGen === generation && isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
		}
	}

	// ─── Derived ──────────────────────────────────────────────────────────────

	let visibleFields = $derived(
		mergedFields.filter((key) => {
			return deviceInfos.some(
				(dev) =>
					enabledDeviceIds.includes(dev.id) &&
					key.startsWith(dev.name + '_') &&
					selectedFields.includes(key.slice(dev.name.length + 1))
			);
		})
	);

	let singleViewChartConfig = $derived.by(() => {
		const enabledDevices = deviceInfos.filter((d) => enabledDeviceIds.includes(d.id));
		if (enabledDevices.length <= 1) return chartConfig;
		const result: ChartConfig = { ...chartConfig };
		enabledDevices.forEach((dev, devIdx) => {
			selectedFields.forEach((field) => {
				const fieldIdx = config.fields.indexOf(field);
				const key = `${dev.name}_${field}`;
				if (key in result)
					result[key] = {
						...result[key],
						color: getDeviceFieldColor(fieldIdx, devIdx)
					} as (typeof result)[string];
			});
		});
		return result;
	});

	let splitSlots = $derived(
		Array.from({ length: splitCount }, (_, i) => ({
			device: deviceInfos[i] ?? deviceInfos[0] ?? null,
			fields: [config.fields[i % Math.max(1, config.fields.length)]].filter(Boolean) as string[]
		}))
	);

	// ─── Interactions ─────────────────────────────────────────────────────────

	function handleBrushSelect(newStart: Date, newEnd: Date) {
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		editorPrefs.setTimeFilter({ mode: 'absolute', start: newStart, end: newEnd });
	}

	function toggleDevice(id: string, shift: boolean) {
		if (shift) {
			if (enabledDeviceIds.includes(id)) {
				if (enabledDeviceIds.length > 1)
					enabledDeviceIds = enabledDeviceIds.filter((x) => x !== id);
			} else {
				enabledDeviceIds = [...enabledDeviceIds, id];
			}
		} else {
			enabledDeviceIds = [id];
		}
	}

	function toggleField(field: string, shift: boolean) {
		if (shift) {
			if (selectedFields.includes(field)) {
				if (selectedFields.length > 1) selectedFields = selectedFields.filter((f) => f !== field);
			} else {
				selectedFields = [...selectedFields, field];
			}
		} else {
			selectedFields = [field];
		}
	}

	function toggleSyncField(field: string, shift: boolean) {
		if (shift) {
			if (syncSelectedFields.includes(field)) {
				if (syncSelectedFields.length > 1)
					syncSelectedFields = syncSelectedFields.filter((f) => f !== field);
			} else {
				syncSelectedFields = [...syncSelectedFields, field];
			}
		} else {
			syncSelectedFields = [field];
		}
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && fullscreen) fullscreen = false;
	}}
/>

<div
	bind:this={widgetEl}
	class="{fullscreen
		? 'fixed inset-0 z-50 bg-background'
		: 'flex h-full flex-col overflow-hidden'} flex flex-col overflow-hidden"
>
	<TlmToolbar
		measurementName={config.measurement}
		bind:timeFilter
		bind:fullscreen
		{compact}
		showZoom={false}
		queryData={mergedData}
		onQuery={query}
	>
		{#snippet toolbarEnd()}
			<button
				onclick={() => (syncFields = !syncFields)}
				disabled={splitCount === 1}
				title={syncFields ? 'Unsync fields' : 'Sync fields across charts'}
				class="flex items-center rounded-md border p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-40
					{syncFields ? 'border-ring bg-accent text-accent-foreground' : 'border-border bg-background'}"
			>
				{#if syncFields}
					<LockIcon class="size-3.5" />
				{:else}
					<LockOpenIcon class="size-3.5" />
				{/if}
			</button>
			<button
				onclick={() => {
					const next = splitCount === 4 ? 1 : ((splitCount + 1) as 1 | 2 | 3 | 4);
					if (next === 1) syncFields = false;
					else if (splitCount === 1) syncFields = true;
					splitCount = next;
				}}
				title={splitCount === 1
					? 'Split in 2'
					: splitCount === 2
						? 'Split in 3'
						: splitCount === 3
							? 'Split in 4'
							: 'Single view'}
				class="flex items-center rounded-md border p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
					{splitCount > 1 ? 'border-ring bg-accent text-accent-foreground' : 'border-border bg-background'}"
			>
				{#if splitCount === 1}
					<SquareIcon class="size-3.5" />
				{:else if splitCount === 2}
					<Columns2Icon class="size-3.5" />
				{:else if splitCount === 3}
					<LayoutDashboardIcon class="size-3.5" />
				{:else}
					<Grid2x2Icon class="size-3.5" />
				{/if}
			</button>
		{/snippet}
		{#snippet compactDropdownExtra()}
			{#if config.fields.length > 0 && (splitCount === 1 || syncFields)}
				{@const activeFields = splitCount === 1 ? selectedFields : syncSelectedFields}
				{@const activeToggle = splitCount === 1 ? toggleField : toggleSyncField}
				<div class="my-1 h-px bg-border"></div>
				<div class="flex flex-col gap-0.5">
					{#each config.fields as field, i (field)}
						{@const selected = activeFields.includes(field)}
						<button
							onclick={(e) => activeToggle(field, e.shiftKey || e.ctrlKey)}
							class="flex items-center gap-2 rounded-sm px-2 py-1 text-xs transition-colors
								{selected ? 'bg-accent font-medium text-accent-foreground' : 'text-foreground hover:bg-accent/60'}"
						>
							<span
								class="inline-block size-2 shrink-0 rounded-full"
								style="background: {CHART_COLORS[i % CHART_COLORS.length]}"
							></span>
							{field}
						</button>
					{/each}
				</div>
			{/if}
		{/snippet}
	</TlmToolbar>

	{#if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else}
		<!-- Field toggles (move into toolbar dropdown when compact) -->
		{#if config.fields.length > 0 && !compact}
			{#if splitCount === 1}
				<TlmFieldToggles fields={config.fields} {selectedFields} ontoggle={toggleField} />
			{:else if syncFields}
				<TlmFieldToggles
					fields={config.fields}
					selectedFields={syncSelectedFields}
					ontoggle={toggleSyncField}
				/>
			{/if}
		{/if}

		<!-- Chart area -->
		<div class="flex min-h-0 flex-1 flex-col overflow-hidden">
			{#if splitCount === 1}
				{#if queryError}
					<p class="px-4 py-2 text-sm text-destructive">{queryError}</p>
				{:else if mergedData}
					<div class="flex min-h-0 flex-1 flex-col pt-1 pb-1">
						<div class="relative min-h-0 flex-1">
							<TlmChart
								data={mergedData}
								selectedFields={visibleFields}
								chartConfig={singleViewChartConfig}
								chartClass="h-full"
								start={queryStart}
								end={queryEnd}
								useUtc={editorPrefs.utc}
								onBrushSelect={handleBrushSelect}
							/>
						</div>
						{#if deviceInfos.length > 0 && selectedFields.length > 0}
							<div class="mt-0.5 flex shrink-0 flex-wrap gap-x-3 gap-y-0.5 px-4">
								{#each deviceInfos as dev (dev.id)}
									{@const enabled = enabledDeviceIds.includes(dev.id)}
									{@const color =
										singleViewChartConfig[`${dev.name}_${selectedFields[0]}`]?.color ??
										'var(--chart-1)'}
									<button
										onclick={(e) => toggleDevice(dev.id, e.shiftKey || e.ctrlKey)}
										class="flex items-center gap-1 font-mono text-[11px] transition-colors {enabled
											? 'text-foreground'
											: 'text-muted-foreground/40'}"
									>
										<span
											class="inline-block size-2 shrink-0 rounded-full transition-opacity {enabled
												? ''
												: 'opacity-30'}"
											style="background: {color}"
										></span>
										{dev.name}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				{:else}
					<p class="px-4 py-2 text-xs text-muted-foreground">No data</p>
				{/if}
			{:else if queryError}
				<p class="px-4 py-2 text-sm text-destructive">{queryError}</p>
			{:else if mergedData}
				<div class="relative min-h-0 flex-1 overflow-hidden py-1">
					<div
						class="grid h-full grid-cols-2 gap-3"
						style="grid-template-rows: repeat({splitCount > 2 ? 2 : 1}, 1fr)"
					>
						{#each splitSlots as slot, i (i)}
							<div
								class="flex min-h-0 flex-col overflow-hidden px-2 {splitCount === 3 && i === 0
									? 'col-span-2'
									: ''}"
							>
								<TlmSlotChart
									devices={deviceInfos}
									measurementFields={config.fields}
									{mergedData}
									{chartConfig}
									{queryStart}
									{queryEnd}
									initialDeviceId={slot.device?.id}
									initialFields={slot.fields}
									chartClass="flex-1 min-h-0"
									forcedFields={syncFields ? syncSelectedFields : undefined}
									onBrushSelect={handleBrushSelect}
								/>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>
