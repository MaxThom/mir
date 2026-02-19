<script lang="ts">
	import {
		createTable,
		getCoreRowModel,
		getFilteredRowModel,
		getPaginationRowModel,
		getSortedRowModel,
		type SortingState,
		type PaginationState,
		type Updater,
		type TableOptionsResolved,
		type TableState
	} from '@tanstack/table-core';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpDownIcon from '@lucide/svelte/icons/arrow-up-down';
	import CircleAlertIcon from '@lucide/svelte/icons/circle-alert';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';
	import * as Table from '$lib/components/ui/table';
	import * as Empty from '$lib/components/ui/empty';
	import * as Card from '$lib/components/ui/card';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { cn } from '$lib/utils';
	import type { Device } from '@mir/sdk';
	import { deviceColumns, deviceGlobalFilterFn } from './device-columns';
	import DeviceTableToolbar from './device-table-toolbar.svelte';
	import DeviceTableSkeleton from './device-table-skeleton.svelte';
	import DeviceTableCell from './device-table-cell.svelte';
	import DeviceTablePagination from './device-table-pagination.svelte';

	let {
		devices,
		isLoading = false,
		error = null,
		onRefresh
	}: {
		devices: Device[];
		isLoading?: boolean;
		error?: string | null;
		onRefresh?: () => void;
	} = $props();

	let sorting = $state<SortingState>([]);
	let globalFilter = $state('');
	let pagination = $state<PaginationState>({ pageIndex: 0, pageSize: 10 });

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
		sorting: []
	};

	let options = $derived<TableOptionsResolved<Device>>({
		data: devices,
		columns: deviceColumns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getSortedRowModel: getSortedRowModel(),
		globalFilterFn: deviceGlobalFilterFn,
		onSortingChange: (updater: Updater<SortingState>) => {
			sorting = typeof updater === 'function' ? updater(sorting) : updater;
		},
		onGlobalFilterChange: (updater: Updater<string>) => {
			globalFilter = typeof updater === 'function' ? updater(globalFilter) : updater;
			pagination = { ...pagination, pageIndex: 0 };
		},
		onPaginationChange: (updater: Updater<PaginationState>) => {
			pagination = typeof updater === 'function' ? updater(pagination) : updater;
		},
		onStateChange() {},
		renderFallbackValue: null,
		state: { ...INITIAL_STATE, sorting, globalFilter, pagination }
	});

	let table = $derived(createTable(options));
	let onlineCount = $derived(
		table.getFilteredRowModel().rows.filter((r) => r.original.status?.online).length
	);
</script>

<Card.Root class="gap-0 overflow-hidden py-0">
	<DeviceTableToolbar
		deviceCount={table.getFilteredRowModel().rows.length}
		{onlineCount}
		{globalFilter}
		{isLoading}
		{onRefresh}
		onglobalfilterchange={(v) => {
			globalFilter = v;
			pagination = { ...pagination, pageIndex: 0 };
		}}
	/>

	{#if isLoading && devices.length === 0}
		<DeviceTableSkeleton />
	{:else if error}
		<Empty.Root class="border-none">
			<Empty.Header>
				<Empty.Media variant="icon">
					<CircleAlertIcon class="text-destructive" />
				</Empty.Media>
				<Empty.Title>Failed to load devices</Empty.Title>
				<Empty.Description>{error}</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	{:else}
		<Tooltip.Provider delayDuration={300}>
			<Table.Root class="min-w-225">
				<Table.Header>
					{#each table.getHeaderGroups() as headerGroup, i (i)}
						<Table.Row class="hover:bg-transparent">
							{#each headerGroup.headers as header, i (i)}
								<Table.Head
									class={cn(
										'h-10 text-xs font-medium tracking-wide text-muted-foreground uppercase',
										header.column.id === 'actions' && 'w-px whitespace-nowrap'
									)}
								>
									{#if !header.isPlaceholder}
										{#if header.column.getCanSort()}
											<button
												onclick={header.column.getToggleSortingHandler()}
												class="flex cursor-pointer items-center gap-1.5 transition-colors hover:text-foreground"
											>
												{header.column.columnDef.header as string}
												{#if header.column.getIsSorted() === 'asc'}
													<ArrowUpIcon class="h-3 w-3" />
												{:else if header.column.getIsSorted() === 'desc'}
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
					{#each table.getRowModel().rows as row, i (i)}
						<Table.Row>
							{#each row.getVisibleCells() as cell, j (j)}
								<DeviceTableCell {cell} {row} />
							{/each}
						</Table.Row>
					{:else}
						<Table.Row class="hover:bg-transparent">
							<Table.Cell colspan={deviceColumns.length} class="p-0">
								<Empty.Root class="border-none">
									<Empty.Header>
										<Empty.Media variant="icon">
											<SatelliteDishIcon />
										</Empty.Media>
										<Empty.Title>No devices found</Empty.Title>
										<Empty.Description>No devices are registered in this context.</Empty.Description>
									</Empty.Header>
								</Empty.Root>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
			<DeviceTablePagination
				pageIndex={pagination.pageIndex}
				pageSize={pagination.pageSize}
				totalRows={table.getFilteredRowModel().rows.length}
				pageCount={table.getPageCount()}
				canPreviousPage={table.getCanPreviousPage()}
				canNextPage={table.getCanNextPage()}
				onpreviouspage={() => table.previousPage()}
				onnextpage={() => table.nextPage()}
				onpagesizechange={(size) => table.setPageSize(size)}
			/>
		</Tooltip.Provider>
	{/if}
</Card.Root>
