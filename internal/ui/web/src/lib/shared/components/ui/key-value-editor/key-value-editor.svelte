<script lang="ts">
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import XIcon from '@lucide/svelte/icons/x';
	import PlusIcon from '@lucide/svelte/icons/plus';

	let {
		items = $bindable([]),
		isEditing = false,
		variant = 'badge',
		addLabel = 'Add item'
	}: {
		items: { key: string; value: string }[];
		isEditing?: boolean;
		variant?: 'badge' | 'list';
		addLabel?: string;
	} = $props();
</script>

{#if isEditing}
	<div class="space-y-1.5">
		{#each items as item, i (i)}
			<div class="flex items-center gap-1">
				<Input bind:value={item.key} placeholder="key" class="h-7 w-24 font-mono text-xs" />
				<span class="text-muted-foreground">=</span>
				<Input bind:value={item.value} placeholder="value" class="h-7 flex-1 font-mono text-xs" />
				<Button variant="ghost" size="icon-sm" onclick={() => items.splice(i, 1)} class="size-7">
					<XIcon class="size-3" />
				</Button>
			</div>
		{/each}
		<Button
			variant="ghost"
			size="sm"
			onclick={() => items.push({ key: '', value: '' })}
			class="h-7 gap-1 px-2 text-xs"
		>
			<PlusIcon class="size-3" />
			{addLabel}
		</Button>
	</div>
{:else if variant === 'badge'}
	{#if items.length > 0}
		<div class="flex flex-wrap gap-1">
			{#each items as { key, value } (key)}
				<Badge variant="secondary" class="font-mono text-xs font-normal">{key}={value}</Badge>
			{/each}
		</div>
	{:else}
		<span class="text-sm text-muted-foreground">—</span>
	{/if}
{:else}
	{#if items.length > 0}
		<div class="space-y-1">
			{#each items as { key, value } (key)}
				<div class="flex gap-2">
					<span class="font-mono text-xs text-muted-foreground">{key}:</span>
					<span class="font-mono text-xs">{value}</span>
				</div>
			{/each}
		</div>
	{:else}
		<span class="text-sm text-muted-foreground">—</span>
	{/if}
{/if}
