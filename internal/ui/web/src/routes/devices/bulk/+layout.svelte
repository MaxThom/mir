<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { setContext } from 'svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { ROUTES } from '$lib/shared/constants/routes';
	import { cn } from '$lib/utils';
	import { resolve } from '$app/paths';
	import { RefreshButtonGroup } from '$lib/shared/components/ui/refresh-button-group';

	let { children } = $props();

	// Guard: redirect to device list if no devices selected
	$effect(() => {
		if (selectionStore.count === 0) {
			goto(resolve(ROUTES.DEVICES.LIST));
		}
	});

	let tabRefreshFn = $state<(() => Promise<void> | void) | null>(null);
	let isRefreshing = $state(false);

	async function handleRefresh() {
		isRefreshing = true;
		try {
			await tabRefreshFn?.();
		} finally {
			isRefreshing = false;
		}
	}

	setContext('multi', {
		setTabRefresh(fn: (() => void) | null) {
			tabRefreshFn = fn;
		}
	});

	const TABS = [
		{ label: 'Telemetry', href: ROUTES.DEVICES.MULTI.TELEMETRY },
		{ label: 'Commands', href: ROUTES.DEVICES.MULTI.COMMANDS },
		{ label: 'Configuration', href: ROUTES.DEVICES.MULTI.CONFIG }
	];

	let isActive = (href: string) => {
		const current = page.url.pathname;
		return current === href || current.startsWith(href + '/');
	};
</script>

<div class="flex flex-col">
	<!-- Header -->
	<div class="border-b bg-background px-4 pt-2 pb-0">
		<!-- Title row -->
		<div class="flex items-center gap-2 pb-2">
			<span class="text-sm font-medium">Bulk Operations</span>
			<span class="text-xs text-muted-foreground">
				{#if selectionStore.activeCount < selectionStore.count}
					{selectionStore.activeCount} of {selectionStore.count} devices
				{:else}
					{selectionStore.count} devices
				{/if}
			</span>
			<div class="ml-auto">
				<RefreshButtonGroup onRefresh={handleRefresh} isLoading={isRefreshing} />
			</div>
		</div>

		<!-- Selected device chips -->
		<div class="flex flex-wrap items-center gap-1.5 pb-2">
			{#each selectionStore.selectedDevices as device (device.spec.deviceId)}
				{@const disabled = selectionStore.isDisabled(device.spec.deviceId)}
				<button
					onclick={() => selectionStore.toggleDisabled(device.spec.deviceId)}
					class={cn(
						'flex items-center gap-1 rounded-full border px-2 py-0.5 font-mono text-xs transition-opacity',
						disabled ? 'opacity-35 hover:opacity-60' : 'bg-muted/50 hover:opacity-80'
					)}
					title={disabled ? 'Click to re-enable' : 'Click to exclude'}
				>
					<span
						class={cn(
							'h-1.5 w-1.5 shrink-0 rounded-full',
							device.status?.online ? 'bg-emerald-500' : 'bg-muted-foreground/30'
						)}
					></span>
					{device.meta?.name}/{device.meta?.namespace}
				</button>
			{/each}
		</div>

		<!-- Tab navigation -->
		<nav class="-mb-px flex gap-0">
			{#each TABS as tab (tab.label)}
				<a
					href={tab.href}
					class={cn(
						'border-b-2 px-3 py-2 text-sm font-medium transition-colors',
						isActive(tab.href)
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
