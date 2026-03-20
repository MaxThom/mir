<script lang="ts">
	import type { DateRange } from 'bits-ui';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { getLocalTimeZone, fromDate } from '@internationalized/date';
	import type { TimeFilter } from '$lib/domains/devices/utils/tlm-time';

	export const PRESETS = [
		{ label: '5m',  minutes: 5    },
		{ label: '15m', minutes: 15   },
		{ label: '30m', minutes: 30   },
		{ label: '1h',  minutes: 60   },
		{ label: '3h',  minutes: 180  },
		{ label: '6h',  minutes: 360  },
		{ label: '12h', minutes: 720  },
		{ label: '24h', minutes: 1440 },
		{ label: '2d',  minutes: 2880 },
		{ label: '7d',  minutes: 10080},
		{ label: '30d', minutes: 43200},
	] as const;

	let {
		timeFilter = $bindable<TimeFilter>({ mode: 'relative', minutes: 60 })
	}: {
		timeFilter: TimeFilter;
	} = $props();

	let popoverOpen = $state(false);
	let calendarValue = $state<DateRange | undefined>(undefined);

	$effect(() => {
		if (timeFilter.mode === 'absolute') {
			const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
			calendarValue = {
				start: fromDate(timeFilter.start, tz),
				end: fromDate(timeFilter.end, tz)
			};
		}
	});

	let label = $derived.by(() => {
		const tf = timeFilter;
		if (tf.mode === 'relative') {
			const preset = PRESETS.find((p) => p.minutes === tf.minutes);
			return `Last ${preset?.label ?? tf.minutes + 'm'}`;
		}
		const tz = editorPrefs.utc ? 'UTC' : undefined;
		const fmt = (d: Date) =>
			d.toLocaleDateString([], {
				month: 'short', day: 'numeric',
				hour: '2-digit', minute: '2-digit',
				timeZone: tz
			});
		return `${fmt(tf.start)} – ${fmt(tf.end)}${editorPrefs.utc ? ' (UTC)' : ''}`;
	});

	function selectPreset(minutes: number) {
		timeFilter = { mode: 'relative', minutes };
		calendarValue = undefined;
		popoverOpen = false;
	}

	function handleCalendarChange(v: DateRange | undefined) {
		if (!v?.start || !v?.end) return;
		const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
		timeFilter = { mode: 'absolute', start: v.start.toDate(tz), end: v.end.toDate(tz) };
		popoverOpen = false;
	}
</script>

<Popover.Root bind:open={popoverOpen}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				class="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
					{popoverOpen ? 'border-ring ring-1 ring-ring' : ''}"
			>
				<CalendarIcon class="size-3.5 text-muted-foreground" />
				<span>{label}</span>
				<ChevronDownIcon class="size-3 text-muted-foreground transition-transform {popoverOpen ? 'rotate-180' : ''}" />
			</button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="w-auto p-0 shadow-lg" align="start">
		<div class="flex">
			<div class="p-4">
				<RangeCalendar
					bind:value={calendarValue}
					onValueChange={handleCalendarChange}
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
