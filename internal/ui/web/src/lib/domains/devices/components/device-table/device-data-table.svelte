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
	import LayersIcon from '@lucide/svelte/icons/layers';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
	import XIcon from '@lucide/svelte/icons/x';
	import * as Table from '$lib/shared/components/shadcn/table';
	import * as Empty from '$lib/shared/components/shadcn/empty';
	import * as Card from '$lib/shared/components/shadcn/card';
	import * as Tooltip from '$lib/shared/components/shadcn/tooltip';
	import * as Checkbox from 'bits-ui';
	import { goto } from '$app/navigation';
	import { cn } from '$lib/utils';
	import type { Device } from '@mir/sdk';
	import { deviceColumns, deviceGlobalFilterFn } from './device-columns';
	import DeviceTableToolbar from './device-table-toolbar.svelte';
	import DeviceTableSkeleton from './device-table-skeleton.svelte';
	import DeviceTableCell from './device-table-cell.svelte';
	import DeviceTablePagination from './device-table-pagination.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';

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
	let rowSelection = $state<Record<string, boolean>>({});

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
		enableRowSelection: true,
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
		onRowSelectionChange: (updater: Updater<Record<string, boolean>>) => {
			rowSelection = typeof updater === 'function' ? updater(rowSelection) : updater;
		},
		onStateChange() {},
		renderFallbackValue: null,
		state: { ...INITIAL_STATE, sorting, globalFilter, pagination, rowSelection }
	});

	let table = $derived(createTable(options));
	let onlineCount = $derived(
		table.getFilteredRowModel().rows.filter((r) => r.original.status?.online).length
	);

	$effect(() => {
		const selected = table.getSelectedRowModel().rows.map((r) => r.original);
		selectionStore.setAll(selected);
	});
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
								{#if header.column.id === 'select'}
									<Table.Head class="w-px px-3">
										<Checkbox.Root
											checked={table.getIsAllPageRowsSelected()}
											indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
											onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
											aria-label="Select all"
											class="flex size-4 items-center justify-center rounded border border-input bg-background transition-colors data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground data-[state=indeterminate]:bg-primary data-[state=indeterminate]:text-primary-foreground"
										>
											{#snippet children({ checked, indeterminate })}
												{#if indeterminate}
													<span class="size-2 rounded-sm bg-current"></span>
												{:else if checked}
													<svg class="size-3" viewBox="0 0 12 12" fill="none">
														<path d="M2 6l3 3 5-5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
													</svg>
												{/if}
											{/snippet}
										</Checkbox.Root>
									</Table.Head>
								{:else}
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
													class="flex cursor-pointer items-center gap-1.5 uppercase transition-colors hover:text-foreground"
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
								{/if}
							{/each}
						</Table.Row>
					{/each}
				</Table.Header>
				<Table.Body>
					{#each table.getRowModel().rows as row, i (i)}
						<Table.Row>
							{#each row.getVisibleCells() as cell, j (j)}
								{#if cell.column.id === 'select'}
									<Table.Cell class="w-px px-3">
										<Checkbox.Root
											checked={row.getIsSelected()}
											onCheckedChange={(v) => row.toggleSelected(!!v)}
											aria-label="Select row"
											class="flex size-4 items-center justify-center rounded border border-input bg-background transition-colors data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground"
										>
											{#snippet children({ checked })}
												{#if checked}
													<svg class="size-3" viewBox="0 0 12 12" fill="none">
														<path d="M2 6l3 3 5-5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
													</svg>
												{/if}
											{/snippet}
										</Checkbox.Root>
									</Table.Cell>
								{:else}
									<DeviceTableCell {cell} {row} />
								{/if}
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
			{#if selectionStore.count > 0}
				<div class="flex items-center justify-between border-t bg-muted/50 px-4 py-2">
					<div class="flex items-center gap-2 text-sm text-muted-foreground">
						<LayersIcon class="size-4" />
						<span>{selectionStore.count} device{selectionStore.count > 1 ? 's' : ''} selected</span>
					</div>
					<div class="flex items-center gap-2">
						<button
							onclick={() => { selectionStore.clearSelection(); rowSelection = {}; }}
							class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
						>
							<XIcon class="size-3.5" />
							Clear
						</button>
						<button
							onclick={() => goto('/devices/multi/telemetry')}
							class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
						>
							<ActivityIcon class="size-3.5" />
							Telemetry
						</button>
						<button
							onclick={() => goto('/devices/multi/commands')}
							class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
						>
							<TerminalIcon class="size-3.5" />
							Commands
						</button>
						<button
							onclick={() => goto('/devices/multi/configuration')}
							class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
						>
							<SlidersHorizontalIcon class="size-3.5" />
							Configuration
						</button>
					</div>
				</div>
			{/if}
		</Tooltip.Provider>
	{/if}
</Card.Root>
