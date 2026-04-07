<script lang="ts">
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Button } from '$lib/shared/components/shadcn/button';
	import GripVerticalIcon from '@lucide/svelte/icons/grip-vertical';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import XIcon from '@lucide/svelte/icons/x';
	import type { Snippet } from 'svelte';

	let {
		title,
		editMode = false,
		onEdit,
		onRemove,
		children,
		headerExtra
	}: {
		title: string;
		editMode?: boolean;
		onEdit?: () => void;
		onRemove?: () => void;
		children: Snippet;
		headerExtra?: Snippet;
	} = $props();
</script>

<Card.Root class="flex h-full flex-col gap-0 py-4">
	<Card.Header class="flex flex-row items-center gap-2 px-2 pb-2">
		<span
			class="grid-stack-item-content-drag-handle text-muted-foreground {editMode
				? 'cursor-grab'
				: 'pointer-events-none invisible'}"
		>
			<GripVerticalIcon class="h-4 w-4" />
		</span>
		<Card.Title class="truncate text-sm">{title}</Card.Title>
		{#if headerExtra}
			<div class="flex min-w-0 flex-1 items-center gap-1 overflow-hidden">
				{@render headerExtra()}
			</div>
		{:else}
			<div class="flex-1"></div>
		{/if}
		{#if editMode && onEdit}
			<Button
				variant="ghost"
				size="icon"
				class="h-6 w-6 shrink-0"
				onclick={onEdit}
				aria-label="Edit widget"
			>
				<PencilIcon class="h-3 w-3" />
			</Button>
		{/if}
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
	<div class="border-b"></div>
	<Card.Content class="min-h-0 flex-1 overflow-visible p-0">
		{@render children()}
	</Card.Content>
</Card.Root>
