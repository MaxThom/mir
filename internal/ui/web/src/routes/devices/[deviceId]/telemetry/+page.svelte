<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { telemetryStore } from '$lib/domains/devices/stores/telemetry.svelte';
	import type { TelemetryDescriptor } from '@mir/sdk';
	import type { Descriptor } from '$lib/domains/devices/types/types';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import DescriptorPanel from '$lib/domains/devices/components/commands/descriptor-panel.svelte';
	import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
	import TlmToolbar from '$lib/domains/devices/components/telemetry/tlm-toolbar.svelte';
	import TlmFieldToggles from '$lib/domains/devices/components/telemetry/tlm-field-toggles.svelte';
	import TlmDataPanel from '$lib/domains/devices/components/telemetry/tlm-data-panel.svelte';
	import TimePicker from '$lib/domains/devices/components/telemetry/time-picker.svelte';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { contextStore } from '$lib/domains/contexts/stores/contexts.svelte';
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

	// ─── State ────────────────────────────────────────────────────────────────

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedMeasurement = $state<TelemetryDescriptor | null>(null);
	let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 5 });
	let selectedFields = $state<string[]>([]);
	let calendarValue = $state<DateRange | undefined>(undefined);
	let startTime = $state('00:00');
	let endTime = $state('23:59');
	let queryStart = $state<Date | null>(null);
	let queryEnd = $state<Date | null>(null);
	let hasZoomed = $state(false);
	let fullscreen = $state(false);

	// ─── Grafana Explore URL ──────────────────────────────────────────────────

	function grafanaExploreUrl(fluxQuery: string): string | null {
		const host = contextStore.activeContext?.grafana;
		if (!host || !fluxQuery) return null;
		const panes = JSON.stringify({
			xyz: {
				datasource: 'mir-influxdb',
				queries: [
					{
						refId: 'A',
						datasource: { type: 'influxdb', uid: 'mir-influxdb' },
						query: fluxQuery
					}
				],
				range: { from: 'now-1h', to: 'now' }
			}
		});
		const params = new URLSearchParams({ schemaVersion: '1', orgId: '1', panes });
		return `http://${host}/explore?${params}`;
	}

	let grafanaUrl = $derived(
		selectedMeasurement ? grafanaExploreUrl(selectedMeasurement.exploreQuery) : null
	);

	// ─── Adapters ─────────────────────────────────────────────────────────────

	let allDescriptors = $derived(
		telemetryStore.measurements.flatMap((g) =>
			g.descriptors.map(
				(d): Descriptor => ({
					name: d.name,
					labels: d.labels,
					template: '',
					error: d.error
				})
			)
		)
	);

	let groupErrors = $derived(telemetryStore.measurements.map((g) => g.error).filter(Boolean));
	let allTelemetryDescriptors = $derived(telemetryStore.measurements.flatMap((g) => g.descriptors));

	// ─── Chart config ─────────────────────────────────────────────────────────

	let chartConfig = $derived.by((): ChartConfig => {
		const config: ChartConfig = {};
		(selectedMeasurement?.fields ?? []).forEach((field, i) => {
			config[field] = { label: field, color: CHART_COLORS[i % CHART_COLORS.length] };
		});
		return config;
	});

	// ─── Context & lifecycle ──────────────────────────────────────────────────

	const deviceCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('device');
	deviceCtx.setTabRefresh(() => {
		if (mirStore.mir && deviceId) {
			telemetryStore.loadMeasurements(mirStore.mir, deviceId);
			runQuery();
		}
	});
	onDestroy(() => {
		deviceCtx.setTabRefresh(null);
	});

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedMeasurement = null;
			selectedFields = [];
			telemetryStore.reset();
			telemetryStore.loadMeasurements(mirStore.mir, deviceId);
		}
	});

	// Keep time inputs in sync with timeFilter and UTC preference
	$effect(() => {
		if (timeFilter.mode === 'absolute') {
			const getH = (d: Date) => (editorPrefs.utc ? d.getUTCHours() : d.getHours());
			const getM = (d: Date) => (editorPrefs.utc ? d.getUTCMinutes() : d.getMinutes());
			startTime = `${String(getH(timeFilter.start)).padStart(2, '0')}:${String(getM(timeFilter.start)).padStart(2, '0')}`;
			endTime = `${String(getH(timeFilter.end)).padStart(2, '0')}:${String(getM(timeFilter.end)).padStart(2, '0')}`;
		}
	});

	// ─── Query logic ──────────────────────────────────────────────────────────

	function runQuery() {
		if (!mirStore.mir || !selectedMeasurement) return;
		const { start, end } = getTimeRange(timeFilter);
		queryStart = start;
		queryEnd = end;
		const aggWindow = getAggregationWindow(start, end);
		telemetryStore.queryMeasurement(
			mirStore.mir,
			deviceId,
			selectedMeasurement.name,
			selectedFields,
			start,
			end,
			aggWindow
		);
	}

	function selectMeasurement(desc: Descriptor) {
		const full = allTelemetryDescriptors.find((d) => d.name === desc.name);
		if (!full) return;
		selectedMeasurement = full;
		selectedFields = full.fields.slice(0, MAX_AUTO_FIELDS);
		telemetryStore.queryData = null;
		telemetryStore.queryError = null;
		runQuery();
	}

	function handleBrushSelect(newStart: Date, newEnd: Date) {
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		hasZoomed = true;
		runQuery();
	}

	function toggleField(field: string, shift: boolean) {
		if (shift) {
			if (selectedFields.includes(field)) {
				if (selectedFields.length > 1) {
					selectedFields = selectedFields.filter((f) => f !== field);
				}
			} else {
				selectedFields = [...selectedFields, field];
			}
		} else {
			selectedFields = [field];
		}
		runQuery();
	}

	// Calendar change handler: applies startTime/endTime to the selected date range
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
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && fullscreen) fullscreen = false;
	}}
/>

<div class="-m-4 flex min-h-125 overflow-hidden rounded-none border-y">
	<!-- Left: measurement list -->
	<DescriptorPanel
		title="Telemetry"
		items={allDescriptors}
		isLoading={telemetryStore.isLoading}
		error={telemetryStore.error}
		{groupErrors}
		selectedName={selectedMeasurement?.name ?? null}
		emptyText="No telemetry found."
		onSelect={selectMeasurement}
	/>

	<!-- Right: chart area -->
	<div
		class="{fullscreen
			? 'fixed inset-0 z-50 bg-background'
			: 'min-w-0 flex-1'} flex flex-col overflow-hidden"
	>
		<TlmToolbar
			measurementName={selectedMeasurement?.name ?? null}
			measurementError={selectedMeasurement?.error ?? null}
			{grafanaUrl}
			bind:timeFilter
			bind:calendarValue
			bind:hasZoomed
			bind:fullscreen
			queryData={telemetryStore.queryData}
			presets={PRESETS}
			onQuery={runQuery}
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

		{#if !selectedMeasurement}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<ActivityIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a measurement to view chart</p>
			</div>
		{:else}
			<TlmFieldToggles fields={selectedMeasurement.fields} {selectedFields} ontoggle={toggleField} />

			<!-- Chart + Table scrollable area -->
			<div class="min-h-0 flex-1 overflow-auto">
				<!-- Chart -->
				<div class="px-4 py-4">
					{#if telemetryStore.queryError}
						<p class="text-sm text-destructive">{telemetryStore.queryError}</p>
					{:else if telemetryStore.isQuerying && !telemetryStore.queryData}
						<div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
							Loading data…
						</div>
					{:else if telemetryStore.queryData}
						<TlmChart
							data={telemetryStore.queryData}
							{selectedFields}
							{chartConfig}
							useUtc={editorPrefs.utc}
							start={queryStart}
							end={queryEnd}
							chartClass={fullscreen ? 'h-[calc(100vh-20vh)]' : 'h-72'}
							onBrushSelect={handleBrushSelect}
						/>
					{/if}
				</div>

				<TlmDataPanel data={telemetryStore.queryData} exploreQuery={selectedMeasurement.exploreQuery} />
			</div>
		{/if}
	</div>
</div>
