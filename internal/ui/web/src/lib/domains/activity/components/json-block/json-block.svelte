<script lang="ts">
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';

	let { value }: { value: unknown } = $props();

	let highlighted = $state('');
	let copied = $state(false);
	let code = $derived(JSON.stringify(value, null, 2));

	$effect(() => {
		const current = code;
		getHighlighter().then((hl) => {
			highlighted = hl.codeToHtml(current, {
				lang: 'json',
				themes: { light: 'github-light', dark: 'github-dark' },
				defaultColor: false
			});
		});
	});

	function copy() {
		navigator.clipboard.writeText(code).then(() => {
			copied = true;
			setTimeout(() => (copied = false), 2000);
		});
	}
</script>

<div class="group relative max-h-40 overflow-auto rounded [&>pre]:!m-0 [&>pre]:!rounded [&>pre]:!p-2 [&>pre]:!text-[10px]">
	<button
		onclick={copy}
		aria-label="Copy"
		class="absolute top-1.5 right-1.5 z-10 rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-foreground"
	>
		{#if copied}
			<CheckIcon class="size-3 text-emerald-500" />
		{:else}
			<CopyIcon class="size-3" />
		{/if}
	</button>
	{#if highlighted}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html highlighted}
	{:else}
		<pre class="rounded bg-muted p-2 font-mono text-[10px] whitespace-pre-wrap break-all">{code}</pre>
	{/if}
</div>
