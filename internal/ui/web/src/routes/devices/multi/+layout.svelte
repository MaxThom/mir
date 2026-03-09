<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { ROUTES } from '$lib/shared/constants/routes';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { cn } from '$lib/utils';
	import XIcon from '@lucide/svelte/icons/x';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';

	let { children } = $props();

	// Guard: redirect to device list if no devices selected
	$effect(() => {
		if (selectionStore.count === 0) {
			goto(ROUTES.DEVICES.LIST);
		}
	});

	const TABS = [
		{ label: 'Telemetry', href: ROUTES.DEVICES.MULTI.TELEMETRY },
		{ label: 'Commands', href: ROUTES.DEVICES.MULTI.COMMANDS },
		{ label: 'Configuration', href: ROUTES.DEVICES.MULTI.CONFIG },
	];

	let isActive = (href: string) => {
		const current = page.url.pathname;
		return current === href || current.startsWith(href + '/');
	};
</script>

<div class="flex flex-col">
	<!-- Header -->
	<div class="border-b bg-background px-4 pt-2 pb-0">
		<!-- Breadcrumb -->
		<div class="flex items-center gap-2 pb-2">
			<button
				onclick={() => goto(ROUTES.DEVICES.LIST)}
				class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
			>
				<ChevronLeftIcon class="size-3.5" />
				Devices
			</button>
			<span class="text-xs text-muted-foreground">/</span>
			<span class="text-xs font-medium">Bulk Operations</span>
			<Badge variant="secondary" class="ml-1 text-xs">{selectionStore.count}</Badge>
		</div>

		<!-- Selected device chips -->
		<div class="flex flex-wrap items-center gap-1.5 pb-2">
			{#each selectionStore.selectedDevices as device (device.spec.deviceId)}
				<span class="flex items-center gap-1 rounded-full border bg-muted/50 px-2 py-0.5 font-mono text-xs">
					<span class={cn(
						'h-1.5 w-1.5 shrink-0 rounded-full',
						device.status?.online ? 'bg-emerald-500' : 'bg-muted-foreground/30'
					)}></span>
					{device.meta?.name}/{device.meta?.namespace}
					<button
						onclick={() => selectionStore.deselect(device.spec.deviceId)}
						class="ml-0.5 text-muted-foreground transition-colors hover:text-foreground"
						aria-label="Remove {device.meta?.name} from selection"
					>
						<XIcon class="size-3" />
					</button>
				</span>
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
