<script lang="ts">
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import XIcon from '@lucide/svelte/icons/x';

	let {
		confirmValue,
		confirmHint,
		error,
		isDeleting,
		onconfirm
	}: {
		confirmValue: string;
		confirmHint: string;
		error?: string | null;
		isDeleting: boolean;
		onconfirm: () => void;
	} = $props();

	let isConfirming = $state(false);
	let inputText = $state('');

	let matches = $derived(inputText === confirmValue);

	function startConfirm() {
		inputText = '';
		isConfirming = true;
	}

	function cancel() {
		isConfirming = false;
		inputText = '';
	}
</script>

{#if isConfirming}
	<div class="flex items-center gap-1.5">
		{#if error}
			<span class="text-xs text-destructive">{error}</span>
		{/if}
		<div class="relative">
			<Input
				bind:value={inputText}
				placeholder={confirmValue}
				class="h-7 w-48 font-mono text-xs"
				autofocus
				onkeydown={(e) => e.key === 'Escape' && cancel()}
			/>
			<span class="absolute top-full left-0 z-50 mt-1 text-xs font-medium whitespace-nowrap text-destructive">
				{confirmHint}
			</span>
		</div>
		<Button
			variant="destructive"
			size="sm"
			class="h-7 text-xs"
			onclick={onconfirm}
			disabled={!matches || isDeleting}
		>
			{#if isDeleting}<Spinner class="mr-1 size-3" />{/if}
			Delete
		</Button>
		<Button variant="ghost" size="icon-sm" class="size-7" onclick={cancel}>
			<XIcon class="size-3.5" />
		</Button>
	</div>
{:else}
	<Button
		variant="ghost"
		size="icon-sm"
		class="text-destructive hover:text-destructive"
		onclick={startConfirm}
	>
		<Trash2Icon class="size-3.5" />
		<span class="sr-only">Delete</span>
	</Button>
{/if}
