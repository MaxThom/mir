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
	import TimePicker from '$lib/domains/devices/components/telemetry/time-picker.svelte';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ZoomInIcon from '@lucide/svelte/icons/zoom-in';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import type { DateRange } from 'bits-ui';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';
	import { SvelteDate } from 'svelte/reactivity';

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

	const CHART_COLORS = [
		'var(--chart-1)',
		'var(--chart-2)',
		'var(--chart-3)',
		'var(--chart-4)',
		'var(--chart-5)'
	];

	const MAX_AUTO_FIELDS = 5;

	// ─── Types ────────────────────────────────────────────────────────────────

	type TimeFilter =
		| { mode: 'relative'; minutes: number }
		| { mode: 'absolute'; start: Date; end: Date };

	// ─── State ────────────────────────────────────────────────────────────────

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedMeasurement = $state<TelemetryDescriptor | null>(null);
	let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 5 });
	let selectedFields = $state<string[]>([]);
	let popoverOpen = $state(false);
	let calendarValue = $state<DateRange | undefined>(undefined);
	let startTime = $state('00:00');
	let endTime = $state('23:59');

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

	// ─── Time filter label ────────────────────────────────────────────────────

	let timeFilterLabel = $derived.by(() => {
		if (timeFilter.mode === 'relative') {
			const preset = PRESETS.find((p) => p.minutes === timeFilter.minutes);
			return `Last ${preset?.label ?? timeFilter.minutes + 'm'}`;
		}
		const fmt = (d: Date) =>
			d.toLocaleDateString([], {
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		return `${fmt(timeFilter.start)} – ${fmt(timeFilter.end)}`;
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

	// ─── Query logic ──────────────────────────────────────────────────────────

	function getTimeRange(): { start: Date; end: Date } {
		if (timeFilter.mode === 'absolute') {
			return { start: timeFilter.start, end: timeFilter.end };
		}
		const end = new Date();
		const start = new Date(end.getTime() - timeFilter.minutes * 60 * 1000);
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

	function selectPreset(minutes: number) {
		timeFilter = { mode: 'relative', minutes };
		calendarValue = undefined;
		runQuery();
		popoverOpen = false;
	}

	function zoom(factor: number) {
		const { start, end } = getTimeRange();
		const delta = (end.getTime() - start.getTime()) * 0.25 * factor;
		const newStart = new Date(start.getTime() + delta);
		const newEnd = new Date(end.getTime() - delta);
		const tz = getLocalTimeZone();
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		calendarValue = { start: fromDate(newStart, tz), end: fromDate(newEnd, tz) };
		startTime = `${String(newStart.getHours()).padStart(2, '0')}:${String(newStart.getMinutes()).padStart(2, '0')}`;
		endTime = `${String(newEnd.getHours()).padStart(2, '0')}:${String(newEnd.getMinutes()).padStart(2, '0')}`;
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

	// Run query when popover closes and an absolute range is set
	let prevPopoverOpen = false;
	$effect(() => {
		if (prevPopoverOpen && !popoverOpen && timeFilter.mode === 'absolute') {
			runQuery();
		}
		prevPopoverOpen = popoverOpen;
	});

	function handleCalendarChange(value: DateRange | undefined) {
		if (value?.start && value?.end) {
			const [startH, startM] = startTime.split(':').map(Number);
			const [endH, endM] = endTime.split(':').map(Number);
			const start = value.start.toDate(getLocalTimeZone());
			start.setHours(startH, startM, 0, 0);
			const end = value.end.toDate(getLocalTimeZone());
			end.setHours(endH, endM, 59, 999);
			timeFilter = { mode: 'absolute', start, end };
		}
	}

	function handleTimeInputChange() {
		if (timeFilter.mode === 'absolute') {
			const [startH, startM] = startTime.split(':').map(Number);
			const [endH, endM] = endTime.split(':').map(Number);
			const start = new SvelteDate(timeFilter.start);
			start.setHours(startH, startM, 0, 0);
			const end = new SvelteDate(timeFilter.end);
			end.setHours(endH, endM, 59, 999);
			timeFilter = { mode: 'absolute', start, end };
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
			<!-- Measurement name -->
			<div class="border-b px-4 py-1.75">
				<span class="font-mono text-sm font-medium">{selectedMeasurement.name}</span>
				{#if selectedMeasurement.error}
					<span class="ml-2 text-xs text-destructive">{selectedMeasurement.error}</span>
				{/if}
			</div>

			<!-- Toolbar -->
			<div class="flex flex-wrap items-center gap-2 border-b px-4 py-1.25">
				<!-- Field toggles -->
				<div class="flex flex-wrap gap-1">
					{#each selectedMeasurement.fields as field, i (field)}
						<button
							onclick={(e) => toggleField(field, e.shiftKey)}
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

				<!-- Zoom controls + time range picker (far right) -->
				<div class="ml-auto flex items-center gap-1">
					<button
						onclick={() => zoom(1)}
						title="Zoom in"
						class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
					>
						<ZoomInIcon class="size-3.5" />
					</button>
					<button
						onclick={() => zoom(-1)}
						title="Zoom out"
						class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
					>
						<ZoomOutIcon class="size-3.5" />
					</button>
				</div>
				<Popover.Root bind:open={popoverOpen}>
					<Popover.Trigger>
						{#snippet child({ props })}
							<button
								{...props}
								class="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
									{popoverOpen ? 'border-ring ring-1 ring-ring' : ''}"
							>
								<CalendarIcon class="size-3.5 text-muted-foreground" />
								<span>{timeFilterLabel}</span>
								<ChevronDownIcon
									class="size-3 text-muted-foreground transition-transform {popoverOpen
										? 'rotate-180'
										: ''}"
								/>
							</button>
						{/snippet}
					</Popover.Trigger>
					<Popover.Content class="w-auto p-0 shadow-lg" align="end">
						<div class="flex">
							<!-- Left: calendar + time inputs -->
							<div class="p-5">
								<p
									class="mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase"
								>
									Custom range
								</p>
								<div class="mb-3 grid grid-cols-2 gap-3">
									<div class="space-y-1.5">
										<label class="text-xs font-medium text-muted-foreground">Start time</label>
										<TimePicker bind:value={startTime} onchange={handleTimeInputChange} />
									</div>
									<div class="space-y-1.5">
										<label class="text-xs font-medium text-muted-foreground">End time</label>
										<TimePicker bind:value={endTime} onchange={handleTimeInputChange} />
									</div>
								</div>
								<RangeCalendar
									bind:value={calendarValue}
									onValueChange={handleCalendarChange}
									numberOfMonths={1}
								/>
							</div>

							<!-- Divider -->
							<div class="w-px self-stretch bg-border"></div>

							<!-- Right: presets -->
							<div class="relative w-36 self-stretch">
								<div class="absolute inset-0 flex flex-col p-3">
									<p
										class="mb-2 px-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase"
									>
										Quick range
									</p>
									<div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto">
										{#each PRESETS as preset (preset.label)}
											<button
												onclick={() => selectPreset(preset.minutes)}
												class="flex items-center justify-between rounded-md px-2 py-1.5 text-left text-xs transition-colors
												{timeFilter.mode === 'relative' && timeFilter.minutes === preset.minutes
													? 'bg-primary font-medium text-primary-foreground'
													: 'text-foreground hover:bg-accent hover:text-accent-foreground'}"
											>
												<span>Last {preset.label}</span>
												{#if timeFilter.mode === 'relative' && timeFilter.minutes === preset.minutes}
													<span class="size-1.5 rounded-full bg-primary-foreground opacity-70"
													></span>
												{/if}
											</button>
										{/each}
									</div>
								</div>
							</div>
						</div>
					</Popover.Content>
				</Popover.Root>
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
