<script lang="ts">
	import SearchIcon from '@lucide/svelte/icons/search';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import CalendarIcon from '@lucide/svelte/icons/calendar';
	import XIcon from '@lucide/svelte/icons/x';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu/index.js';
	import * as Popover from '$lib/shared/components/shadcn/popover/index.js';
	import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar/index.js';
	import { DateFormatter, getLocalTimeZone } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';

	export type TypeFilter = 'all' | 'normal' | 'warning';

	type Props = {
		eventCount: number;
		globalFilter: string;
		typeFilter: TypeFilter;
		dateRange: DateRange | undefined;
		allReasons: string[];
		reasonFilter: Set<string>;
		onglobalfilterchange: (value: string) => void;
		ontypefilterchange: (value: TypeFilter) => void;
		ondaterangechange: (value: DateRange | undefined) => void;
		onreasonfiltertoggle: (reason: string) => void;
	};

	let {
		eventCount,
		globalFilter,
		typeFilter,
		dateRange,
		allReasons,
		reasonFilter,
		onglobalfilterchange,
		ontypefilterchange,
		ondaterangechange,
		onreasonfiltertoggle
	}: Props = $props();

	const typeOptions: { value: TypeFilter; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'normal', label: 'Normal' },
		{ value: 'warning', label: 'Warning' }
	];

	const fmt = new DateFormatter('en-US', { month: 'short', day: 'numeric' });

	let dateLabel = $derived.by(() => {
		if (!dateRange?.start) return 'All time';
		const start = fmt.format(dateRange.start.toDate(getLocalTimeZone()));
		if (!dateRange.end) return `${start} – …`;
		const end = fmt.format(dateRange.end.toDate(getLocalTimeZone()));
		return `${start} – ${end}`;
	});

	let hasDateRange = $derived(!!dateRange?.start && !!dateRange?.end);

	let calendarValue = $derived<DateRange | undefined>(undefined);

	// Keep the calendar in sync when the parent resets the range (e.g. X button)
	$effect(() => {
		calendarValue = dateRange;
	});

	function handleCalendarChange(value: DateRange | undefined) {
		if (value?.start && value?.end) {
			ondaterangechange(value);
		}
	}
</script>

<div class="flex items-center justify-between border-b px-6 py-4">
	<div class="flex items-center gap-3">
		<span class="text-sm font-semibold">Events</span>
		<Badge variant="secondary" class="tabular-nums">{eventCount}</Badge>
		<div class="relative">
			<SearchIcon
				class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
			/>
			<Input
				type="search"
				placeholder="Search…"
				class="h-7 w-48 rounded-md pl-8 text-xs transition-[width] focus:w-64"
				value={globalFilter}
				oninput={(e) => onglobalfilterchange((e.target as HTMLInputElement).value)}
			/>
		</div>
	</div>
	<div class="flex items-center gap-3">
		<!-- Reason filter dropdown -->
		{#if allReasons.length > 0}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button variant="outline" size="sm" class="h-7 gap-1.5 px-2.5 text-xs" {...props}>
							Reason
							<ChevronDownIcon class="h-3 w-3 opacity-50" />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content class="max-h-72 overflow-y-auto" align="end">
					{#each allReasons as reason (reason)}
						<DropdownMenu.CheckboxItem
							checked={reasonFilter.has(reason)}
							onCheckedChange={() => onreasonfiltertoggle(reason)}
							closeOnSelect={false}
						>
							<span class="font-mono text-xs">{reason}</span>
						</DropdownMenu.CheckboxItem>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/if}

		<!-- Type filter toggles -->
		<div class="flex items-center rounded-md border">
			{#each typeOptions as opt (opt.value)}
				<Button
					variant="ghost"
					size="sm"
					class="h-7 rounded-none px-2.5 text-xs first:rounded-l-md last:rounded-r-md {typeFilter ===
					opt.value
						? 'bg-muted font-medium text-foreground'
						: 'text-muted-foreground'}"
					onclick={() => ontypefilterchange(opt.value)}
				>
					{opt.label}
				</Button>
			{/each}
		</div>

		<!-- Date range picker -->
		<div class="flex items-center gap-1">
			<Popover.Root>
				<Popover.Trigger>
					{#snippet child({ props })}
						<Button
							variant={hasDateRange ? 'secondary' : 'outline'}
							size="sm"
							class="h-7 gap-1.5 px-2.5 text-xs"
							{...props}
						>
							<CalendarIcon class="h-3 w-3" />
							<span class="text-xs">{dateLabel}</span>
						</Button>
					{/snippet}
				</Popover.Trigger>
				<Popover.Content class="w-auto p-0" align="end">
					<RangeCalendar
						bind:value={calendarValue}
						onValueChange={handleCalendarChange}
						numberOfMonths={2}
					/>
				</Popover.Content>
			</Popover.Root>
			{#if hasDateRange}
				<Button
					variant="ghost"
					size="icon-sm"
					class="h-7 w-7 text-muted-foreground hover:text-foreground"
					onclick={() => ondaterangechange(undefined)}
				>
					<XIcon class="h-3 w-3" />
				</Button>
			{/if}
		</div>
	</div>
</div>
