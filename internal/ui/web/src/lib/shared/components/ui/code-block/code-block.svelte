<script lang="ts">
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import { Copy, Check } from '@lucide/svelte';

	let { title, code, lang }: { title?: string; code: string; lang: 'go' | 'bash' | 'typescript' } =
		$props();

	let highlighted = $state('');
	let copied = $state(false);

	const langLabels: Record<string, string> = {
		go: 'Go',
		bash: 'Bash',
		typescript: 'TypeScript'
	};

	$effect(() => {
		const currentCode = code;
		const currentLang = lang;
		getHighlighter().then((hl) => {
			highlighted = hl.codeToHtml(currentCode, {
				lang: currentLang,
				themes: {
					light: 'github-light',
					dark: 'github-dark'
				},
				defaultColor: false
			});
		});
	});

	function copyCode() {
		navigator.clipboard.writeText(code).then(() => {
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		});
	}
</script>

<div class="overflow-hidden rounded-md border border-border">
	<div class="flex items-center justify-between border-b border-border bg-muted px-3 py-1.5">
		<span class="text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
			{title}
		</span>
		<div class="flex items-center gap-2">
			<span class="text-[10px] font-semibold tracking-wide text-muted-foreground uppercase">
				{langLabels[lang] ?? lang}
			</span>
			<button
				onclick={copyCode}
				class="text-muted-foreground transition-colors hover:text-foreground"
				aria-label="Copy code"
			>
				{#if copied}
					<Check class="size-3.5 text-green-500" />
				{:else}
					<Copy class="size-3.5" />
				{/if}
			</button>
		</div>
	</div>
	{#if highlighted}
		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html highlighted}
	{:else}
		<pre class="overflow-x-auto bg-muted px-3 py-2 font-mono text-xs">{code}</pre>
	{/if}
</div>
