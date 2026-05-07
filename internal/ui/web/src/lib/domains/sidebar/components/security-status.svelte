<script lang="ts">
	import * as Tooltip from '$lib/shared/components/shadcn/tooltip/index.js';
	import ShieldCheckIcon from '@lucide/svelte/icons/shield-check';
	import ShieldAlertIcon from '@lucide/svelte/icons/shield-alert';
	import LockIcon from '@lucide/svelte/icons/lock';
	import LockOpenIcon from '@lucide/svelte/icons/lock-open';
	import WifiIcon from '@lucide/svelte/icons/wifi';
	import WifiOffIcon from '@lucide/svelte/icons/wifi-off';
	import { contextStore } from '$lib/domains/contexts/stores/contexts.svelte';

	const isHttps = typeof window !== 'undefined' ? window.location.protocol === 'https:' : false;
	const isSecured = $derived(contextStore.activeContext?.secured ?? false);
	const isWss = $derived(contextStore.activeContext?.target?.startsWith('wss://') ?? false);
</script>

<!-- Expanded: three indicators in a row -->
<div class="mt-1 flex items-center justify-center gap-3 group-data-[collapsible=icon]:hidden">
	<Tooltip.Root>
		<Tooltip.Trigger class="flex items-center gap-1 text-xs">
			{#if isHttps}
				<ShieldCheckIcon class="size-3.5 text-green-500" />
				<span class="text-sidebar-foreground/70">HTTPS</span>
			{:else}
				<ShieldAlertIcon class="size-3.5 text-yellow-500" />
				<span class="text-sidebar-foreground/50">HTTPS</span>
			{/if}
		</Tooltip.Trigger>
		<Tooltip.Content side="top">
			<span class="text-xs"
				>{isHttps ? 'Cockpit served over HTTPS' : 'Cockpit served over HTTP'}</span
			>
		</Tooltip.Content>
	</Tooltip.Root>

	<Tooltip.Root>
		<Tooltip.Trigger class="flex items-center gap-1 text-xs">
			{#if isWss}
				<WifiIcon class="size-3.5 text-green-500" />
				<span class="text-sidebar-foreground/70">WSS</span>
			{:else}
				<WifiOffIcon class="size-3.5 text-yellow-500" />
				<span class="text-sidebar-foreground/50">WSS</span>
			{/if}
		</Tooltip.Trigger>
		<Tooltip.Content side="top">
			<span class="text-xs">{isWss ? 'NATS connected over WSS' : 'NATS connected over WS'}</span>
		</Tooltip.Content>
	</Tooltip.Root>

	<Tooltip.Root>
		<Tooltip.Trigger class="flex items-center gap-1 text-xs">
			{#if isSecured}
				<LockIcon class="size-3.5 text-green-500" />
				<span class="text-sidebar-foreground/70">Auth</span>
			{:else}
				<LockOpenIcon class="size-3.5 text-sidebar-foreground/30" />
				<span class="text-sidebar-foreground/50">Auth</span>
			{/if}
		</Tooltip.Trigger>
		<Tooltip.Content side="top">
			<span class="text-xs">{isSecured ? 'NATS authenticated' : 'NATS unauthenticated'}</span>
		</Tooltip.Content>
	</Tooltip.Root>
</div>
