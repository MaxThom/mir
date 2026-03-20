<script lang="ts">
	import type { Device } from '@mir/sdk';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Checkbox } from '$lib/shared/components/shadcn/checkbox';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import * as Table from '$lib/shared/components/shadcn/table';
	import { cn } from '$lib/utils';
	import SearchIcon from '@lucide/svelte/icons/search';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';
	import XIcon from '@lucide/svelte/icons/x';
	import LayersIcon from '@lucide/svelte/icons/layers';

	let {
		devices,
		selectedIds = $bindable<string[]>([]),
		isLoading = false
	}: {
		devices: Device[];
		selectedIds: string[];
		isLoading?: boolean;
	} = $props();

	let search = $state('');

	let filtered = $derived(() => {
		const q = search.toLowerCase().trim();
		if (!q) return devices;
		return devices.filter((d) => {
			const name = (d.meta?.name ?? '').toLowerCase();
			const ns = (d.meta?.namespace ?? '').toLowerCase();
			const id = (d.spec?.deviceId ?? '').toLowerCase();
			const labels = Object.entries(d.meta?.labels ?? {}).map(([k, v]) => `${k}=${v}`).join(' ').toLowerCase();
			const pkgs = (d.status?.schema?.packageNames ?? [])
				.filter((p) => p !== 'mir.device.v1' && p !== 'google.protobuf')
				.join(' ').toLowerCase();
			return name.includes(q) || ns.includes(q) || id.includes(q) || labels.includes(q) || pkgs.includes(q);
		});
	});

	let onlineCount = $derived(devices.filter((d) => d.status?.online).length);

	function toggle(id: string) {
		if (selectedIds.includes(id)) {
			selectedIds = selectedIds.filter((x) => x !== id);
		} else {
			selectedIds = [...selectedIds, id];
		}
	}

	function toggleAll() {
		const ids = filtered().map((d) => d.spec?.deviceId ?? '').filter(Boolean);
		const allSelected = ids.every((id) => selectedIds.includes(id));
		if (allSelected) {
			selectedIds = selectedIds.filter((id) => !ids.includes(id));
		} else {
			selectedIds = [...new Set([...selectedIds, ...ids])];
		}
	}

	let allFilteredSelected = $derived(
		filtered().length > 0 &&
			filtered().map((d) => d.spec?.deviceId ?? '').filter(Boolean).every((id) => selectedIds.includes(id))
	);

	let someFilteredSelected = $derived(
		!allFilteredSelected &&
			filtered().map((d) => d.spec?.deviceId ?? '').filter(Boolean).some((id) => selectedIds.includes(id))
	);
</script>

<div class="border-border rounded-md border">
	<!-- Toolbar -->
	<div class="flex items-center justify-between border-b px-4 py-2">
		<div class="flex items-center gap-3">
			<span class="text-sm font-semibold">Devices</span>
			<Badge variant="secondary" class="tabular-nums">{filtered().length}</Badge>
			<div class="relative">
				<SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
				<Input
					type="search"
					placeholder="Search…"
					class="h-7 w-40 rounded-md pl-8 text-xs transition-[width] focus:w-56"
					bind:value={search}
				/>
			</div>
		</div>
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
			{onlineCount} online
		</div>
	</div>

	<!-- Table -->
	{#if isLoading}
		<div class="flex items-center justify-center py-10">
			<Spinner class="h-4 w-4" />
		</div>
	{:else}
		<div class="max-h-[50vh] overflow-y-auto">
		<Table.Root>
			<Table.Header>
				<Table.Row class="hover:bg-transparent">
					<Table.Head class="w-px px-3">
						<Checkbox
							checked={allFilteredSelected}
							indeterminate={someFilteredSelected}
							onCheckedChange={toggleAll}
							aria-label="Select all"
						/>
					</Table.Head>
					<Table.Head class="w-5"></Table.Head>
					<Table.Head class="h-8 text-xs font-medium tracking-wide uppercase text-muted-foreground">Name</Table.Head>
					<Table.Head class="h-8 text-xs font-medium tracking-wide uppercase text-muted-foreground">Namespace</Table.Head>
					<Table.Head class="h-8 text-xs font-medium tracking-wide uppercase text-muted-foreground">Device ID</Table.Head>
					<Table.Head class="h-8 text-xs font-medium tracking-wide uppercase text-muted-foreground">Labels</Table.Head>
					<Table.Head class="h-8 text-xs font-medium tracking-wide uppercase text-muted-foreground">Schema</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#if filtered().length === 0}
					<Table.Row class="hover:bg-transparent">
						<Table.Cell colspan={7} class="py-8 text-center">
							<div class="flex flex-col items-center gap-2 text-muted-foreground">
								<SatelliteDishIcon class="h-6 w-6 opacity-40" />
								<span class="text-sm">{search ? 'No devices match your search.' : 'No devices found.'}</span>
							</div>
						</Table.Cell>
					</Table.Row>
				{:else}
					{#each filtered() as device (device.spec?.deviceId ?? device.meta?.name)}
						{@const id = device.spec?.deviceId ?? ''}
						{@const checked = selectedIds.includes(id)}
						{@const pkgs = (device.status?.schema?.packageNames ?? []).filter(
							(p) => p !== 'mir.device.v1' && p !== 'google.protobuf'
						)}
						<Table.Row
							class={cn('cursor-pointer', checked && 'bg-muted/50')}
							onclick={() => toggle(id)}
						>
							<Table.Cell class="w-px px-3" onclick={(e) => e.stopPropagation()}>
								<Checkbox {checked} onCheckedChange={() => toggle(id)} aria-label="Select device" />
							</Table.Cell>
							<Table.Cell class="w-5 pr-0">
								<span class={cn(
									'block h-2 w-2 rounded-full',
									device.status?.online
										? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
										: 'bg-muted-foreground/30'
								)}></span>
							</Table.Cell>
							<Table.Cell class="font-medium text-sm">{device.meta?.name ?? '—'}</Table.Cell>
							<Table.Cell>
								<Badge variant="outline" class="font-mono text-xs font-normal">
									{device.meta?.namespace ?? 'default'}
								</Badge>
							</Table.Cell>
							<Table.Cell class="font-mono text-xs text-muted-foreground">{id || '—'}</Table.Cell>
							<Table.Cell>
								<div class="flex flex-wrap gap-1">
									{#each Object.entries(device.meta?.labels ?? {}) as [k, v] (k)}
										<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/each}
								</div>
							</Table.Cell>
							<Table.Cell>
								<div class="flex flex-wrap gap-1">
									{#each pkgs as pkg (pkg)}
										<Badge variant="outline" class="font-mono text-xs font-normal">{pkg}</Badge>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/each}
								</div>
							</Table.Cell>
						</Table.Row>
					{/each}
				{/if}
			</Table.Body>
		</Table.Root>
		</div>
	{/if}

	<!-- Selection footer -->
	{#if selectedIds.length > 0}
		<div class="flex items-center justify-between border-t bg-muted/50 px-4 py-2">
			<div class="flex items-center gap-2 text-sm text-muted-foreground">
				<LayersIcon class="size-4" />
				<span>{selectedIds.length} device{selectedIds.length > 1 ? 's' : ''} selected</span>
			</div>
			<button
				onclick={() => (selectedIds = [])}
				class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
			>
				<XIcon class="size-3.5" />
				Clear
			</button>
		</div>
	{/if}
</div>
