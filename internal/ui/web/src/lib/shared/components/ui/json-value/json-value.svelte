<script lang="ts">
	import JsonValue from './json-value.svelte';

	let { value }: { value: unknown } = $props();

	let isObject = $derived(value !== null && typeof value === 'object' && !Array.isArray(value));
	let isArray = $derived(Array.isArray(value));

	let entries = $derived<[string, unknown][] | null>(
		isObject
			? Object.entries(value as Record<string, unknown>)
			: isArray
				? (value as unknown[]).map((v, i) => [String(i), v] as [string, unknown])
				: null
	);
</script>

{#if entries !== null && entries.length > 0}
	<div class="border-l border-border/40 pl-2">
		{#each entries as [k, v] (k)}
			<div class="flex gap-1.5">
				<span class="shrink-0 font-mono text-xs text-muted-foreground">{k}:</span>
				<span class="min-w-0 font-mono text-xs">
					<JsonValue value={v} />
				</span>
			</div>
		{/each}
	</div>
{:else if value === null || value === undefined}
	<span class="font-mono text-xs text-muted-foreground/50">—</span>
{:else if value === ''}
	<span class="font-mono text-xs text-muted-foreground/50">""</span>
{:else}
	<span class="font-mono text-xs">{String(value)}</span>
{/if}
