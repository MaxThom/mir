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
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import type { Row } from '@tanstack/table-core';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';
	import CircleAlertIcon from '@lucide/svelte/icons/circle-alert';
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import ArrowDownIcon from '@lucide/svelte/icons/arrow-down';
	import ArrowUpDownIcon from '@lucide/svelte/icons/arrow-up-down';
	import SearchIcon from '@lucide/svelte/icons/search';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import ListIcon from '@lucide/svelte/icons/list';
	import BracesIcon from '@lucide/svelte/icons/braces';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Input } from '$lib/components/ui/input';
	import * as Table from '$lib/components/ui/table';
	import * as Empty from '$lib/components/ui/empty';
	import * as Card from '$lib/components/ui/card';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import type { Device } from '@mir/sdk';
	import { deviceColumns } from './device-columns';

	let {
		devices,
		isLoading = false,
		error = null
	}: { devices: Device[]; isLoading?: boolean; error?: string | null } = $props();

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

	// Default global filter function for devices
	function deviceGlobalFilterFn(row: Row<Device>, columnId: string, value: string): boolean {
		const searchValue = value.toLowerCase();
		const device = row.original as Device;

		// Search in device name, namespace, deviceId, and labels
		const searchableText = [
			device.meta?.name,
			device.meta?.namespace,
			device.spec?.deviceId,
			...Object.entries(device.meta?.labels || {}).map(([k, v]) => `${k}=${v}`)
		]
			.filter(Boolean)
			.join(' ')
			.toLowerCase();

		return searchableText.includes(searchValue);
	}

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

	let onlineCount = $derived(devices.filter((d) => d.status?.online).length);

	function relativeTime(seconds: bigint | number): string {
		const diff = Date.now() - Number(seconds) * 1000;
		const minutes = Math.floor(diff / 60000);
		if (minutes < 1) return 'just now';
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		return `${Math.floor(hours / 24)}d ago`;
	}

	function formatFullDate(seconds: bigint | number): string {
		return new Date(Number(seconds) * 1000).toLocaleString();
	}
</script>

<Card.Root class="gap-0 overflow-hidden py-0">
	<div class="flex items-center justify-between border-b px-6 py-4">
		<div class="flex items-center gap-3">
			<span class="text-sm font-semibold">Devices</span>
			<Badge variant="secondary" class="tabular-nums">{devices.length}</Badge>
			<div class="relative">
				<SearchIcon
					class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
				/>
				<Input
					type="search"
					placeholder="Search…"
					class="h-7 w-48 rounded-md pl-8 text-xs transition-[width] focus:w-64"
					value={globalFilter}
					oninput={(e) => {
						globalFilter = (e.target as HTMLInputElement).value;
						pagination = { ...pagination, pageIndex: 0 };
					}}
				/>
			</div>
		</div>
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
			{onlineCount} online
		</div>
	</div>

	{#if isLoading}
		<Table.Root class="min-w-225">
			<Table.Header>
				<Table.Row class="hover:bg-transparent">
					{#each deviceColumns as col, i (i)}
						<Table.Head
							class="h-10 text-xs font-medium tracking-wide text-muted-foreground uppercase"
						>
							{col.header as string}
						</Table.Head>
					{/each}
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each { length: 5 }, i (i)}
					<Table.Row class="hover:bg-transparent">
						{#each deviceColumns, j (j)}
							<Table.Cell><Skeleton class="h-4 w-full" /></Table.Cell>
						{/each}
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
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
								<Table.Cell
									class={cell.column.id === 'actions' ? 'w-px pr-2 whitespace-nowrap' : ''}
								>
									{#if cell.column.id === 'name'}
										<span class="font-medium">{cell.getValue() ?? '—'}</span>
									{:else if cell.column.id === 'namespace'}
										<Badge variant="outline" class="font-mono text-xs font-normal">
											{cell.getValue() ?? '—'}
										</Badge>
									{:else if cell.column.id === 'deviceId'}
										<span class="font-mono text-xs text-muted-foreground">
											{cell.getValue() ?? '—'}
										</span>
									{:else if cell.column.id === 'labels'}
										<div class="flex flex-col gap-1">
											{#each Object.entries(cell.getValue() as Record<string, string>) as [k, v] (k)}
												<Badge variant="secondary" class="font-mono text-xs font-normal"
													>{k}={v}</Badge
												>
											{/each}
										</div>
									{:else if cell.column.id === 'status'}
										<div class="flex items-center gap-2">
											<span
												class={cn(
													'h-2 w-2 shrink-0 rounded-full',
													cell.getValue()
														? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
														: 'bg-muted-foreground/30'
												)}
											></span>
											<span
												class={cn(
													'text-sm',
													cell.getValue()
														? 'font-medium text-emerald-600 dark:text-emerald-400'
														: 'text-muted-foreground'
												)}
											>
												{cell.getValue() ? 'Online' : 'Offline'}
											</span>
											{#if row.original.spec?.disabled}
												<Badge variant="destructive" class="text-xs">Disabled</Badge>
											{/if}
										</div>
									{:else if cell.column.id === 'lastHeartbeat'}
										{#if cell.getValue()}
											<Tooltip.Root>
												<Tooltip.Trigger
													class="cursor-default text-sm text-muted-foreground underline decoration-dotted underline-offset-2 hover:text-foreground"
												>
													{relativeTime((cell.getValue() as { seconds: bigint | number }).seconds)}
												</Tooltip.Trigger>
												<Tooltip.Content side="left">
													{formatFullDate(
														(cell.getValue() as { seconds: bigint | number }).seconds
													)}
												</Tooltip.Content>
											</Tooltip.Root>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/if}
									{:else if cell.column.id === 'actions'}
										{@const deviceId = row.original.spec?.deviceId ?? ''}
										<div class="flex items-center gap-0.5">
											{#each [{ icon: ActivityIcon, label: 'Telemetry', path: 'telemetry' }, { icon: TerminalIcon, label: 'Commands', path: 'commands' }, { icon: SettingsIcon, label: 'Configuration', path: 'configuration' }, { icon: ListIcon, label: 'Events', path: 'events' }, { icon: BracesIcon, label: 'Schema', path: 'schema' }] as action (action.path)}
												<Tooltip.Root>
													<Tooltip.Trigger>
														<Button
															variant="ghost"
															size="icon-sm"
															disabled={!deviceId}
															href="/devices/{deviceId}/{action.path}"
														>
															<action.icon class="h-3.5 w-3.5" />
														</Button>
													</Tooltip.Trigger>
													<Tooltip.Content>{action.label}</Tooltip.Content>
												</Tooltip.Root>
											{/each}
										</div>
									{:else}
										{cell.getValue() ?? '—'}
									{/if}
								</Table.Cell>
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
										<Empty.Description>No devices are registered in this context.</Empty.Description
										>
									</Empty.Header>
								</Empty.Root>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Tooltip.Provider>
		<div class="flex items-center justify-between border-t px-6 py-3 text-sm text-muted-foreground">
			<span class="tabular-nums">
				{pagination.pageIndex * pagination.pageSize + 1}–{Math.min(
					(pagination.pageIndex + 1) * pagination.pageSize,
					table.getFilteredRowModel().rows.length
				)} of {table.getFilteredRowModel().rows.length}
			</span>
			<div class="flex items-center gap-2">
				<Button
					variant="outline"
					size="icon-sm"
					disabled={!table.getCanPreviousPage()}
					onclick={() => table.previousPage()}
				>
					<ChevronLeftIcon class="h-3.5 w-3.5" />
				</Button>
				<span class="tabular-nums">
					{pagination.pageIndex + 1} / {table.getPageCount()}
				</span>
				<Button
					variant="outline"
					size="icon-sm"
					disabled={!table.getCanNextPage()}
					onclick={() => table.nextPage()}
				>
					<ChevronRightIcon class="h-3.5 w-3.5" />
				</Button>
			</div>
			<div class="flex items-center gap-2">
				<span>Rows</span>
				<select
					class="h-7 rounded-md border bg-background px-2 text-xs"
					value={pagination.pageSize}
					onchange={(e) => table.setPageSize(Number((e.target as HTMLSelectElement).value))}
				>
					{#each [5, 10, 20, 50] as size, i (i)}
						<option value={size}>{size}</option>
					{/each}
				</select>
			</div>
		</div>
	{/if}
</Card.Root>
