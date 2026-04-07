<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { Device } from '@mir/sdk';
	import type { DeviceWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { JsonValue } from '$lib/shared/components/ui/json-value';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { cn } from '$lib/utils';
	import CircleCheckBigIcon from '@lucide/svelte/icons/circle-check-big';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { PieChart } from 'layerchart';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import { ChartContainer } from '$lib/shared/components/shadcn/chart';

	let {
		config,
		widgetId,
		onDevicesReady,
		refreshTick = 0
	}: {
		config: DeviceWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
		refreshTick?: number;
	} = $props();

	let devices = $state<Device[]>([]);
	let isLoading = $state(false);
	let hasLoaded = $state(false);
	let loadError = $state<string | null>(null);
	let isInRefresh = false;

	let pillsEl = $state<HTMLDivElement | undefined>(undefined);
	let pillsWrapEl = $state<HTMLDivElement | undefined>(undefined);
	let hasOverflow = $state(false);
	let dropdownOpen = $state(false);

	$effect(() => {
		if (!pillsEl) return;
		function check() {
			if (!pillsEl) return;
			hasOverflow = pillsEl.scrollWidth > pillsEl.clientWidth + 2;
		}
		check();
		const ro = new ResizeObserver(check);
		ro.observe(pillsEl);
		return () => ro.disconnect();
	});

	$effect(() => {
		if (!dropdownOpen) return;
		function handleClick(e: MouseEvent) {
			if (pillsWrapEl && !pillsWrapEl.contains(e.target as Node)) {
				dropdownOpen = false;
			}
		}
		document.addEventListener('click', handleClick);
		return () => document.removeEventListener('click', handleClick);
	});

	const selectedDevice = $derived(
		devices.find((d) => d.spec?.deviceId === config.selectedDeviceId) ?? devices[0] ?? null
	);

	const deviceInfos = $derived(
		devices.map((d, i) => ({
			id: d.spec?.deviceId ?? '',
			name: d.meta?.name ?? '',
			color: CHART_COLORS[i % CHART_COLORS.length]
		}))
	);

	const pillsDeviceInfos = $derived(() => {
		const selectedId = selectedDevice?.spec?.deviceId ?? '';
		const idx = deviceInfos.findIndex((d) => d.id === selectedId);
		if (idx <= 0) return deviceInfos;
		return [deviceInfos[idx], ...deviceInfos.slice(0, idx), ...deviceInfos.slice(idx + 1)];
	});

	const onlineCount  = $derived(devices.filter((d) =>  d.status?.online).length);
	const offlineCount = $derived(devices.filter((d) => !d.status?.online).length);

	const statusChartData = $derived([
		{ key: 'online',  label: 'Online',  value: onlineCount  },
		{ key: 'offline', label: 'Offline', value: offlineCount }
	]);

	const statusChartConfig: ChartConfig = {
		online:  { label: 'Online',  color: 'hsl(162 63% 41%)' },
		offline: { label: 'Offline', color: 'hsl(0 65% 55%)' }
	};

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadDevices);
		} else {
			devices = [];
			hasLoaded = false;
		}
	});

	$effect(() => {
		if (refreshTick > 0 && mirStore.mir && hasLoaded) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			untrack(loadDevices);
		}
	});

	async function loadDevices() {
		const mir = mirStore.mir;
		if (!mir) return;
		if (!hasLoaded) isLoading = true;
		loadError = null;
		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			devices = await mir.client().listDevices().request(target, false);
			onDevicesReady?.(
				devices.map((d, i) => ({
					id: d.spec?.deviceId ?? '',
					name: d.meta?.name ?? '',
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);
			// Default to first device if savedId is gone from the list
			if (
				devices.length > 0 &&
				!devices.some((d) => d.spec?.deviceId === config.selectedDeviceId)
			) {
				const firstId = devices[0].spec?.deviceId;
				if (firstId) selectDevice(firstId);
			}
			hasLoaded = true;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load devices';
		} finally {
			isLoading = false;
			if (isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
		}
	}

	function selectDevice(deviceId: string) {
		if (!dashboardStore.activeDashboard) return;
		dashboardStore.saveWidgetViewState(widgetId, { ...config, selectedDeviceId: deviceId });
	}

	function isMatchingDesired(device: Device, key: string, reportedVal: unknown): boolean {
		const desired = (device.properties?.desired ?? {}) as Record<string, unknown>;
		if (!(key in desired)) return false;
		return JSON.stringify(desired[key]) === JSON.stringify(reportedVal);
	}
</script>

<div class="flex h-full flex-col overflow-hidden">
	{#if isLoading}
		<div class="flex flex-1 items-center justify-center">
			<span class="text-xs text-muted-foreground">Loading…</span>
		</div>
	{:else if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if devices.length === 0 && hasLoaded}
		<p class="p-4 text-xs text-muted-foreground">No devices found for this target.</p>
	{:else if selectedDevice}
		<!-- Device selector (hidden for status view — aggregate across all devices) -->
		{#if config.view !== 'status' && devices.length > 1}
			<div class="relative shrink-0 border-b" bind:this={pillsWrapEl}>
				<div bind:this={pillsEl} class="pills-scroll flex items-center gap-1.5 overflow-x-auto px-3 py-1.5" class:pr-8={hasOverflow}>
					{#each pillsDeviceInfos() as info (info.id)}
						<button
							onclick={() => selectDevice(info.id)}
							class={cn(
								'shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
								info.id === (selectedDevice.spec?.deviceId ?? '')
									? 'bg-primary text-primary-foreground'
									: 'bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground'
							)}
						>
							{info.name}
						</button>
					{/each}
				</div>
				{#if hasOverflow}
					<div class="absolute inset-y-0 right-0 flex items-center bg-gradient-to-l from-background via-background/80 to-transparent pl-6 pr-1">
						<button
							onclick={() => (dropdownOpen = !dropdownOpen)}
							class={cn(
								'flex size-5 items-center justify-center rounded transition-colors hover:bg-muted',
								dropdownOpen ? 'text-foreground' : 'text-muted-foreground'
							)}
						>
							<ChevronDownIcon class="size-3" />
						</button>
					</div>
				{/if}
				{#if dropdownOpen}
					<div class="absolute left-0 right-0 top-full z-50 rounded-b-md border-x border-b bg-popover shadow-md">
						{#each pillsDeviceInfos() as info (info.id)}
							<button
								onclick={() => { selectDevice(info.id); dropdownOpen = false; }}
								class={cn(
									'flex w-full items-center px-3 py-1.5 text-left text-xs transition-colors hover:bg-accent',
									info.id === (selectedDevice.spec?.deviceId ?? '')
										? 'font-medium text-foreground'
										: 'text-muted-foreground'
								)}
							>
								{info.name}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		{/if}

		<div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
			{#if config.view === 'info'}
				<!-- ── Info view ────────────────────────────────────────────────── -->
				<div class="space-y-1">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Meta</p>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Name</span>
						<span class="flex-1 text-xs font-medium">{selectedDevice.meta?.name ?? '—'}</span>
					</div>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Namespace</span>
						<span class="flex-1 font-mono text-xs">{selectedDevice.meta?.namespace ?? '—'}</span>
					</div>

					<div class="flex items-start gap-2">
						<span class="w-20 shrink-0 pt-0.5 text-xs text-muted-foreground">Labels</span>
						<div class="flex flex-1 flex-wrap gap-1">
							{#each Object.entries(selectedDevice.meta?.labels ?? {}) as [k, v] (k)}
								<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
							{:else}
								<span class="text-xs text-muted-foreground">—</span>
							{/each}
						</div>
					</div>

					<div class="flex items-start gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Annotations</span>
						<div class="flex flex-1 flex-wrap gap-1">
							{#each Object.entries(selectedDevice.meta?.annotations ?? {}) as [k, v] (k)}
								<span class="text-xs text-muted-foreground">{k}: {v}</span>
							{:else}
								<span class="text-xs text-muted-foreground">—</span>
							{/each}
						</div>
					</div>
				</div>

				<Separator class="my-1.5" />

				<div class="space-y-1">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Spec</p>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Device ID</span>
						<span class="flex-1 truncate font-mono text-xs text-muted-foreground">
							{selectedDevice.spec?.deviceId ?? '—'}
						</span>
					</div>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Disabled</span>
						{#if selectedDevice.spec?.disabled}
							<Badge variant="destructive" class="text-xs">Yes</Badge>
						{:else}
							<span class="text-xs text-muted-foreground">No</span>
						{/if}
					</div>
				</div>

				<Separator class="my-1.5" />

				<div class="space-y-1">
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Status</p>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Connectivity</span>
						<div class="flex items-center gap-1.5">
							<span
								class={cn(
									'h-1.5 w-1.5 shrink-0 rounded-full',
									selectedDevice.status?.online
										? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
										: 'bg-muted-foreground/30'
								)}
							></span>
							<span
								class={cn(
									'text-xs font-medium',
									selectedDevice.status?.online
										? 'text-emerald-600 dark:text-emerald-400'
										: 'text-muted-foreground'
								)}
							>
								{selectedDevice.status?.online ? 'Online' : 'Offline'}
							</span>
						</div>
					</div>

					<div class="flex items-center gap-2">
						<span class="w-20 shrink-0 text-xs text-muted-foreground">Last Heartbeat</span>
						<div class="flex-1">
							{#if selectedDevice.status?.lastHearthbeat}
								<TimeTooltip
									timestamp={selectedDevice.status.lastHearthbeat}
									utc={editorPrefs.utc}
									class="text-xs hover:text-foreground"
								/>
							{:else}
								<span class="text-xs text-muted-foreground">—</span>
							{/if}
						</div>
					</div>

					<div class="flex items-start gap-2">
						<span class="w-20 shrink-0 pt-0.5 text-xs text-muted-foreground">Schema</span>
						<div class="flex-1">
							{#if selectedDevice.status?.schema?.packageNames?.length}
								<div class="flex flex-wrap gap-1">
									{#each selectedDevice.status.schema.packageNames as pkg (pkg)}
										<Badge variant="outline" class="font-mono text-xs font-normal">{pkg}</Badge>
									{/each}
								</div>
								{#if selectedDevice.status.schema.lastSchemaFetch}
									<TimeTooltip
										timestamp={selectedDevice.status.schema.lastSchemaFetch}
										utc={editorPrefs.utc}
										prefix="fetched "
										class="mt-0.5 text-xs text-muted-foreground hover:text-foreground"
									/>
								{/if}
							{:else}
								<span class="text-xs text-muted-foreground">Not loaded</span>
							{/if}
						</div>
					</div>
				</div>
			{:else if config.view === 'properties'}
				<!-- ── Properties view ──────────────────────────────────────────── -->
				{@const desiredProps = Object.entries(selectedDevice.properties?.desired ?? {}).sort(([a], [b]) => a.localeCompare(b))}
				{@const reportedProps = Object.entries(selectedDevice.properties?.reported ?? {}).sort(([a], [b]) => a.localeCompare(b))}

				{#if desiredProps.length === 0 && reportedProps.length === 0}
					<p class="text-sm text-muted-foreground">No properties configured.</p>
				{:else}
					<div class="space-y-3">
						<div>
							<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
								Desired
							</p>
							{#if desiredProps.length === 0}
								<p class="text-xs text-muted-foreground">—</p>
							{:else}
								<div class="space-y-1.5">
									{#each desiredProps as [k, v] (k)}
										<div class="flex flex-col">
											<div class="flex items-center gap-1.5">
												<span class="font-mono text-xs text-muted-foreground">{k}</span>
												{#if selectedDevice.status?.properties?.desired?.[k]}
													<TimeTooltip
														timestamp={selectedDevice.status.properties.desired[k]}
														utc={editorPrefs.utc}
														class="text-[10px] text-muted-foreground/60"
													/>
												{/if}
											</div>
											<JsonValue value={v} />
										</div>
									{/each}
								</div>
							{/if}
						</div>

						<Separator />

						<div>
							<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
								Reported
							</p>
							{#if reportedProps.length === 0}
								<p class="text-xs text-muted-foreground">—</p>
							{:else}
								<div class="space-y-1.5">
									{#each reportedProps as [k, v] (k)}
										<div class="flex flex-col">
											<div class="flex items-center gap-1.5">
												<span class="font-mono text-xs text-muted-foreground">{k}</span>
												{#if isMatchingDesired(selectedDevice, k, v)}
													<CircleCheckBigIcon class="size-3 text-emerald-500" />
												{/if}
												{#if selectedDevice.status?.properties?.reported?.[k]}
													<TimeTooltip
														timestamp={selectedDevice.status.properties.reported[k]}
														utc={editorPrefs.utc}
														class="text-[10px] text-muted-foreground/60"
													/>
												{/if}
											</div>
											<JsonValue value={v} />
										</div>
									{/each}
								</div>
							{/if}
						</div>
						</div>
					{/if}
				{:else if config.view === 'status'}
					<!-- ── Status view ──────────────────────────────────────────────── -->
					<div class="relative mb-2">
						<ChartContainer config={statusChartConfig} class="h-48 w-full">
							<PieChart
								data={statusChartData}
								key="key"
								label="label"
								value="value"
								innerRadius={0.55}
								cRange={['hsl(162 63% 41%)', 'hsl(220 9% 72%)']}
							/>
						</ChartContainer>
						<div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
							<span class="text-2xl font-bold">{onlineCount}</span>
							<span class="text-xs text-muted-foreground">of {devices.length} online</span>
						</div>
					</div>
					<div class="space-y-1">
						{#each devices as device (device.spec?.deviceId)}
							<div class="flex items-center gap-2">
								<span class={cn(
									'h-1.5 w-1.5 shrink-0 rounded-full',
									device.status?.online ? 'bg-emerald-500' : 'bg-muted-foreground/40'
								)}></span>
								<span class="text-xs text-muted-foreground">
									{device.meta?.name ?? device.spec?.deviceId ?? '—'}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
	{/if}
</div>

<style>
	.pills-scroll {
		scrollbar-width: none;
		-ms-overflow-style: none;
	}
	.pills-scroll::-webkit-scrollbar {
		display: none;
	}
</style>
