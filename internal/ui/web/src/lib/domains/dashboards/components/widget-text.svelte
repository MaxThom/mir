<script lang="ts">
	import { untrack } from 'svelte';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import type { TextWidgetConfig } from '../api/dashboard-api';

	let { widgetId, config }: { widgetId: string; config: TextWidgetConfig } = $props();

	const editMode = $derived(dashboardStore.editMode || dashboardStore.isCreatingNew);
	const urlMode  = $derived(!!config.url);

	// Static mode state
	let localContent = $state('');

	$effect.pre(() => {
		const incoming = config.content ?? '';
		if (incoming !== untrack(() => localContent)) {
			localContent = incoming;
		}
	});

	let saveTimer: ReturnType<typeof setTimeout> | null = null;

	function flushSave() {
		if (saveTimer) {
			clearTimeout(saveTimer);
			saveTimer = null;
		}
		if (!dashboardStore.activeDashboard) return;
		dashboardStore.saveWidgetConfig(dashboardStore.activeDashboard, widgetId, {
			content: localContent,
			url: config.url,
			jsonKey: config.jsonKey
		} satisfies TextWidgetConfig);
	}

	function onInput(e: Event) {
		localContent = (e.currentTarget as HTMLTextAreaElement).value;
		if (saveTimer) clearTimeout(saveTimer);
		saveTimer = setTimeout(flushSave, 600);
	}

	$effect(() => {
		if (!editMode && saveTimer) flushSave();
	});

	// URL mode state
	let fetchedHtml            = $state(''); // plain escaped fallback
	let fetchedHighlightedHtml = $state(''); // Shiki output
	let fetchedIsJson          = $state(false);
	let fetchError             = $state('');
	let fetching               = $state(false);

	function externalLinks(html: string): string {
		return html.replace(/<a\s(?![^>]*\btarget=)/g, '<a target="_blank" rel="noopener noreferrer" ');
	}

	function processContent(text: string): { html: string; isJson: boolean; rawJson?: string } {
		const trimmed = text.trim();
		if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
			try {
				const rawJson = JSON.stringify(JSON.parse(trimmed), null, 2);
				const escaped = rawJson.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
				return { html: escaped, isJson: true, rawJson };
			} catch { /* not valid JSON, fall through to markdown */ }
		}
		return { html: externalLinks(DOMPurify.sanitize(marked.parse(text) as string)), isJson: false };
	}

	async function highlightJson(code: string) {
		const hl = await getHighlighter();
		fetchedHighlightedHtml = hl.codeToHtml(code, {
			lang: 'json',
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
	}

	async function doFetch() {
		if (!config.url) return;
		fetching = true;
		fetchError = '';
		fetchedHighlightedHtml = '';
		try {
			const res = await fetch(config.url);
			if (!res.ok) throw new Error(`HTTP ${res.status}`);
			let raw: string;
			if (config.jsonKey) {
				const text = await res.text();
				try {
					let current: unknown = JSON.parse(text);
					for (const part of config.jsonKey.split('.')) {
						if (current === null || typeof current !== 'object') { current = undefined; break; }
						current = (current as Record<string, unknown>)[part];
					}
					raw = typeof current === 'object' && current !== null
						? JSON.stringify(current)
						: String(current ?? '');
				} catch {
					raw = text;
				}
			} else {
				raw = await res.text();
			}
			const result = processContent(raw);
			fetchedHtml   = result.html;
			fetchedIsJson = result.isJson;
			if (result.isJson && result.rawJson) highlightJson(result.rawJson);
		} catch (e) {
			fetchError = (e as Error).message;
		} finally {
			fetching = false;
		}
	}

	$effect(() => {
		const url = config.url;
		const key = config.jsonKey;
		if (url) {
			untrack(() => doFetch());
		}
	});

	const rendered = $derived(externalLinks(DOMPurify.sanitize(marked.parse(localContent || '') as string)));

	const proseFontSize = $derived(
		config.fontSize === 'base' ? '13px'
		: config.fontSize === 'lg'  ? '15px'
		: config.fontSize === 'xl'  ? '18px'
		: '11px' // sm default
	);
</script>

<div class="flex h-full flex-col">
	<div class="mt-2.5 shrink-0 border-b"></div>

	{#if editMode}
		{#if urlMode}
			{#if fetching}
				<div class="flex flex-1 items-center justify-center gap-2 text-sm text-muted-foreground select-none opacity-60">
					<Spinner class="h-4 w-4" />
					<span>Fetching…</span>
				</div>
			{:else if fetchError}
				<div class="px-4 py-3 text-sm text-destructive select-none opacity-60">{fetchError}</div>
			{:else if fetchedIsJson}
				<div class="text-[11px] leading-relaxed [&_.shiki]:bg-transparent [&>pre]:px-4 [&>pre]:py-3 [&>pre]:break-all [&>pre]:whitespace-pre-wrap min-h-0 flex-1 overflow-y-auto select-none opacity-60">
					{#if fetchedHighlightedHtml}
						{@html fetchedHighlightedHtml}
					{:else}
						<pre class="px-4 py-3 break-all whitespace-pre-wrap">{@html fetchedHtml}</pre>
					{/if}
				</div>
			{:else}
				<div class="prose dark:prose-invert min-h-0 flex-1 max-w-none overflow-y-auto px-4 py-3 select-none opacity-60 [&_hr]:my-3" style="font-size: {proseFontSize}">
					{@html fetchedHtml}
				</div>
			{/if}
		{:else}
			<textarea
				class="min-h-0 flex-1 resize-none bg-transparent px-4 py-3 font-mono text-sm focus:outline-none"
				value={localContent}
				oninput={onInput}
				onkeydown={(e) => e.stopPropagation()}
				placeholder="Write markdown here…"
			></textarea>
		{/if}
	{:else}
		{#if urlMode}
			{#if fetching}
				<div class="flex flex-1 items-center justify-center gap-2 text-sm text-muted-foreground">
					<Spinner class="h-4 w-4" />
					<span>Fetching…</span>
				</div>
			{:else if fetchError}
				<div class="px-4 py-3 text-sm text-destructive">{fetchError}</div>
			{:else if fetchedIsJson}
				<div class="text-[11px] leading-relaxed [&_.shiki]:bg-transparent [&>pre]:px-4 [&>pre]:py-3 [&>pre]:break-all [&>pre]:whitespace-pre-wrap min-h-0 flex-1 overflow-y-auto">
					{#if fetchedHighlightedHtml}
						{@html fetchedHighlightedHtml}
					{:else}
						<pre class="px-4 py-3 break-all whitespace-pre-wrap">{@html fetchedHtml}</pre>
					{/if}
				</div>
			{:else}
				<div class="prose dark:prose-invert min-h-0 flex-1 max-w-none overflow-y-auto px-4 py-3 [&_hr]:my-3" style="font-size: {proseFontSize}">
					{@html fetchedHtml}
				</div>
			{/if}
		{:else}
			<div class="prose dark:prose-invert min-h-0 flex-1 max-w-none overflow-y-auto px-4 py-3 [&_hr]:my-3" style="font-size: {proseFontSize}">
				{@html rendered}
			</div>
		{/if}
	{/if}
</div>
