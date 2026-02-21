<script lang="ts">
	import { page } from '$app/state';
	import { setContext } from 'svelte';
	import { goto } from '$app/navigation';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { ROUTES } from '$lib/shared/constants/routes';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import * as Empty from '$lib/components/ui/empty';
	import { Separator } from '$lib/components/ui/separator';
	import { cn } from '$lib/utils';
	import { relativeTime, formatFullDate } from '$lib/shared/utils/time';
	import { Input } from '$lib/components/ui/input';
	import UnplugIcon from '@lucide/svelte/icons/unplug';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import XIcon from '@lucide/svelte/icons/x';
	import { RefreshButtonGroup } from '$lib/components/ui/refresh-button-group';
	import * as Tooltip from '$lib/components/ui/tooltip';

	let { children } = $props();

	let deviceId = $derived(page.params.deviceId ?? '');

	const TABS = [
		{ label: 'Overview', href: (id: string) => ROUTES.DEVICES.DETAIL(id) },
		{ label: 'Telemetry', href: (id: string) => ROUTES.DEVICES.TELEMETRY(id) },
		{ label: 'Commands', href: (id: string) => ROUTES.DEVICES.COMMANDS(id) },
		{ label: 'Config', href: (id: string) => ROUTES.DEVICES.CONFIG(id) },
		{ label: 'Events', href: (id: string) => ROUTES.DEVICES.EVENTS(id) },
		{ label: 'Schema', href: (id: string) => ROUTES.DEVICES.SCHEMA(id) }
	];

	let isActive = (tabHref: string) => {
		const current = page.url.pathname;
		if (tabHref === ROUTES.DEVICES.DETAIL(deviceId)) {
			return current === tabHref;
		}
		return current === tabHref || current.startsWith(tabHref + '/');
	};

	$effect(() => {
		if (mirStore.mir && deviceId) {
			deviceStore.loadDevice(mirStore.mir, deviceId);
		} else {
			deviceStore.resetDevice();
		}
	});

	function handleRefresh() {
		if (mirStore.mir && deviceId) {
			deviceStore.resetDevice();
			deviceStore.loadDevice(mirStore.mir, deviceId);
		}
	}

	// ── Delete confirmation ──────────────────────────────────────────────────────
	let isConfirmingDelete = $state(false);
	let deleteConfirmText = $state('');

	function startDelete() {
		deleteConfirmText = '';
		deviceStore.deleteError = null;
		isConfirmingDelete = true;
	}

	function cancelDelete() {
		isConfirmingDelete = false;
		deviceStore.deleteError = null;
	}

	async function confirmDelete() {
		if (!mirStore.mir || !device?.spec?.deviceId) return;
		try {
			await deviceStore.deleteDevice(mirStore.mir, device.spec.deviceId);
			goto('/devices');
		} catch {
			// error shown inline via deviceStore.deleteError
		}
	}

	let deleteNameMatches = $derived(
		deleteConfirmText ===
			`${deviceStore.selectedDevice?.meta?.name}/${deviceStore.selectedDevice?.meta?.namespace}`
	);

	setContext('device', {
		get device() {
			return deviceStore.selectedDevice;
		}
	});

	let device = $derived(deviceStore.selectedDevice);
	let labels = $derived(Object.entries(device?.meta?.labels ?? {}));
</script>

<div class="flex flex-col">
	<!-- Device header -->
	<div class="border-b bg-background px-4 pt-2 pb-0">
		{#if deviceStore.isLoadingDevice && !device}
			<div class="flex h-16 items-center gap-3">
				<Spinner class="size-4 text-muted-foreground" />
				<span class="text-sm text-muted-foreground">Loading device...</span>
			</div>
		{:else if deviceStore.deviceError}
			<div class="flex h-16 items-center gap-2">
				<UnplugIcon class="size-4 text-destructive" />
				<span class="text-sm text-destructive">{deviceStore.deviceError}</span>
			</div>
		{:else if device}
			<!-- Identity row -->
			<div class="flex items-center gap-3 pb-2">
				<!-- Online indicator + name -->
				<div class="flex items-center gap-2">
					<span
						class={cn(
							'h-2.5 w-2.5 shrink-0 rounded-full',
							device.status?.online
								? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
								: 'bg-muted-foreground/30'
						)}
					></span>
					<h1 class="text-lg font-semibold">{device.meta?.name ?? deviceId}</h1>
				</div>

				{#if device.spec?.disabled}
					<Badge variant="destructive" class="text-xs">Disabled</Badge>
				{/if}

				<!-- Heartbeat -->
				{#if device.status?.lastHearthbeat}
					<Tooltip.Root>
						<Tooltip.Trigger
							class="cursor-default text-xs text-muted-foreground underline decoration-dotted underline-offset-2 hover:text-foreground"
						>
							♥ {relativeTime(device.status.lastHearthbeat.seconds)}
						</Tooltip.Trigger>
						<Tooltip.Content>
							Last heartbeat: {formatFullDate(device.status.lastHearthbeat.seconds)}
						</Tooltip.Content>
					</Tooltip.Root>
				{/if}

				<div class="ml-auto flex items-center gap-2">
					{#if isConfirmingDelete}
						<div class="flex items-center gap-1.5">
							{#if deviceStore.deleteError}
								<span class="text-xs text-destructive">{deviceStore.deleteError}</span>
							{/if}
							<div class="relative">
								<Input
									bind:value={deleteConfirmText}
									placeholder="{device?.meta?.name}/{device?.meta?.namespace}"
									class="h-7 w-48 font-mono text-xs"
									autofocus
									onkeydown={(e) => e.key === 'Escape' && cancelDelete()}
								/>
								<span
									class="absolute top-full left-0 z-50 mt-1 text-xs font-medium whitespace-nowrap text-destructive"
								>
									Type <span class="font-mono">name/namespace</span> to confirm.
								</span>
							</div>
							<Button
								variant="destructive"
								size="sm"
								class="h-7 text-xs"
								onclick={confirmDelete}
								disabled={!deleteNameMatches || deviceStore.isDeleting}
							>
								{#if deviceStore.isDeleting}<Spinner class="mr-1 size-3" />{/if}
								Delete
							</Button>
							<Button variant="ghost" size="icon-sm" class="size-7" onclick={cancelDelete}>
								<XIcon class="size-3.5" />
							</Button>
						</div>
					{:else}
						<Button
							variant="ghost"
							size="icon-sm"
							class="text-destructive hover:text-destructive"
							onclick={startDelete}
						>
							<Trash2Icon class="size-3.5" />
							<span class="sr-only">Delete device</span>
						</Button>
					{/if}
					<RefreshButtonGroup isLoading={deviceStore.isLoadingDevice} onRefresh={handleRefresh} />
				</div>
			</div>

			<!-- Meta row: namespace, device ID, labels -->
			<div class="flex flex-wrap items-center gap-1.5 pb-2">
				<span class="font-mono text-xs text-muted-foreground">
					ns: <span class="text-foreground">{device.meta?.namespace ?? 'default'}</span>
				</span>
				<Separator orientation="vertical" class="h-3 data-[orientation=vertical]:h-3" />
				<span class="font-mono text-xs text-muted-foreground">
					id: <span class="text-foreground">{device.spec?.deviceId ?? '—'}</span>
				</span>
				{#if labels.length > 0}
					<Separator orientation="vertical" class="h-3 data-[orientation=vertical]:h-3" />
					{#each labels as [k, v] (k)}
						<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
					{/each}
				{/if}
			</div>
		{/if}

		<!-- Tab navigation -->
		<nav class="-mb-px flex gap-0">
			{#each TABS as tab (tab.label)}
				{@const href = tab.href(deviceId)}
				<a
					{href}
					class={cn(
						'border-b-2 px-3 py-2 text-sm font-medium transition-colors',
						isActive(href)
							? 'border-primary text-foreground'
							: 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
					)}
				>
					{tab.label}
				</a>
			{/each}
		</nav>
	</div>

	<!-- Tab content -->
	<div class="flex-1 p-4">
		{@render children()}
	</div>
</div>
