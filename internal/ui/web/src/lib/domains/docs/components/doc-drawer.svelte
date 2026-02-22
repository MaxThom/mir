<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import type { Component } from 'svelte';
	import { docsStore } from '../stores/docs.svelte';

	let { Content }: { Content: Component } = $props();

	$effect(() => {
		// let cancelled = false;
		// highlightHtml(raw).then((processed) => {
		// 	if (!cancelled) renderedHtml = processed;
		// });
		// return () => {
		// 	cancelled = true;
		// };
	});
</script>

<Sheet.Root bind:open={docsStore.isOpen}>
	<Sheet.Content side="right" class="flex w-105 flex-col gap-0 p-0 sm:max-w-105">
		<Content />
	</Sheet.Content>
</Sheet.Root>
