<script lang="ts">
	import { getTooltipContext, Tooltip } from 'layerchart';
	import { cn } from '$lib/utils';
	import type { Device } from '@mir/sdk';

	let {
		onlineDevices,
		offlineDevices
	}: {
		onlineDevices: Device[];
		offlineDevices: Device[];
	} = $props();

	const tooltipCtx = getTooltipContext();

	const key = $derived((tooltipCtx.payload?.[0]?.key ?? '') as string);
	const list = $derived(key === 'online' ? onlineDevices : offlineDevices);
	const isOnline = $derived(key === 'online');
</script>

<Tooltip.Root variant="none">
	{#if key && list.length > 0}
		<div class="rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl">
			<p class="mb-1 font-medium">{isOnline ? 'Online' : 'Offline'} ({list.length})</p>
			{#each list as device (device.spec?.deviceId)}
				<div class="flex items-center gap-1.5 py-0.5">
					<span class={cn(
						'h-1.5 w-1.5 shrink-0 rounded-full',
						isOnline ? 'bg-[hsl(150_65%_40%)]' : 'bg-[hsl(220_9%_72%)]'
					)}></span>
					<span class="text-muted-foreground">{device.meta?.name ?? device.spec?.deviceId ?? '—'}</span>
				</div>
			{/each}
		</div>
	{/if}
</Tooltip.Root>
