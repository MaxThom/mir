<script lang="ts">
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';
	import TimePicker from './time-picker.svelte';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';
	import type { TimeFilter } from '../../utils/tlm-time';

	let {
		timeFilter,
		presets = undefined,
		ontimechange,
		fullWidth = false
	}: {
		timeFilter: TimeFilter;
		presets?: readonly { label: string; minutes: number }[];
		ontimechange: (filter: TimeFilter) => void;
		fullWidth?: boolean;
	} = $props();

	let popoverOpen = $state(false);
	let calendarValue = $state<DateRange | undefined>(undefined);
	let startTime = $state('00:00');
	let endTime = $state('23:59');
	let _syncInProgress = false;

	// Sync calendar + time inputs when timeFilter becomes absolute (e.g. brush zoom)
	$effect(() => {
		if (timeFilter.mode === 'absolute' && !_syncInProgress) {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			calendarValue = {
				start: fromDate(timeFilter.start, tz),
				end: fromDate(timeFilter.end, tz)
			};
			const getH = (d: Date) => (editorPrefs.utc ? d.getUTCHours() : d.getHours());
			const getM = (d: Date) => (editorPrefs.utc ? d.getUTCMinutes() : d.getMinutes());
			startTime = `${String(getH(timeFilter.start)).padStart(2, '0')}:${String(getM(timeFilter.start)).padStart(2, '0')}`;
			endTime = `${String(getH(timeFilter.end)).padStart(2, '0')}:${String(getM(timeFilter.end)).padStart(2, '0')}`;
		}
	});

	const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
	function fmtDate(d: Date): string {
		const mo = editorPrefs.utc ? d.getUTCMonth() : d.getMonth();
		const day = editorPrefs.utc ? d.getUTCDate() : d.getDate();
		const h = editorPrefs.utc ? d.getUTCHours() : d.getHours();
		const m = editorPrefs.utc ? d.getUTCMinutes() : d.getMinutes();
		return `${MONTHS[mo]} ${day} ${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
	}

	let timeFilterLabel = $derived.by(() => {
		if (timeFilter.mode === 'relative') {
			const preset = presets?.find((p) => p.minutes === timeFilter.minutes);
			return `Last ${preset?.label ?? timeFilter.minutes + 'm'}`;
		}
		return `${fmtDate(timeFilter.start)} – ${fmtDate(timeFilter.end)}`;
	});

	function applyTimes(v: DateRange | undefined, sTime: string, eTime: string) {
		if (!v?.start || !v?.end) return;
		const [sh, sm] = sTime.split(':').map(Number);
		const [eh, em] = eTime.split(':').map(Number);
		const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
		const start = v.start.toDate(tz);
		const end = v.end.toDate(tz);
		if (editorPrefs.utc) {
			start.setUTCHours(sh, sm, 0, 0);
			end.setUTCHours(eh, em, 59, 999);
		} else {
			start.setHours(sh, sm, 0, 0);
			end.setHours(eh, em, 59, 999);
		}
		_syncInProgress = true;
		ontimechange({ mode: 'absolute', start, end });
		_syncInProgress = false;
	}

	function handleCalendarValueChange(v: DateRange | undefined) {
		applyTimes(v, startTime, endTime);
	}

	function handleTimeChange() {
		if (timeFilter.mode === 'absolute') {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			const cv = calendarValue ?? {
				start: fromDate(timeFilter.start, tz),
				end: fromDate(timeFilter.end, tz)
			};
			applyTimes(cv, startTime, endTime);
		}
	}

	function selectPreset(minutes: number) {
		calendarValue = undefined;
		popoverOpen = false;
		ontimechange({ mode: 'relative', minutes });
	}
</script>

<Popover.Root bind:open={popoverOpen}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				aria-label="Time range"
				class="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
					{fullWidth ? 'w-full justify-between' : ''}
					{popoverOpen ? 'border-ring ring-1 ring-ring' : ''}"
			>
				<CalendarIcon class="size-3.5 text-muted-foreground" />
				<span>{timeFilterLabel}</span>
				<ChevronDownIcon
					class="size-3 text-muted-foreground transition-transform {popoverOpen ? 'rotate-180' : ''}"
				/>
			</button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="w-auto p-0 shadow-lg" align="end">
		<div class="flex">
			<div class="p-5">
				<p class="mb-3 text-xs font-semibold tracking-wider text-muted-foreground uppercase">
					Custom range{editorPrefs.utc ? ' (UTC)' : ''}
				</p>
				<div class="mb-3 grid grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<span class="text-xs font-medium text-muted-foreground">Start time</span>
						<TimePicker bind:value={startTime} onchange={handleTimeChange} />
					</div>
					<div class="space-y-1.5">
						<span class="text-xs font-medium text-muted-foreground">End time</span>
						<TimePicker bind:value={endTime} onchange={handleTimeChange} />
					</div>
				</div>
				<RangeCalendar
					bind:value={calendarValue}
					onValueChange={handleCalendarValueChange}
					numberOfMonths={1}
				/>
			</div>
			{#if presets}
				<div class="w-px self-stretch bg-border"></div>
				<div class="relative w-36 self-stretch">
					<div class="absolute inset-0 flex flex-col p-3">
						<p class="mb-2 px-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase">
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
			{/if}
		</div>
	</Popover.Content>
</Popover.Root>
