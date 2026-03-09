<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { TelemetryGroup, TelemetryDescriptor, QueryData, QueryRow } from '@mir/sdk';
	import DescriptorPanel from '$lib/domains/devices/components/commands/descriptor-panel.svelte';
	import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import type { Descriptor } from '$lib/domains/devices/types/types';
	import type { DateRange } from 'bits-ui';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';

	// ─── Constants ────────────────────────────────────────────────────────────

	const CHART_COLORS = [
		'var(--chart-1)',
		'var(--chart-2)',
		'var(--chart-3)',
		'var(--chart-4)',
		'var(--chart-5)'
	];

	const PRESETS = [
		{ label: '5m', minutes: 5 },
		{ label: '15m', minutes: 15 },
		{ label: '30m', minutes: 30 },
		{ label: '1h', minutes: 60 },
		{ label: '3h', minutes: 180 },
		{ label: '6h', minutes: 360 },
		{ label: '24h', minutes: 1440 },
		{ label: '7d', minutes: 10080 }
	] as const;

	type TimeFilter =
		| { mode: 'relative'; minutes: number }
		| { mode: 'absolute'; start: Date; end: Date };

	type GroupState = {
		group: TelemetryGroup;
		selectedMeasurement: TelemetryDescriptor | null;
		mergedData: QueryData | null;
		isQuerying: boolean;
		queryError: string | null;
		chartConfig: ChartConfig;
		mergedFields: string[];
	};

	let tlmGroups = $state<TelemetryGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let groupStates = $state<GroupState[]>([]);

	// Cancellation guard for parallel queries
	let generation = 0;

	// Shared time filter
	let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 5 });
	let popoverOpen = $state(false);
	let calendarValue = $state<DateRange | undefined>(undefined);

	// Guard against $effect re-triggering when onValueChange sets timeFilter
	let _calendarSyncInProgress = false;

	// ─── Load measurements ────────────────────────────────────────────────────

	$effect(() => {
		if (!mirStore.mir || selectionStore.count === 0) return;
		loadMeasurements();
	});

	async function loadMeasurements() {
		if (!mirStore.mir) return;
		generation++;
		isLoading = true;
		error = null;
		try {
			const allIds = selectionStore.selectedDevices.map((d) => d.spec.deviceId);
			const target = new DeviceTarget({ ids: allIds });
			const groups = await mirStore.mir.client().listTelemetry().request(target);
			tlmGroups = groups;
			groupStates = groups.map((g) => ({
				group: g,
				selectedMeasurement: null,
				mergedData: null,
				isQuerying: false,
				queryError: null,
				chartConfig: {},
				mergedFields: []
			}));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load telemetry';
		} finally {
			isLoading = false;
		}
	}

	// ─── Time helpers ─────────────────────────────────────────────────────────

	function getTimeRange(): { start: Date; end: Date } {
		if (timeFilter.mode === 'absolute') {
			const start = timeFilter.start;
			const end =
				timeFilter.end.getTime() <= start.getTime()
					? new Date(start.getTime() + 1000)
					: timeFilter.end;
			return { start, end };
		}
		const end = new Date();
		const start = new Date(end.getTime() - timeFilter.minutes * 60 * 1000);
		return { start, end };
	}

	function getAggregationWindow(start: Date, end: Date): string | undefined {
		const hours = (end.getTime() - start.getTime()) / (1000 * 60 * 60);
		if (hours < 1) return undefined;
		if (hours < 6) return '10s';
		if (hours < 24) return '1m';
		if (hours < 168) return '10m';
		if (hours < 720) return '1h';
		return '6h';
	}

	let timeFilterLabel = $derived.by(() => {
		if (timeFilter.mode === 'relative') {
			const preset = PRESETS.find((p) => p.minutes === timeFilter.minutes);
			return `Last ${preset?.label ?? timeFilter.minutes + 'm'}`;
		}
		const fmt = (d: Date) =>
			d.toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
		return `${fmt(timeFilter.start)} – ${fmt(timeFilter.end)}`;
	});

	// ─── Query and merge ──────────────────────────────────────────────────────

	async function queryGroup(groupIdx: number) {
		if (!mirStore.mir) return;
		const gs = groupStates[groupIdx];
		if (!gs.selectedMeasurement) return;

		// Capture generation and stable fields before any await
		const myGen = generation;
		const group = gs.group;
		const selectedMeasurement = gs.selectedMeasurement;

		groupStates[groupIdx] = {
			group: gs.group,
			selectedMeasurement: gs.selectedMeasurement,
			mergedData: gs.mergedData,
			mergedFields: gs.mergedFields,
			chartConfig: gs.chartConfig,
			isQuerying: true,
			queryError: null
		};

		try {
			const { start, end } = getTimeRange();
			const aggWindow = getAggregationWindow(start, end);
			const fields = selectedMeasurement.fields;

			// Query each device in the group in parallel
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

			// Discard if a newer loadMeasurements() has run
			if (myGen !== generation) return;

			// Build merged fields list and chartConfig with per-device colors
			const mergedFields: string[] = [];
			const newChartConfig: ChartConfig = {};
			let colorIdx = 0;

			results.forEach(({ deviceName, data }) => {
				data.headers
					.filter((h) => h !== '_time')
					.forEach((field) => {
						const key = `${deviceName}_${field}`;
						mergedFields.push(key);
						newChartConfig[key] = {
							label: `${deviceName}: ${field}`,
							color: CHART_COLORS[colorIdx % CHART_COLORS.length]
						};
						colorIdx++;
					});
			});

			// Merge rows by _time
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
						.filter((h) => h !== '_time')
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

			groupStates[groupIdx] = {
				group,
				selectedMeasurement,
				isQuerying: false,
				mergedData: { headers: ['_time', ...mergedFields], rows: sortedRows },
				mergedFields,
				chartConfig: newChartConfig,
				queryError: null
			};
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Query failed';

			// Capture current state fields before writing back (no post-await spread)
			const current = groupStates[groupIdx];

			activityStore.add({
				kind: 'error',
				category: 'Telemetry',
				title: current.selectedMeasurement?.name ?? 'query',
				error: message
			});

			// Discard if a newer loadMeasurements() has run
			if (myGen !== generation) return;

			groupStates[groupIdx] = {
				group: current.group,
				selectedMeasurement: current.selectedMeasurement,
				mergedData: current.mergedData,
				mergedFields: current.mergedFields,
				chartConfig: current.chartConfig,
				isQuerying: false,
				queryError: message
			};
		}
	}

	function selectMeasurement(groupIdx: number, desc: Descriptor) {
		const gs = groupStates[groupIdx];
		const full = gs.group.descriptors.find((d) => d.name === desc.name);
		if (!full) return;
		groupStates[groupIdx] = {
			...gs,
			selectedMeasurement: full,
			mergedData: null,
			queryError: null
		};
		queryGroup(groupIdx);
	}

	function selectPreset(minutes: number) {
		timeFilter = { mode: 'relative', minutes };
		popoverOpen = false;
		groupStates.forEach((_, idx) => {
			if (groupStates[idx].selectedMeasurement) queryGroup(idx);
		});
	}

	$effect(() => {
		if (timeFilter.mode === 'absolute' && !_calendarSyncInProgress) {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			calendarValue = {
				start: fromDate(timeFilter.start, tz),
				end: fromDate(timeFilter.end, tz)
			};
		}
	});
</script>

<div class="flex flex-col gap-6">
	<!-- Shared time picker -->
	<div class="flex items-center justify-end">
		<Popover.Root bind:open={popoverOpen}>
			<Popover.Trigger>
				{#snippet child({ props })}
					<button
						{...props}
						class="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
					>
						<CalendarIcon class="size-3.5 text-muted-foreground" />
						<span>{timeFilterLabel}</span>
						<ChevronDownIcon class="size-3 text-muted-foreground" />
					</button>
				{/snippet}
			</Popover.Trigger>
			<Popover.Content class="w-auto p-0 shadow-lg" align="end">
				<div class="flex">
					<div class="p-5">
						<RangeCalendar
							bind:value={calendarValue}
							onValueChange={(v) => {
								if (!v?.start || !v?.end) return;
								const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
								_calendarSyncInProgress = true;
								timeFilter = {
									mode: 'absolute',
									start: v.start.toDate(tz),
									end: v.end.toDate(tz)
								};
								_calendarSyncInProgress = false;
								groupStates.forEach((_, idx) => {
									if (groupStates[idx].selectedMeasurement) queryGroup(idx);
								});
							}}
							numberOfMonths={1}
						/>
					</div>
					<div class="w-px self-stretch bg-border"></div>
					<div class="relative w-36 self-stretch">
						<div class="absolute inset-0 flex flex-col p-3">
							<p class="mb-2 px-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase">
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
										Last {preset.label}
									</button>
								{/each}
							</div>
						</div>
					</div>
				</div>
			</Popover.Content>
		</Popover.Root>
	</div>

	{#if isLoading}
		<div class="flex items-center justify-center py-12 text-muted-foreground">
			<Spinner class="mr-2 size-4" />
			Loading telemetry...
		</div>
	{:else if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if tlmGroups.length === 0}
		<div class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
			<ActivityIcon class="size-8 opacity-30" />
			<p class="text-sm">No telemetry found for selected devices</p>
		</div>
	{:else}
		{#each groupStates as gs, idx (idx)}
			<div>
				<div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
					<span class="font-mono font-medium text-foreground">
						{gs.group.ids.map((id) => `${id.name}/${id.namespace}`).join(', ')}
					</span>
					<span>({gs.group.ids.length} device{gs.group.ids.length > 1 ? 's' : ''})</span>
				</div>

				<div class="flex min-h-80 overflow-hidden rounded-lg border">
					<DescriptorPanel
						title="Telemetry"
						items={gs.group.descriptors.map((d) => ({
							name: d.name,
							labels: d.labels,
							template: '',
							error: d.error
						}))}
						isLoading={false}
						error={null}
						groupErrors={gs.group.error ? [gs.group.error] : []}
						selectedName={gs.selectedMeasurement?.name ?? null}
						emptyText="No telemetry."
						onSelect={(desc) => selectMeasurement(idx, desc)}
					/>

					<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
						{#if !gs.selectedMeasurement}
							<div
								class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground"
							>
								<ActivityIcon class="size-8 opacity-30" />
								<p class="text-sm">Select a measurement to view chart</p>
							</div>
						{:else}
							<!-- Legend -->
							<div class="flex flex-wrap items-center gap-2 border-b px-4 py-1.5">
								{#each gs.mergedFields as field, i (field)}
									<span class="flex items-center gap-1 font-mono text-[11px]">
										<span
											class="h-2 w-2 rounded-full"
											style="background: {gs.chartConfig[field]?.color ?? CHART_COLORS[i % CHART_COLORS.length]}"
										></span>
										{gs.chartConfig[field]?.label ?? field}
									</span>
								{/each}
							</div>

							<!-- Chart -->
							<div class="flex-1 px-4 py-4">
								{#if gs.queryError}
									<p class="text-sm text-destructive">{gs.queryError}</p>
								{:else if gs.isQuerying && !gs.mergedData}
									<div
										class="flex h-48 items-center justify-center text-sm text-muted-foreground"
									>
										Loading data…
									</div>
								{:else if gs.mergedData}
									<TlmChart
										data={gs.mergedData}
										selectedFields={gs.mergedFields}
										chartConfig={gs.chartConfig}
										useUtc={editorPrefs.utc}
										chartClass="h-72"
									/>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			</div>

			{#if idx < groupStates.length - 1}
				<Separator />
			{/if}
		{/each}
	{/if}
</div>
