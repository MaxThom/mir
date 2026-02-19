<script lang="ts">
	import SearchIcon from '@lucide/svelte/icons/search';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as ButtonGroup from '$lib/components/ui/button-group';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Spinner } from '$lib/components/ui/spinner';
	import { cn } from '$lib/utils';

	let {
		deviceCount,
		onlineCount,
		globalFilter,
		isLoading = false,
		onRefresh,
		onglobalfilterchange
	}: {
		deviceCount: number;
		onlineCount: number;
		globalFilter: string;
		isLoading?: boolean;
		onRefresh?: () => void;
		onglobalfilterchange: (value: string) => void;
	} = $props();

	const INTERVALS = [
		{ value: 0, label: 'Off' },
		{ value: 5, label: '5s' },
		{ value: 10, label: '10s' },
		{ value: 30, label: '30s' },
		{ value: 60, label: '1m' }
	] as const;

	let refreshInterval = $state(10);
	let intervalLabel = $derived(INTERVALS.find((i) => i.value === refreshInterval)?.label ?? '10s');

	$effect(() => {
		if (refreshInterval === 0 || !onRefresh) return;
		const id = setInterval(() => onRefresh?.(), refreshInterval * 1000);
		return () => clearInterval(id);
	});
</script>

<div class="flex items-center justify-between border-b px-6 py-4">
	<div class="flex items-center gap-3">
		<span class="text-sm font-semibold">Devices</span>
		<Badge variant="secondary" class="tabular-nums">{deviceCount}</Badge>
		<div class="relative">
			<SearchIcon
				class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
			/>
			<Input
				type="search"
				placeholder="Search…"
				class="h-7 w-48 rounded-md pl-8 text-xs transition-[width] focus:w-64"
				value={globalFilter}
				oninput={(e) => onglobalfilterchange((e.target as HTMLInputElement).value)}
			/>
		</div>
	</div>
	<div class="flex items-center gap-3">
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
			{onlineCount} online
		</div>
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
					<RefreshCwIcon class="h-3.5 w-3.5" />
				{/if}
			</Button>
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="secondary" size="sm" class="px-1.5 text-xs">
							{intervalLabel}
							<ChevronDownIcon class="h-3 w-3" />
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="min-w-24">
					{#each INTERVALS as interval (interval.value)}
						<DropdownMenu.Item
							class={cn('text-xs', refreshInterval === interval.value && 'font-medium')}
							onclick={() => (refreshInterval = interval.value)}
						>
							{interval.label}
						</DropdownMenu.Item>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</ButtonGroup.Root>
	</div>
</div>
