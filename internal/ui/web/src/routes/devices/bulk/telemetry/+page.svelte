<script lang="ts">
	import { getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { TelemetryGroup, TelemetryDescriptor, QueryData, QueryRow } from '@mir/sdk';
	import DescriptorPanel from '$lib/domains/devices/components/commands/descriptor-panel.svelte';
	import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
	import TlmToolbar from '$lib/domains/devices/components/telemetry/tlm-toolbar.svelte';
	import TlmFieldToggles from '$lib/domains/devices/components/telemetry/tlm-field-toggles.svelte';
	import TlmDataPanel from '$lib/domains/devices/components/telemetry/tlm-data-panel.svelte';
	import TlmSlotChart from '$lib/domains/devices/components/telemetry/tlm-slot-chart.svelte';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import Columns2Icon from '@lucide/svelte/icons/columns-2';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import Grid2x2Icon from '@lucide/svelte/icons/grid-2x2';
	import SquareIcon from '@lucide/svelte/icons/square';
	import LockIcon from '@lucide/svelte/icons/lock';
	import LockOpenIcon from '@lucide/svelte/icons/lock-open';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { contextStore } from '$lib/domains/contexts/stores/contexts.svelte';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import type { Descriptor } from '$lib/domains/devices/types/types';
	import {
		CHART_COLORS,
		getDeviceFieldColor,
		MAX_AUTO_FIELDS,
		type TimeFilter,
		getTimeRange,
		getAggregationWindow
	} from '$lib/domains/devices/utils/tlm-time';

	// ─── Constants ────────────────────────────────────────────────────────────

	const PRESETS = [
		{ label: '1m', minutes: 1 },
		{ label: '5m', minutes: 5 },
		{ label: '10m', minutes: 10 },
		{ label: '15m', minutes: 15 },
		{ label: '30m', minutes: 30 },
		{ label: '1h', minutes: 60 },
		{ label: '3h', minutes: 180 },
		{ label: '6h', minutes: 360 },
		{ label: '12h', minutes: 720 },
		{ label: '24h', minutes: 1440 },
		{ label: '2d', minutes: 2880 },
		{ label: '7d', minutes: 10080 },
		{ label: '30d', minutes: 43200 },
		{ label: '90d', minutes: 129600 }
	] as const;

	type Selection = {
		groupIdx: number;
		measurement: TelemetryDescriptor;
	} | null;

	let tlmGroups = $state<TelemetryGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let selection = $state<Selection>(null);
	let selectedFields = $state<string[]>([]);
	let enabledDeviceIds = $state<string[]>([]);
	let mergedData = $state<QueryData | null>(null);
	let mergedFields = $state<string[]>([]);
	let chartConfig = $state<ChartConfig>({});
	let isQuerying = $state(false);
	let queryError = $state<string | null>(null);
	let queryStart = $state<Date | null>(null);
	let queryEnd = $state<Date | null>(null);

	// Cancellation guard for parallel queries
	let generation = 0;

	// Toolbar state (bound to TlmToolbar)
	let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 5 });
	let fullscreen = $state(false);
	let splitCount = $state<1 | 2 | 3 | 4>(1);
	let syncFields = $state(false);
	let syncSelectedFields = $state<string[]>([]);

	// ─── Refresh context ──────────────────────────────────────────────────────

	const multiCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('multi');
	multiCtx.setTabRefresh(() => loadMeasurements(true));
	onDestroy(() => multiCtx.setTabRefresh(null));

	// ─── Derived state ────────────────────────────────────────────────────────

	let groups = $derived(
		tlmGroups.map((g) => ({
			label: g.ids.map((id) => `${id.name}/${id.namespace}`).join(', '),
			items: g.descriptors.map((d) => ({
				name: d.name,
				labels: d.labels,
				template: '',
				error: d.error
			})),
			errors: g.error ? [g.error] : []
		}))
	);

	let selectedKey = $derived(
		selection ? `${selection.groupIdx}:${selection.measurement.name}` : null
	);

	let grafanaUrl = $derived.by(() => {
		const host = contextStore.activeContext?.grafana;
		const query = selection?.measurement.exploreQuery;
		if (!host || !query) return null;
		const panes = JSON.stringify({
			xyz: {
				datasource: 'mir-influxdb',
				queries: [{ refId: 'A', datasource: { type: 'influxdb', uid: 'mir-influxdb' }, query }],
				range: { from: 'now-1h', to: 'now' }
			}
		});
		return `http://${host}/explore?${new URLSearchParams({ schemaVersion: '1', orgId: '1', panes })}`;
	});

	// ─── Load measurements ────────────────────────────────────────────────────

	$effect(() => {
		if (!mirStore.mir || selectionStore.activeCount === 0) return;
		loadMeasurements();
	});

	async function loadMeasurements(preserveSelection = false) {
		if (!mirStore.mir) return;
		generation++;
		isLoading = true;
		error = null;

		// Capture selection identity before async fetch
		const prevGroupKey =
			preserveSelection && selection
				? tlmGroups[selection.groupIdx]?.ids
						.map((id) => id.id)
						.sort()
						.join(',')
				: null;
		const prevMeasurementName = preserveSelection ? (selection?.measurement.name ?? null) : null;

		try {
			const allIds = selectionStore.activeDevices.map((d) => d.spec.deviceId);
			const target = new DeviceTarget({ ids: allIds });
			const fetched = await mirStore.mir.client().listTelemetry().request(target);
			tlmGroups = fetched;

			if (prevGroupKey && prevMeasurementName) {
				const newGroupIdx = fetched.findIndex(
					(g) =>
						g.ids
							.map((id) => id.id)
							.sort()
							.join(',') === prevGroupKey
				);
				if (newGroupIdx === -1) {
					clearSelection();
				} else {
					const newMeasurement =
						fetched[newGroupIdx].descriptors.find((d) => d.name === prevMeasurementName) ?? null;
					if (!newMeasurement) {
						clearSelection();
					} else {
						selection = { groupIdx: newGroupIdx, measurement: newMeasurement };
						queryGroup();
					}
				}
			} else {
				clearSelection();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load telemetry';
		} finally {
			isLoading = false;
		}
	}

	function clearSelection() {
		selection = null;
		selectedFields = [];
		enabledDeviceIds = [];
		mergedData = null;
		mergedFields = [];
		chartConfig = {};
		queryError = null;
		queryStart = null;
		queryEnd = null;
		splitCount = 1;
		syncFields = false;
		syncSelectedFields = [];
	}

	// ─── Query and merge ──────────────────────────────────────────────────────

	async function queryGroup() {
		if (!mirStore.mir || !selection) return;

		const myGen = generation;
		const group = tlmGroups[selection.groupIdx];
		const selectedMeasurement = selection.measurement;

		isQuerying = true;
		queryError = null;

		try {
			const { start, end } = getTimeRange(timeFilter);
			queryStart = start;
			queryEnd = end;
			// Multi-device merge requires aligned timestamps — always aggregate
			const aggWindow = getAggregationWindow(start, end) ?? '10s';
			const fields = selectedMeasurement.fields;

			const results = await Promise.all(
				group.ids.map((devId) =>
					mirStore.mir!
						.client()
						.queryTelemetry()
						.request(
							new DeviceTarget({ ids: [devId.id] }),
							selectedMeasurement.name,
							fields,
							start,
							end,
							aggWindow
						)
						.then((data) => ({ deviceId: devId.id, deviceName: devId.name, data }))
				)
			);

			if (myGen !== generation) return;

			const newMergedFields: string[] = [];
			const newChartConfig: ChartConfig = {};

			selectedMeasurement.fields.forEach((field, fieldIdx) => {
				results.forEach(({ deviceName, data }) => {
					if (!data.headers.includes(field)) return;
					const key = `${deviceName}_${field}`;
					newMergedFields.push(key);
					newChartConfig[key] = {
						label: `${field} - ${deviceName} `,
						color: CHART_COLORS[fieldIdx % CHART_COLORS.length]
					};
				});
			});

			const timeMap = new Map<number, QueryRow>();
			results.forEach(({ deviceName, data }) => {
				data.rows.forEach((row) => {
					const t = row.values['_time'] as Date;
					const key = t instanceof Date ? t.getTime() : 0;
					if (!timeMap.has(key)) {
						timeMap.set(key, { values: { _time: t } });
					}
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

			isQuerying = false;
			mergedData = { headers: ['_time', ...newMergedFields], rows: sortedRows };
			mergedFields = newMergedFields;
			chartConfig = newChartConfig;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Query failed';

			activityStore.add({
				kind: 'error',
				category: 'Telemetry',
				title: selectedMeasurement.name,
				error: message
			});

			if (myGen !== generation) return;

			isQuerying = false;
			queryError = message;
		}
	}

	function selectMeasurement(groupIdx: number, desc: Descriptor) {
		const full = tlmGroups[groupIdx].descriptors.find((d) => d.name === desc.name);
		if (!full) return;
		selection = { groupIdx, measurement: full };
		selectedFields = full.fields.slice(0, 1);
		enabledDeviceIds = tlmGroups[groupIdx].ids.slice(0, 5).map((id) => id.id);
		syncSelectedFields = full.fields.slice(0, 1);
		mergedData = null;
		queryError = null;
		queryGroup();
	}

	function toggleDevice(deviceId: string, shift: boolean) {
		if (shift) {
			if (enabledDeviceIds.includes(deviceId)) {
				if (enabledDeviceIds.length > 1) enabledDeviceIds = enabledDeviceIds.filter((id) => id !== deviceId);
			} else {
				enabledDeviceIds = [...enabledDeviceIds, deviceId];
			}
		} else {
			enabledDeviceIds = [deviceId];
		}
	}

	let visibleFields = $derived(
		mergedFields.filter((key) => {
			const group = tlmGroups[selection?.groupIdx ?? -1];
			if (!group) return true;
			return group.ids.some(
				(id) =>
					enabledDeviceIds.includes(id.id) &&
					key.startsWith(id.name + '_') &&
					selectedFields.includes(key.slice(id.name.length + 1))
			);
		})
	);

	let splitSlots = $derived.by(() => {
		const group = tlmGroups[selection?.groupIdx ?? -1];
		if (!group) return [];
		const fields = selection?.measurement.fields ?? [];
		return Array.from({ length: splitCount }, (_, i) => ({
			device: group.ids[i] ?? group.ids[0] ?? null,
			fields: [fields[i % Math.max(1, fields.length)]].filter(Boolean) as string[]
		}));
	});

	let singleViewChartConfig = $derived.by(() => {
		const group = tlmGroups[selection?.groupIdx ?? -1];
		if (!group) return chartConfig;
		const enabledDevices = enabledDeviceIds.map((id) => group.ids.find((g) => g.id === id)).filter((d): d is typeof group.ids[0] => d !== undefined);
		if (enabledDevices.length <= 1) return chartConfig;
		const result: ChartConfig = { ...chartConfig };
		enabledDevices.forEach((dev, devIdx) => {
			selectedFields.forEach((field) => {
				const fieldIdx = selection?.measurement.fields.indexOf(field) ?? 0;
				const key = `${dev.name}_${field}`;
				if (key in result) result[key] = { ...result[key], color: getDeviceFieldColor(fieldIdx, devIdx) } as typeof result[string];
			});
		});
		return result;
	});

	

	function handleBrushSelect(newStart: Date, newEnd: Date) {
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		queryGroup();
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

<div class="-m-4 flex h-[calc(100svh-13rem)] overflow-hidden rounded-none border-y">
	<DescriptorPanel
		title="Telemetry"
		items={[]}
		{groups}
		{isLoading}
		{error}
		groupErrors={[]}
		{selectedKey}
		emptyText="No telemetry found."
		onSelect={() => {}}
		onSelectGrouped={selectMeasurement}
	/>

	<div
		class="{fullscreen
			? 'fixed inset-0 z-50 bg-background'
			: 'min-w-0 flex-1'} flex flex-col overflow-hidden"
	>
		<TlmToolbar
			measurementName={selection?.measurement.name ?? null}
			measurementError={selection?.measurement.error ?? null}
			{grafanaUrl}
			bind:timeFilter
			bind:fullscreen
			queryData={mergedData}
			presets={PRESETS}
			onQuery={() => { if (selection) queryGroup(); }}
		>
		{#snippet toolbarEnd()}
			{#if splitCount > 1}
				<button
					onclick={() => (syncFields = !syncFields)}
					title={syncFields ? 'Unsync fields' : 'Sync fields across charts'}
					class="flex items-center rounded-md border p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
						{syncFields ? 'border-ring bg-accent text-accent-foreground' : 'border-border bg-background'}"
				>
					{#if syncFields}
						<LockIcon class="size-3.5" />
					{:else}
						<LockOpenIcon class="size-3.5" />
					{/if}
				</button>
			{/if}
			<button
				onclick={() => {
					const next = splitCount === 4 ? 1 : (splitCount + 1) as 1 | 2 | 3 | 4;
					if (next === 1) syncFields = false;
					else if (splitCount === 1) syncFields = true;
					splitCount = next;
				}}
				title={splitCount === 1 ? 'Split in 2' : splitCount === 2 ? 'Split in 3' : splitCount === 3 ? 'Split in 4' : 'Single view'}
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
		</TlmToolbar>

		{#if !selection}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<ActivityIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a measurement to view chart</p>
			</div>
		{:else}
			{#if splitCount === 1}
			<TlmFieldToggles fields={selection.measurement.fields} {selectedFields} ontoggle={toggleField} />

			<!-- Device pills -->
			<div class="flex shrink-0 flex-wrap items-center gap-2 border-b px-4 py-1.5">
				{#each tlmGroups[selection.groupIdx]?.ids ?? [] as dev (dev.id)}
					<button
						onclick={(e) => toggleDevice(dev.id, e.shiftKey || e.ctrlKey)}
						class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
							{enabledDeviceIds.includes(dev.id)
								? 'border-transparent bg-foreground text-background'
								: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
					>
						{dev.name}
					</button>
				{/each}
			</div>
		{:else if syncFields}
			<TlmFieldToggles
				fields={selection.measurement.fields}
				selectedFields={syncSelectedFields}
				ontoggle={toggleSyncField}
			/>
		{/if}

			<!-- Chart + data panel scrollable area -->
			{#if splitCount === 1}
				<div class="min-h-0 flex-1 overflow-auto">
					<!-- Single chart -->
					<div class="px-4 py-4">
						{#if queryError}
							<p class="text-sm text-destructive">{queryError}</p>
						{:else if isQuerying && !mergedData}
							<div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
								<Spinner class="mr-2 size-4" />
								Loading data…
							</div>
						{:else if mergedData}
							<TlmChart
								data={mergedData}
								selectedFields={visibleFields}
								chartConfig={singleViewChartConfig}
								useUtc={editorPrefs.utc}
								start={queryStart}
								end={queryEnd}
								chartClass={fullscreen ? 'h-[calc(100vh-20vh)]' : 'h-72'}
								onBrushSelect={handleBrushSelect}
							/>
						{/if}
					</div>
					<TlmDataPanel data={mergedData} exploreQuery={selection.measurement.exploreQuery} />
				</div>
			{:else}
				<!-- Split grid: fills available height -->
				<div class="min-h-0 flex-1 overflow-hidden px-4 py-4">
					{#if queryError}
						<p class="text-sm text-destructive">{queryError}</p>
					{:else if isQuerying && !mergedData}
						<div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
							<Spinner class="mr-2 size-4" />
							Loading data…
						</div>
					{:else if mergedData}
						<div
							class="h-full grid grid-cols-2 gap-3"
							style="grid-template-rows: repeat({splitCount > 2 ? 2 : 1}, 1fr)"
						>
							{#each splitSlots as slot, i (i)}
								<div class="min-h-0 flex flex-col overflow-hidden {splitCount === 3 && i === 0 ? 'col-span-2' : ''}">
									<TlmSlotChart
										devices={tlmGroups[selection.groupIdx].ids}
										measurementFields={selection.measurement.fields}
										{mergedData}
										{chartConfig}
										{queryStart}
										{queryEnd}
										useUtc={editorPrefs.utc}
										initialDeviceId={slot.device?.id}
										initialFields={slot.fields}
										chartClass="flex-1 min-h-0"
										forcedFields={syncFields ? syncSelectedFields : undefined}
										onBrushSelect={handleBrushSelect}
									/>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</div>
