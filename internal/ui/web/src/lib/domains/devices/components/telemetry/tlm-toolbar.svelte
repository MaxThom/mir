<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { DateRange } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ZoomInIcon from '@lucide/svelte/icons/zoom-in';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import RotateCcwIcon from '@lucide/svelte/icons/rotate-ccw';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';

	type TimeFilter =
		| { mode: 'relative'; minutes: number }
		| { mode: 'absolute'; start: Date; end: Date };

	let {
		measurementName = null,
		measurementError = null,
		grafanaUrl = null,
		timeFilter = $bindable<TimeFilter>({ mode: 'relative', minutes: 5 }),
		calendarValue = $bindable<DateRange | undefined>(undefined),
		hasZoomed = $bindable(false),
		fullscreen = $bindable(false),
		queryData = null,
		presets,
		onQuery,
		onCalendarChange = undefined,
		calendarTop = undefined,
		toolbarEnd = undefined
	}: {
		measurementName?: string | null;
		measurementError?: string | null;
		grafanaUrl?: string | null;
		timeFilter?: TimeFilter;
		calendarValue?: DateRange | undefined;
		hasZoomed?: boolean;
		fullscreen?: boolean;
		queryData?: QueryData | null;
		presets: readonly { label: string; minutes: number }[];
		onQuery: () => void;
		// Optional override: return a TimeFilter to use instead of the default date-only conversion
		onCalendarChange?: (v: DateRange | undefined) => TimeFilter | undefined;
		// Optional extra content above the calendar (e.g. time inputs for single-device)
		calendarTop?: Snippet;
		// Optional extra buttons rendered just before the fullscreen button
		toolbarEnd?: Snippet;
	} = $props();

	let popoverOpen = $state(false);
	let copied = $state(false);
	let baseTimeFilter = $state<TimeFilter>(timeFilter);
	let _calendarSyncInProgress = false;

	// Sync calendarValue when timeFilter changes to absolute (e.g. from brush zoom)
	$effect(() => {
		if (timeFilter.mode === 'absolute' && !_calendarSyncInProgress) {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			calendarValue = {
				start: fromDate(timeFilter.start, tz),
				end: fromDate(timeFilter.end, tz)
			};
		}
	});

	let timeFilterLabel = $derived.by(() => {
		const f = timeFilter;
		if (f.mode === 'relative') {
			const preset = presets.find((p) => p.minutes === f.minutes);
			return `Last ${preset?.label ?? f.minutes + 'm'}`;
		}
		const tz = editorPrefs.utc ? 'UTC' : undefined;
		const fmt = (d: Date) =>
			d.toLocaleDateString([], {
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
				timeZone: tz
			});
		return `${fmt(f.start)} – ${fmt(f.end)}${editorPrefs.utc ? ' (UTC)' : ''}`;
	});

	function getTimeRange(): { start: Date; end: Date } {
		const f = timeFilter;
		if (f.mode === 'absolute') {
			const start = f.start;
			const end = f.end.getTime() <= start.getTime() ? new Date(start.getTime() + 1000) : f.end;
			return { start, end };
		}
		const end = new Date();
		const start = new Date(end.getTime() - f.minutes * 60 * 1000);
		return { start, end };
	}

	function selectPreset(minutes: number) {
		timeFilter = { mode: 'relative', minutes };
		baseTimeFilter = timeFilter;
		hasZoomed = false;
		calendarValue = undefined;
		popoverOpen = false;
		onQuery();
	}

	function zoom(factor: number) {
		const { start, end } = getTimeRange();
		const delta = (end.getTime() - start.getTime()) * 0.25 * factor;
		const newStart = new Date(start.getTime() + delta);
		const newEnd = new Date(end.getTime() - delta);
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		hasZoomed = true;
		onQuery();
	}

	function resetZoom() {
		timeFilter = baseTimeFilter;
		hasZoomed = false;
		onQuery();
	}

	function handleCalendarValueChange(v: DateRange | undefined) {
		if (!v?.start || !v?.end) return;
		const override = onCalendarChange?.(v);
		_calendarSyncInProgress = true;
		if (override) {
			timeFilter = override;
		} else {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			timeFilter = { mode: 'absolute', start: v.start.toDate(tz), end: v.end.toDate(tz) };
		}
		_calendarSyncInProgress = false;
		baseTimeFilter = timeFilter;
		hasZoomed = false;
		onQuery();
	}

	function dateToRFC3339(date: Date): string {
		if (editorPrefs.utc) return date.toISOString();
		const offset = -date.getTimezoneOffset();
		const sign = offset >= 0 ? '+' : '-';
		const pad = (n: number, w = 2) => String(n).padStart(w, '0');
		const tz = `${sign}${pad(Math.floor(Math.abs(offset) / 60))}:${pad(Math.abs(offset) % 60)}`;
		return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}${tz}`;
	}

	async function copyAsCsv() {
		if (!queryData) return;
		const { headers, rows } = queryData;
		const lines = [
			headers.join(','),
			...rows.map((row) =>
				headers
					.map((h) => {
						const v = row.values[h] ?? null;
						if (v === null || v === undefined) return '';
						if (v instanceof Date) return dateToRFC3339(v);
						if (typeof v === 'boolean') return v ? 'true' : 'false';
						return String(v);
					})
					.join(',')
			)
		];
		await navigator.clipboard.writeText(lines.join('\n'));
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="flex shrink-0 items-center gap-2 border-b px-4 py-1.5">
	{#if measurementName}
		<span class="font-mono text-sm font-medium">{measurementName}</span>
		{#if grafanaUrl}
			<a
				href={grafanaUrl}
				target="_blank"
				rel="noreferrer"
				title="Open in Grafana"
				class="inline-flex -translate-y-0.5 items-center text-muted-foreground transition-colors hover:text-foreground"
			>
				<ExternalLinkIcon class="size-3.5" />
			</a>
		{/if}
		{#if measurementError}
			<span class="text-xs text-destructive">{measurementError}</span>
		{/if}
	{/if}

	<div class="ml-auto flex items-center gap-1">
		{#if hasZoomed}
			<button
				onclick={resetZoom}
				title="Reset to last selection"
				class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
			>
				<RotateCcwIcon class="size-3.5" />
			</button>
		{/if}
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
					<div class="p-5">
						{@render calendarTop?.()}
						<RangeCalendar
							bind:value={calendarValue}
							onValueChange={handleCalendarValueChange}
							numberOfMonths={1}
						/>
					</div>
					<div class="w-px self-stretch bg-border"></div>
					<div class="relative w-36 self-stretch">
						<div class="absolute inset-0 flex flex-col p-3">
							<p
								class="mb-2 px-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase"
							>
								Quick range
							</p>
							<div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto">
								{#each presets as preset (preset.label)}
									<button
										onclick={() => selectPreset(preset.minutes)}
										class="flex items-center justify-between rounded-md px-2 py-1.5 text-left text-xs transition-colors
										{timeFilter.mode === 'relative' && timeFilter.minutes === preset.minutes
											? 'bg-primary font-medium text-primary-foreground'
											: 'text-foreground hover:bg-accent hover:text-accent-foreground'}"
									>
										<span>Last {preset.label}</span>
										{#if timeFilter.mode === 'relative' && timeFilter.minutes === preset.minutes}
											<span class="size-1.5 rounded-full bg-primary-foreground opacity-70"></span>
										{/if}
									</button>
								{/each}
							</div>
						</div>
					</div>
				</div>
			</Popover.Content>
		</Popover.Root>
		{@render toolbarEnd?.()}
		<button
			onclick={() => (fullscreen = !fullscreen)}
			title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
			class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
		>
			{#if fullscreen}
				<MinimizeIcon class="size-3.5" />
			{:else}
				<MaximizeIcon class="size-3.5" />
			{/if}
		</button>
		<button
			onclick={copyAsCsv}
			disabled={!queryData}
			title="Copy data as CSV"
			class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-40"
		>
			{#if copied}
				<CheckIcon class="size-3.5 text-green-500" />
			{:else}
				<CopyIcon class="size-3.5" />
			{/if}
		</button>
	</div>
</div>
