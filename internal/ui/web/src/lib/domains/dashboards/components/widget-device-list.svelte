<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { Device } from '@mir/sdk';
	import type { DeviceListWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';

	let {
		config,
		refreshTick = 0,
		onDevicesReady
	}: {
		config: DeviceListWidgetConfig;
		refreshTick?: number;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	let devices = $state<Device[]>([]);
	let isLoading = $state(false);
	let hasLoaded = $state(false);
	let error = $state<string | null>(null);
	let isInRefresh = false;
	let expandedId = $state<string | null>(null);
	let containerWidth = $state(0);
	const narrow = $derived(containerWidth < 280);

	const sorted = $derived(
		[...devices].sort((a, b) => {
			const aOn = a.status?.online ? 0 : 1;
			const bOn = b.status?.online ? 0 : 1;
			if (aOn !== bOn) return aOn - bOn;
			const nsCmp = (a.meta?.namespace ?? '').localeCompare(b.meta?.namespace ?? '');
			if (nsCmp !== 0) return nsCmp;
			return (a.meta?.name ?? '').localeCompare(b.meta?.name ?? '');
		})
	);

	async function loadDevices() {
		const mir = mirStore.mir;
		if (!mir) return;
		isLoading = true;
		error = null;
		try {
			const target = new DeviceTarget({
				names: config.target.names ?? [],
				namespaces: config.target.namespaces ?? [],
				labels: config.target.labels ?? {}
			});
			devices = await mir.client().listDevices().request(target, false);
			onDevicesReady?.(
				devices.map((d, i) => ({
					id: d.spec?.deviceId ?? d.meta?.name ?? '',
					name: d.meta?.name ?? '',
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load devices';
		} finally {
			isLoading = false;
			hasLoaded = true;
			if (isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
		}
	}

	$effect(() => {
		if (refreshTick > 0) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			loadDevices();
		}
	});

	$effect(() => {
		if (mirStore.mir) {
			devices = [];
			hasLoaded = false;
			loadDevices();
		} else {
			devices = [];
			hasLoaded = false;
		}
	});
</script>

<div class="flex h-full flex-col" bind:clientWidth={containerWidth}>
	<div class="mt-2.5 shrink-0 border-b"></div>

	{#if isLoading && !hasLoaded}
		<div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
			Loading…
		</div>
	{:else if error}
		<div class="flex flex-1 items-center justify-center text-sm text-destructive">{error}</div>
	{:else if sorted.length === 0}
		<div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
			No devices
		</div>
	{:else}
		<div class="min-h-0 flex-1 overflow-y-auto">
			<table class="w-full table-fixed text-xs">
				<colgroup>
					<col class="w-7" />
					<col class="w-4" />
					<col />
					{#if narrow}
						<col class="w-[40%]" />
					{:else}
						<col class="w-[25%]" />
						<col class="w-20" />
					{/if}
				</colgroup>
				<thead class="sticky top-0 bg-card">
					<tr class="border-b text-left text-muted-foreground">
						<th></th>
						<th></th>
						<th class="px-2 py-1.5 font-medium">Name</th>
						<th class="px-2 py-1.5 font-medium">Namespace</th>
						{#if !narrow}<th class="px-2 py-1.5 font-medium">Last seen</th>{/if}
					</tr>
				</thead>
				<tbody>
					{#each sorted as device (device.spec?.deviceId ?? device.meta?.name)}
						{@const rowId = device.spec?.deviceId ?? device.meta?.name ?? ''}
						{@const isExpanded = expandedId === rowId}
						{@const online = device.status?.online ?? false}
						{@const labels = device.meta?.labels ?? {}}
						{@const packages = (device.status?.schema?.packageNames ?? []).filter(
							(p) => p !== 'google.protobuf' && p !== 'mir.device.v1'
						)}
						<tr
							class="cursor-pointer border-b border-border/50 hover:bg-muted/40"
							onclick={() => (expandedId = isExpanded ? null : rowId)}
						>
							<td class="py-1.5 pl-2">
								<ChevronRightIcon
									class="h-3.5 w-3.5 text-muted-foreground transition-transform {isExpanded
										? 'rotate-90'
										: ''}"
								/>
							</td>
							<td class="py-1.5 pl-1">
								<span
									class="block h-1.5 w-1.5 rounded-full {online
										? 'bg-emerald-500'
										: 'bg-muted-foreground/30'}"
								></span>
							</td>
							<td class="overflow-hidden px-2 py-1.5">
								<a
									href="/devices/{device.spec?.deviceId}"
									class="block truncate font-mono font-medium hover:underline"
									onclick={(e) => e.stopPropagation()}
									title={device.meta?.name ?? '—'}
								>
									{device.meta?.name ?? '—'}
								</a>
							</td>
							<td class="overflow-hidden px-2 py-1.5 text-muted-foreground">
								<span class="block truncate font-mono">{device.meta?.namespace ?? '—'}</span>
							</td>
							{#if !narrow}
								<td class="px-2 py-1.5 text-muted-foreground">
									{#if device.status?.lastHearthbeat}
										<TimeTooltip
											timestamp={device.status.lastHearthbeat}
											class="text-xs text-muted-foreground"
										/>
									{:else}
										—
									{/if}
								</td>
							{/if}
						</tr>
						{#if isExpanded}
							<tr class="border-b border-border/50">
								<td colspan="5" class="p-0">
									<div class="flex flex-col gap-2 px-8 py-3 text-[11px]">
										<div class="flex items-baseline gap-2">
											<span class="w-16 shrink-0 uppercase tracking-wide text-muted-foreground"
												>ID</span
											>
											<span class="font-mono">{device.spec?.deviceId ?? '—'}</span>
										</div>
										{#if Object.keys(labels).length > 0}
											<div class="flex items-baseline gap-2">
												<span
													class="w-16 shrink-0 uppercase tracking-wide text-muted-foreground"
													>Labels</span
												>
												<div class="flex flex-wrap gap-1">
													{#each Object.entries(labels) as [k, v] (k)}
														<Badge variant="secondary" class="font-mono text-[10px] font-normal"
															>{k}={v}</Badge
														>
													{/each}
												</div>
											</div>
										{/if}
										{#if packages.length > 0}
											<div class="flex items-baseline gap-2">
												<span
													class="w-16 shrink-0 uppercase tracking-wide text-muted-foreground"
													>Schemas</span
												>
												<div class="flex flex-wrap gap-1">
													{#each packages as pkg (pkg)}
														<Badge variant="outline" class="font-mono text-[10px] font-normal"
															>{pkg}</Badge
														>
													{/each}
												</div>
											</div>
										{/if}
									</div>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
