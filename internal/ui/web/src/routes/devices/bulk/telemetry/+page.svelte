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
	import TimePicker from '$lib/domains/devices/components/telemetry/time-picker.svelte';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { contextStore } from '$lib/domains/contexts/stores/contexts.svelte';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import type { Descriptor } from '$lib/domains/devices/types/types';
	import type { DateRange } from 'bits-ui';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';
	import { SvelteDate } from 'svelte/reactivity';
	import {
		CHART_COLORS,
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
	let enabledDeviceIds = $state<Set<string>>(new Set());
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
	let calendarValue = $state<DateRange | undefined>(undefined);
	let fullscreen = $state(false);
	let hasZoomed = $state(false);

	// Time input state
	let startTime = $state('00:00');
	let endTime = $state('23:59');

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
		enabledDeviceIds = new Set();
		mergedData = null;
		mergedFields = [];
		chartConfig = {};
		queryError = null;
		queryStart = null;
		queryEnd = null;
	}

	// ─── Time helpers ─────────────────────────────────────────────────────────

	// Keep time inputs in sync with timeFilter and UTC preference
	$effect(() => {
		if (timeFilter.mode === 'absolute') {
			const getH = (d: Date) => (editorPrefs.utc ? d.getUTCHours() : d.getHours());
			const getM = (d: Date) => (editorPrefs.utc ? d.getUTCMinutes() : d.getMinutes());
			startTime = `${String(getH(timeFilter.start)).padStart(2, '0')}:${String(getM(timeFilter.start)).padStart(2, '0')}`;
			endTime = `${String(getH(timeFilter.end)).padStart(2, '0')}:${String(getM(timeFilter.end)).padStart(2, '0')}`;
		}
	});

	function handleCalendarChange(value: DateRange | undefined): TimeFilter | undefined {
		if (value?.start && value?.end) {
			const [startH, startM] = startTime.split(':').map(Number);
			const [endH, endM] = endTime.split(':').map(Number);
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			const start = value.start.toDate(tz);
			const end = value.end.toDate(tz);
			if (editorPrefs.utc) {
				start.setUTCHours(startH, startM, 0, 0);
				end.setUTCHours(endH, endM, 59, 999);
			} else {
				start.setHours(startH, startM, 0, 0);
				end.setHours(endH, endM, 59, 999);
			}
			return { mode: 'absolute', start, end };
		}
		return undefined;
	}

	function handleTimeInputChange() {
		if (timeFilter.mode === 'absolute') {
			const [startH, startM] = startTime.split(':').map(Number);
			const [endH, endM] = endTime.split(':').map(Number);
			const start = new SvelteDate(timeFilter.start);
			const end = new SvelteDate(timeFilter.end);
			if (editorPrefs.utc) {
				start.setUTCHours(startH, startM, 0, 0);
				end.setUTCHours(endH, endM, 59, 999);
			} else {
				start.setHours(startH, startM, 0, 0);
				end.setHours(endH, endM, 59, 999);
			}
			timeFilter = { mode: 'absolute', start, end };
		}
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
			const fields = selectedFields.length ? selectedFields : selectedMeasurement.fields;

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
			let colorIdx = 0;

			results.forEach(({ deviceName, data }) => {
				data.headers
					.filter((h) => !h.startsWith('_'))
					.forEach((field) => {
						const key = `${deviceName}_${field}`;
						newMergedFields.push(key);
						newChartConfig[key] = {
							label: `${deviceName}: ${field}`,
							color: CHART_COLORS[colorIdx % CHART_COLORS.length]
						};
						colorIdx++;
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
		selectedFields = full.fields.slice(0, MAX_AUTO_FIELDS);
		enabledDeviceIds = new Set(tlmGroups[groupIdx].ids.map((id) => id.id));
		mergedData = null;
		queryError = null;
		queryGroup();
	}

	function toggleDevice(deviceId: string, shift: boolean) {
		if (shift) {
			const next = new Set(enabledDeviceIds);
			if (next.has(deviceId)) {
				if (next.size > 1) next.delete(deviceId);
			} else {
				next.add(deviceId);
			}
			enabledDeviceIds = next;
		} else {
			enabledDeviceIds = new Set([deviceId]);
		}
	}

	let visibleFields = $derived(
		mergedFields.filter((key) => {
			const group = tlmGroups[selection?.groupIdx ?? -1];
			if (!group) return true;
			return group.ids.some((id) => enabledDeviceIds.has(id.id) && key.startsWith(id.name + '_'));
		})
	);

	function handleBrushSelect(newStart: Date, newEnd: Date) {
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		hasZoomed = true;
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
		queryGroup();
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
			bind:calendarValue
			bind:fullscreen
			bind:hasZoomed
			queryData={mergedData}
			presets={PRESETS}
			onQuery={() => { if (selection) queryGroup(); }}
			onCalendarChange={handleCalendarChange}
		>
			{#snippet calendarTop()}
				<p class="mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase">
					Custom range{editorPrefs.utc ? ' (UTC)' : ''}
				</p>
				<div class="mb-3 grid grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<span class="text-xs font-medium text-muted-foreground">Start time</span>
						<TimePicker bind:value={startTime} onchange={handleTimeInputChange} />
					</div>
					<div class="space-y-1.5">
						<span class="text-xs font-medium text-muted-foreground">End time</span>
						<TimePicker bind:value={endTime} onchange={handleTimeInputChange} />
					</div>
				</div>
			{/snippet}
		</TlmToolbar>

		{#if !selection}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<ActivityIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a measurement to view chart</p>
			</div>
		{:else}
			<!-- Device pills -->
			<div class="flex shrink-0 flex-wrap items-center gap-2 border-b px-4 py-1.5">
				{#each tlmGroups[selection.groupIdx]?.ids ?? [] as dev (dev.id)}
					<button
						onclick={(e) => toggleDevice(dev.id, e.shiftKey)}
						class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
							{enabledDeviceIds.has(dev.id)
								? 'border-transparent bg-foreground text-background'
								: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
					>
						{dev.name}
					</button>
				{/each}
			</div>

			<TlmFieldToggles fields={selection.measurement.fields} {selectedFields} ontoggle={toggleField} />

			<!-- Chart + data panel scrollable area -->
			<div class="min-h-0 flex-1 overflow-auto">
				<!-- Chart -->
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
							{chartConfig}
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
		{/if}
	</div>
</div>
