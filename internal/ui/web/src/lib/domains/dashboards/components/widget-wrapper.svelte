<script lang="ts">
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Button } from '$lib/shared/components/shadcn/button';
	import GripVerticalIcon from '@lucide/svelte/icons/grip-vertical';
	import XIcon from '@lucide/svelte/icons/x';
	import type { Snippet } from 'svelte';

	let {
		title,
		editMode = false,
		onRemove,
		children
	}: {
		title: string;
		editMode?: boolean;
		onRemove?: () => void;
		children: Snippet;
	} = $props();
</script>

<Card.Root class="flex h-full flex-col gap-0 overflow-hidden py-4">
	<Card.Header class="flex flex-row items-center gap-2 px-2">
		<span
			class="grid-stack-item-content-drag-handle text-muted-foreground {editMode
				? 'cursor-grab'
				: 'pointer-events-none invisible'}"
		>
			<GripVerticalIcon class="h-4 w-4" />
		</span>
		<Card.Title class="flex-1 truncate text-sm">{title}</Card.Title>
		{#if editMode && onRemove}
			<Button
				variant="ghost"
				size="icon"
				class="h-6 w-6 shrink-0"
				onclick={onRemove}
				aria-label="Remove widget"
			>
				<XIcon class="h-3 w-3" />
			</Button>
		{/if}
	</Card.Header>
	<Card.Content class="min-h-0 flex-1 overflow-auto p-0">
		{@render children()}
	</Card.Content>
</Card.Root>
