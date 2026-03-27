<script lang="ts">
	import { Input } from '$lib/shared/components/shadcn/input';

	let {
		value = $bindable(''),
		suggestions,
		placeholder = '',
		disabled = false,
		onchange,
		class: className = ''
	}: {
		value?: string;
		suggestions: string[];
		placeholder?: string;
		disabled?: boolean;
		onchange?: (value: string) => void;
		class?: string;
	} = $props();

	let open = $state(false);

	const filtered = $derived(
		value
			? suggestions.filter((s) => s.toLowerCase().includes(value.toLowerCase()) && s !== value)
			: suggestions
	);

	function select(s: string) {
		value = s;
		onchange?.(s);
		open = false;
	}
</script>

<div class="relative">
	<Input
		bind:value
		{placeholder}
		{disabled}
		class={className}
		oninput={() => onchange?.(value)}
		onfocus={() => (open = true)}
		onblur={() => setTimeout(() => (open = false), 150)}
	/>
	{#if open && filtered.length > 0}
		<div
			class="absolute left-0 right-0 top-full z-50 mt-1 max-h-48 overflow-y-auto rounded-md border bg-popover p-1 shadow-md"
		>
			{#each filtered as s (s)}
				<button
					class="w-full rounded px-2 py-1.5 text-left font-mono text-xs hover:bg-accent hover:text-accent-foreground"
					onmousedown={(e) => {
						e.preventDefault();
						select(s);
					}}
				>
					{s}
				</button>
			{/each}
		</div>
	{/if}
</div>
