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
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';

	// ─── Constants ────────────────────────────────────────────────────────────

	const TIME_RANGES = [
		{ label: '5m', minutes: 5 },
		{ label: '15m', minutes: 15 },
		{ label: '1h', minutes: 60 },
		{ label: '6h', minutes: 360 },
		{ label: '24h', minutes: 1440 }
	] as const;

	const CHART_COLORS = [
		'var(--chart-1)',
		'var(--chart-2)',
		'var(--chart-3)',
		'var(--chart-4)',
		'var(--chart-5)'
	];

	const MAX_AUTO_FIELDS = 5;

	// ─── State ────────────────────────────────────────────────────────────────

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedMeasurement = $state<TelemetryDescriptor | null>(null);
	let timeRangeMinutes = $state(5);
	let selectedFields = $state<string[]>([]);
	let autoRefresh = $state(false);
	let autoRefreshInterval = $state<ReturnType<typeof setInterval> | null>(null);

	// ─── Adapters ─────────────────────────────────────────────────────────────

	// Flatten all descriptors for the DescriptorPanel (which needs Descriptor type)
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

	// Full TelemetryDescriptor for the selected item (to get fields)
	let allTelemetryDescriptors = $derived(telemetryStore.measurements.flatMap((g) => g.descriptors));

	// ─── Chart config ─────────────────────────────────────────────────────────

	let chartConfig = $derived.by((): ChartConfig => {
		const config: ChartConfig = {};
		selectedFields.forEach((field, i) => {
			config[field] = { label: field, color: CHART_COLORS[i % CHART_COLORS.length] };
		});
		return config;
	});

	// ─── Context & lifecycle ──────────────────────────────────────────────────

	const deviceCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('device');
	deviceCtx.setTabRefresh(() => {
		if (mirStore.mir && deviceId) {
			telemetryStore.loadMeasurements(mirStore.mir, deviceId);
		}
	});
	onDestroy(() => {
		deviceCtx.setTabRefresh(null);
		stopAutoRefresh();
	});

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedMeasurement = null;
			selectedFields = [];
			telemetryStore.reset();
			telemetryStore.loadMeasurements(mirStore.mir, deviceId);
		}
	});

	// ─── Query logic ──────────────────────────────────────────────────────────

	function getTimeRange(): { start: Date; end: Date } {
		const end = new Date();
		const start = new Date(end.getTime() - timeRangeMinutes * 60 * 1000);
		return { start, end };
	}

	function runQuery() {
		if (!mirStore.mir || !selectedMeasurement) return;
		const { start, end } = getTimeRange();
		telemetryStore.queryMeasurement(
			mirStore.mir,
			deviceId,
			selectedMeasurement.name,
			selectedFields,
			start,
			end
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

	function setTimeRange(minutes: number) {
		timeRangeMinutes = minutes;
		runQuery();
	}

	function toggleField(field: string) {
		if (selectedFields.includes(field)) {
			if (selectedFields.length > 1) {
				selectedFields = selectedFields.filter((f) => f !== field);
			}
		} else {
			selectedFields = [...selectedFields, field];
		}
		runQuery();
	}

	// ─── Auto-refresh ─────────────────────────────────────────────────────────

	function startAutoRefresh() {
		stopAutoRefresh();
		autoRefreshInterval = setInterval(runQuery, 30_000);
	}

	function stopAutoRefresh() {
		if (autoRefreshInterval !== null) {
			clearInterval(autoRefreshInterval);
			autoRefreshInterval = null;
		}
	}

	function toggleAutoRefresh() {
		autoRefresh = !autoRefresh;
		if (autoRefresh) {
			startAutoRefresh();
		} else {
			stopAutoRefresh();
		}
	}
</script>

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
	<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
		{#if !selectedMeasurement}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<ActivityIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a measurement to view chart</p>
			</div>
		{:else}
			<!-- Toolbar -->
			<div class="flex flex-wrap items-center gap-2 border-b px-4 py-2">
				<!-- Time range -->
				<div class="flex items-center gap-1">
					{#each TIME_RANGES as range (range.label)}
						<button
							onclick={() => setTimeRange(range.minutes)}
							class="rounded px-2 py-0.5 text-xs transition-colors
								{timeRangeMinutes === range.minutes
								? 'bg-primary text-primary-foreground'
								: 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground'}"
						>
							{range.label}
						</button>
					{/each}
				</div>

				<div class="h-4 w-px bg-border"></div>

				<!-- Field toggles -->
				<div class="flex flex-wrap gap-1">
					{#each selectedMeasurement.fields as field, i (field)}
						<button
							onclick={() => toggleField(field)}
							class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
								{selectedFields.includes(field)
								? 'border-transparent text-white'
								: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
							style={selectedFields.includes(field)
								? `background: ${CHART_COLORS[i % CHART_COLORS.length]};`
								: ''}
						>
							{field}
						</button>
					{/each}
				</div>

				<div class="ml-auto flex items-center gap-1.5">
					<!-- Manual refresh -->
					<button
						onclick={runQuery}
						disabled={telemetryStore.isQuerying}
						class="flex items-center gap-1 rounded p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
						title="Refresh"
					>
						<RefreshCwIcon class="size-3.5 {telemetryStore.isQuerying ? 'animate-spin' : ''}" />
					</button>

					<!-- Auto-refresh toggle -->
					<button
						onclick={toggleAutoRefresh}
						class="rounded px-2 py-0.5 text-xs transition-colors
							{autoRefresh
							? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
							: 'bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground'}"
					>
						Auto
					</button>
				</div>
			</div>

			<!-- Measurement name -->
			<div class="border-b px-4 py-2">
				<span class="font-mono text-sm font-medium">{selectedMeasurement.name}</span>
				{#if selectedMeasurement.error}
					<span class="ml-2 text-xs text-destructive">{selectedMeasurement.error}</span>
				{/if}
			</div>

			<!-- Chart -->
			<div class="flex-1 overflow-auto px-4 py-4">
				{#if telemetryStore.queryError}
					<p class="text-sm text-destructive">{telemetryStore.queryError}</p>
				{:else if telemetryStore.isQuerying && !telemetryStore.queryData}
					<div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
						Loading data…
					</div>
				{:else if telemetryStore.queryData}
					<TlmChart data={telemetryStore.queryData} {selectedFields} {chartConfig} />
				{/if}
			</div>
		{/if}
	</div>
</div>
