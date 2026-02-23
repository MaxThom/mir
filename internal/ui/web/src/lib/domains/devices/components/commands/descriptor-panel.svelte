<script lang="ts">
	import type { Descriptor } from '$lib/domains/devices/types/types';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import SearchIcon from '@lucide/svelte/icons/search';

	let {
		title,
		items,
		isLoading,
		error,
		groupErrors,
		selectedName,
		emptyText = 'No items found.',
		onSelect
	}: {
		title: string;
		items: Descriptor[];
		isLoading: boolean;
		error: string | null;
		groupErrors: string[];
		selectedName: string | null;
		emptyText?: string;
		onSelect: (desc: Descriptor) => void;
	} = $props();

	let searchQuery = $state('');
	let filteredItems = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return items;
		return items.filter((desc) => {
			if (desc.name.toLowerCase().includes(q)) return true;
			return Object.entries(desc.labels ?? {}).some(
				([k, v]) =>
					k.toLowerCase().includes(q) ||
					String(v).toLowerCase().includes(q) ||
					`${k}=${v}`.toLowerCase().includes(q)
			);
		});
	});
</script>

<div class="flex w-64 shrink-0 flex-col overflow-hidden border-r">
	<!-- Panel header -->
	<div class="flex items-center border-b px-3 py-2.75">
		<span class="text-xs font-medium tracking-wide text-muted-foreground uppercase">{title}</span>
	</div>

	<!-- Search -->
	<div class="flex items-center gap-2 border-b px-3 py-1.5">
		<SearchIcon class="size-3.5 shrink-0 text-muted-foreground" />
		<input
			bind:value={searchQuery}
			placeholder="name or label…"
			class="w-full bg-transparent py-1 text-xs outline-none placeholder:text-muted-foreground/60"
		/>
	</div>

	<!-- Loading / error / list -->
	<div class="flex-1 overflow-y-auto">
		{#if isLoading && items.length === 0}
			<div class="flex items-center gap-2 px-3 py-4 text-xs text-muted-foreground">
				<Spinner class="size-3" />
				Loading…
			</div>
		{:else if error}
			<p class="px-3 py-4 text-xs text-destructive">{error}</p>
		{:else}
			{#each groupErrors as err}
				<p class="border-b bg-yellow-500/10 px-3 py-2 text-xs text-yellow-700 dark:text-yellow-400">
					{err}
				</p>
			{/each}
			{#if items.length === 0}
				<p class="px-3 py-4 text-xs text-muted-foreground">{emptyText}</p>
			{:else if filteredItems.length === 0}
				<p class="px-3 py-4 text-xs text-muted-foreground">No match for "{searchQuery}".</p>
			{:else}
				{#each filteredItems as desc (desc.name)}
					<button
						onclick={() => onSelect(desc)}
						class="flex w-full flex-col gap-1.5 border-b px-3 py-2.5 text-left transition-colors last:border-0 hover:bg-accent
							{selectedName === desc.name ? 'bg-accent' : ''}"
					>
						<span class="truncate font-mono text-xs font-medium">
							{#if desc.error}
								<span class="translate-y-1px mr-1 inline-block size-1.5 rounded-full bg-destructive"
								></span>
							{/if}{desc.name}</span
						>
						<div class="flex flex-wrap gap-1">
							{#each Object.entries(desc.labels ?? {}) as [k, v] (k)}
								<span
									class="rounded-sm border border-border/60 bg-muted/60 px-1.5 py-px font-mono text-[11px] text-muted-foreground"
								>
									{k}={v}
								</span>
							{/each}
						</div>
					</button>
				{/each}
			{/if}
		{/if}
	</div>
</div>
