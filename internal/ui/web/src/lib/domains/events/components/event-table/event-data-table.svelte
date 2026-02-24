<script lang="ts">
	import { untrack } from 'svelte';
	import {
		createTable,
		getCoreRowModel,
		getFilteredRowModel,
		getPaginationRowModel,
		getSortedRowModel,
		getExpandedRowModel,
		type SortingState,
		type PaginationState,
		type ExpandedState,
		type Updater,
		type TableState,
		type Row
	} from '@tanstack/table-core';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpDownIcon from '@lucide/svelte/icons/arrow-up-down';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import CircleAlertIcon from '@lucide/svelte/icons/circle-alert';
	import CalendarClockIcon from '@lucide/svelte/icons/calendar-clock';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';
	import * as Table from '$lib/shared/components/shadcn/table';
	import * as Empty from '$lib/shared/components/shadcn/empty';
	import * as Card from '$lib/shared/components/shadcn/card';
	import * as Tooltip from '$lib/shared/components/shadcn/tooltip';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { cn } from '$lib/utils';
	import type { MirEvent } from '@mir/sdk';
	import { SvelteSet } from 'svelte/reactivity';
	import { eventColumns, eventGlobalFilterFn } from './event-columns';
	import EventTableToolbar from './event-table-toolbar.svelte';
	import type { TypeFilter } from './event-table-toolbar.svelte';
	import { getLocalTimeZone } from '@internationalized/date';
	import type { DateRange } from 'bits-ui';
	import DeviceTablePagination from '$lib/domains/devices/components/device-table/device-table-pagination.svelte';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { getHighlighter } from '$lib/shared/utils/highlighter';

	let {
		events,
		isLoading = false,
		hasLoaded = false,
		error = null,
		onrefetch
	}: {
		events: MirEvent[];
		isLoading?: boolean;
		hasLoaded?: boolean;
		error?: string | null;
		onrefetch?: (from?: Date, to?: Date) => void;
	} = $props();

	let sorting = $state<SortingState>([{ id: 'lastAt', desc: true }]);
	let globalFilter = $state('');
	let pagination = $state<PaginationState>({ pageIndex: 0, pageSize: 10 });
	let expanded = $state<ExpandedState>({});
	let typeFilter = $state<TypeFilter>('all');
	let dateRange = $state<DateRange | undefined>(undefined);
	let reasonFilter = new SvelteSet<string>();

	// Payload highlight state
	let highlightedPayloads = $state<Record<string, string>>({});
	let copiedKey = $state<string | null>(null);

	let allReasons = $derived(
		[...new Set(events.map((e) => e.spec?.reason ?? '').filter(Boolean))].sort()
	);

	let filteredEvents = $derived.by(() => {
		let evts = events;
		if (typeFilter !== 'all') evts = evts.filter((e) => e.spec?.type === typeFilter);
		if (reasonFilter.size > 0) evts = evts.filter((e) => reasonFilter.has(e.spec?.reason ?? ''));
		return evts;
	});

	const INITIAL_STATE: TableState = {
		columnPinning: { left: [], right: [] },
		columnFilters: [],
		columnOrder: [],
		columnSizing: {},
		columnSizingInfo: {
			startOffset: null,
			startSize: null,
			deltaOffset: null,
			deltaPercentage: null,
			isResizingColumn: false,
			columnSizingStart: []
		},
		columnVisibility: {},
		expanded: {},
		globalFilter: undefined,
		grouping: [],
		pagination: { pageIndex: 0, pageSize: 10 },
		rowPinning: { top: [], bottom: [] },
		rowSelection: {},
		sorting: [{ id: 'lastAt', desc: true }]
	};

	// Stable event handlers — defined once, not re-created on each update
	const handleSortingChange = (updater: Updater<SortingState>) => {
		sorting = typeof updater === 'function' ? updater(sorting) : updater;
	};
	const handleGlobalFilterChange = (updater: Updater<string>) => {
		globalFilter = typeof updater === 'function' ? updater(globalFilter) : updater;
		pagination = { ...pagination, pageIndex: 0 };
	};
	const handlePaginationChange = (updater: Updater<PaginationState>) => {
		pagination = typeof updater === 'function' ? updater(pagination) : updater;
	};
	const handleExpandedChange = (updater: Updater<ExpandedState>) => {
		expanded = typeof updater === 'function' ? updater(expanded) : updater;
	};

	// Create the table ONCE — never recreated on data/state changes
	const table = createTable({
		data: [] as MirEvent[],
		columns: eventColumns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getExpandedRowModel: getExpandedRowModel(),
		globalFilterFn: eventGlobalFilterFn,
		onSortingChange: handleSortingChange,
		onGlobalFilterChange: handleGlobalFilterChange,
		onPaginationChange: handlePaginationChange,
		onExpandedChange: handleExpandedChange,
		onStateChange() {},
		renderFallbackValue: null,
		state: { ...INITIAL_STATE }
	});

	// Version counter: incremented whenever the table is updated, triggering targeted re-reads
	let tableVersion = $state(0);

	// Sync reactive Svelte state → table options without recreating the table instance
	$effect(() => {
		const data = filteredEvents;
		const currentSorting = sorting;
		const currentGlobalFilter = globalFilter;
		const currentPagination = pagination;
		const currentExpanded = expanded;
		untrack(() => {
			table.setOptions((prev) => ({
				...prev,
				data,
				state: {
					...prev.state,
					sorting: currentSorting,
					globalFilter: currentGlobalFilter,
					pagination: currentPagination,
					expanded: currentExpanded
				}
			}));
			tableVersion++;
		});
	});

	// Re-read from the stable table instance whenever it's been updated
	let headerGroups = $derived.by(() => {
		void tableVersion;
		return table.getHeaderGroups();
	});
	let rows = $derived.by(() => {
		void tableVersion;
		return table.getRowModel().rows;
	});
	let filteredRowCount = $derived.by(() => {
		void tableVersion;
		return table.getFilteredRowModel().rows.length;
	});
	let pageCount = $derived.by(() => {
		void tableVersion;
		return table.getPageCount();
	});
	let canPreviousPage = $derived.by(() => {
		void tableVersion;
		return table.getCanPreviousPage();
	});
	let canNextPage = $derived.by(() => {
		void tableVersion;
		return table.getCanNextPage();
	});

	function formatPayload(payload: unknown): string {
		if (payload === undefined || payload === null) return '';
		try {
			return JSON.stringify(payload, null, 2);
		} catch {
			return String(payload);
		}
	}

	function rowKey(rowId: string): string {
		return rowId;
	}

	async function highlightPayload(key: string, payload: string) {
		if (highlightedPayloads[key] !== undefined) return;
		const hl = await getHighlighter();
		highlightedPayloads[key] = hl.codeToHtml(payload, {
			lang: 'json',
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
	}

	function copyPayload(key: string, payload: string) {
		navigator.clipboard.writeText(payload).then(() => {
			copiedKey = key;
			setTimeout(() => (copiedKey = null), 2000);
		});
	}

	function toggleRow(row: Row<MirEvent>) {
		if (!row.getIsExpanded()) {
			const payload = formatPayload(row.original.spec?.payload);
			if (payload) highlightPayload(rowKey(row.id), payload);
		}
		row.toggleExpanded();
	}
</script>

<Card.Root class="gap-0 overflow-hidden py-0">
	<EventTableToolbar
		eventCount={filteredRowCount}
		{globalFilter}
		{typeFilter}
		{dateRange}
		{allReasons}
		{reasonFilter}
		onglobalfilterchange={(v) => {
			globalFilter = v;
			pagination = { ...pagination, pageIndex: 0 };
		}}
		ontypefilterchange={(v) => {
			typeFilter = v;
			pagination = { ...pagination, pageIndex: 0 };
		}}
		ondaterangechange={(v) => {
			dateRange = v;
			pagination = { ...pagination, pageIndex: 0 };
			if (v?.start && v?.end) {
				const tz = getLocalTimeZone();
				const from = v.start.toDate(tz);
				const to = v.end.toDate(tz);
				to.setHours(23, 59, 59, 999);
				onrefetch?.(from, to);
			} else {
				onrefetch?.();
			}
		}}
		onreasonfiltertoggle={(reason) => {
			if (reasonFilter.has(reason)) reasonFilter.delete(reason);
			else reasonFilter.add(reason);
			pagination = { ...pagination, pageIndex: 0 };
		}}
	/>

	{#if error}
		<Empty.Root class="border-none">
			<Empty.Header>
				<Empty.Media variant="icon">
					<CircleAlertIcon class="text-destructive" />
				</Empty.Media>
				<Empty.Title>Failed to load events</Empty.Title>
				<Empty.Description>{error}</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<Tooltip.Provider delayDuration={300}>
			<Table.Root class="min-w-200">
				<Table.Header>
					{#each headerGroups as headerGroup, i (i)}
						<Table.Row class="hover:bg-transparent">
							{#each headerGroup.headers as header, j (j)}
								<Table.Head
									class={cn(
										'h-10 text-xs font-medium tracking-wide text-muted-foreground uppercase',
										header.column.id === 'expand' && 'w-8',
										header.column.id === 'type' && 'w-24',
										header.column.id === 'lastAt' && 'w-28'
									)}
								>
									{#if !header.isPlaceholder && header.column.id !== 'expand'}
										{#if header.column.getCanSort()}
											{@const sortEntry = sorting.find((s) => s.id === header.column.id)}
											<button
												onclick={header.column.getToggleSortingHandler()}
												class="flex cursor-pointer items-center gap-1.5 uppercase transition-colors hover:text-foreground"
											>
												{header.column.columnDef.header as string}
												{#if sortEntry && !sortEntry.desc}
													<ArrowUpIcon class="h-3 w-3" />
												{:else if sortEntry?.desc}
													<ArrowDownIcon class="h-3 w-3" />
												{:else}
													<ArrowUpDownIcon class="h-3 w-3 opacity-40" />
												{/if}
											</button>
										{:else}
											{header.column.columnDef.header as string}
										{/if}
									{/if}
								</Table.Head>
							{/each}
						</Table.Row>
					{/each}
				</Table.Header>
				<Table.Body>
					{#if isLoading && !hasLoaded}
						{#each Array(5), i (i)}
							<Table.Row class="hover:bg-transparent">
								{#each eventColumns, j (j)}
									<Table.Cell>
										<div class="h-4 animate-pulse rounded bg-muted"></div>
									</Table.Cell>
								{/each}
							</Table.Row>
						{/each}
					{:else}
						{#each rows as row, i (i)}
							{@const event = row.original}
							{@const isExpanded = expanded === true || !!(expanded as Record<string, boolean>)?.[row.id]}
							{@const payload = formatPayload(event.spec?.payload)}
							{@const key = rowKey(row.id)}

							<Table.Row class="cursor-pointer" onclick={() => toggleRow(row)}>
								<!-- Expand chevron -->
								<Table.Cell class="w-8 pr-0">
									<ChevronRightIcon
										class={cn(
											'h-3.5 w-3.5 text-muted-foreground transition-transform',
											isExpanded && 'rotate-90'
										)}
									/>
								</Table.Cell>

								<!-- Type badge -->
								<Table.Cell>
									<Badge
										variant={event.spec?.type === 'warning' ? 'destructive' : 'secondary'}
										class="font-mono text-[10px]"
									>
										{event.spec?.type ?? 'normal'}
									</Badge>
								</Table.Cell>

								<!-- Reason -->
								<Table.Cell class="font-mono text-xs">
									{event.spec?.reason || '—'}
								</Table.Cell>

								<!-- Message -->
								<Table.Cell class="max-w-xs truncate text-xs text-muted-foreground">
									{event.spec?.message || '—'}
								</Table.Cell>

								<!-- Last seen -->
								<Table.Cell
									class="text-xs"
									onclick={(e) => e.stopPropagation()}
									onkeydown={(e) => e.stopPropagation()}
									role="cell"
								>
									{#if event.status?.lastAt}
										<TimeTooltip
											timestamp={event.status.lastAt}
											utc={editorPrefs.utc}
											class="text-xs text-muted-foreground"
										/>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/if}
								</Table.Cell>
							</Table.Row>

							{#if isExpanded}
								<Table.Row class="hover:bg-transparent">
									<Table.Cell colspan={eventColumns.length} class="px-8 py-3">
										{#if payload}
											<div
												class="group relative overflow-hidden rounded border border-border text-[10px] leading-relaxed [&>pre]:px-3 [&>pre]:py-2 [&>pre]:break-all [&>pre]:whitespace-pre-wrap"
											>
												<button
													onclick={(e) => {
														e.stopPropagation();
														copyPayload(key, payload);
													}}
													aria-label="Copy payload"
													class="absolute top-1.5 right-1.5 z-10 rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-foreground"
												>
													{#if copiedKey === key}
														<CheckIcon class="size-3 text-emerald-500" />
													{:else}
														<CopyIcon class="size-3" />
													{/if}
												</button>
												{#if highlightedPayloads[key]}
													<!-- eslint-disable-next-line svelte/no-at-html-tags -->
													{@html highlightedPayloads[key]}
												{:else}
													<pre
														class="bg-muted px-3 py-2 font-mono text-[10px] break-all whitespace-pre-wrap">{payload}</pre>
												{/if}
											</div>
										{:else}
											<p class="text-xs text-muted-foreground">No payload.</p>
										{/if}
									</Table.Cell>
								</Table.Row>
							{/if}
						{:else}
							<Table.Row class="hover:bg-transparent">
								<Table.Cell colspan={eventColumns.length} class="p-0">
									<Empty.Root class="border-none">
										<Empty.Header>
											<Empty.Media variant="icon">
												<CalendarClockIcon />
											</Empty.Media>
											<Empty.Title>No events found</Empty.Title>
											<Empty.Description>No events match the current filters.</Empty.Description>
										</Empty.Header>
									</Empty.Root>
								</Table.Cell>
							</Table.Row>
						{/each}
					{/if}
				</Table.Body>
			</Table.Root>
			<DeviceTablePagination
				pageIndex={pagination.pageIndex}
				pageSize={pagination.pageSize}
				totalRows={filteredRowCount}
				{pageCount}
				{canPreviousPage}
				{canNextPage}
				onpreviouspage={() => table.previousPage()}
				onnextpage={() => table.nextPage()}
				onpagesizechange={(size) => table.setPageSize(size)}
			/>
		</Tooltip.Provider>
	{/if}
</Card.Root>
