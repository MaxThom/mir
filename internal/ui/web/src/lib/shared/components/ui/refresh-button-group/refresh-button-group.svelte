<script lang="ts">
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as ButtonGroup from '$lib/shared/components/shadcn/button-group';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { cn } from '$lib/utils';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	let {
		isLoading = false,
		onRefresh
	}: {
		isLoading?: boolean;
		onRefresh?: () => void;
	} = $props();

	const INTERVALS = [
		{ value: 0, label: 'Off' },
		{ value: 5, label: '5s' },
		{ value: 10, label: '10s' },
		{ value: 30, label: '30s' },
		{ value: 60, label: '1m' }
	] as const;

	let intervalLabel = $derived(INTERVALS.find((i) => i.value === editorPrefs.refreshInterval)?.label ?? '10s');

	$effect(() => {
		if (editorPrefs.refreshInterval === 0 || !onRefresh) return;
		const id = setInterval(() => onRefresh?.(), editorPrefs.refreshInterval * 1000);
		return () => clearInterval(id);
	});
</script>

<ButtonGroup.Root>
	<Button
		variant="secondary"
		size="icon-sm"
		onclick={() => onRefresh?.()}
		disabled={isLoading || !onRefresh}
	>
		{#if isLoading}
			<Spinner class="size-3.5" />
		{:else}
			<RefreshCwIcon class="size-3.5" />
		{/if}
	</Button>
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button {...props} variant="secondary" size="sm" class="px-1.5 text-xs">
					{intervalLabel}
					<ChevronDownIcon class="size-3" />
				</Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end" class="min-w-24">
			{#each INTERVALS as interval (interval.value)}
				<DropdownMenu.Item
					class={cn('text-xs', editorPrefs.refreshInterval === interval.value && 'font-medium')}
					onclick={() => editorPrefs.setRefreshInterval(interval.value)}
				>
					{interval.label}
				</DropdownMenu.Item>
			{/each}
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</ButtonGroup.Root>
