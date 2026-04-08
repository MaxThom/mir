<script lang="ts">
	import { untrack } from 'svelte';
	import { marked } from 'marked';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import type { TextWidgetConfig } from '../api/dashboard-api';

	let { widgetId, config }: { widgetId: string; config: TextWidgetConfig } = $props();

	const editMode = $derived(dashboardStore.editMode || dashboardStore.isCreatingNew);

	let localContent = $state('');

	// Sync from config when it changes externally, but don't overwrite in-flight edits.
	// $effect.pre runs before DOM update so the initial value is set before first render.
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
			content: localContent
		} satisfies TextWidgetConfig);
	}

	function onInput(e: Event) {
		localContent = (e.currentTarget as HTMLTextAreaElement).value;
		if (saveTimer) clearTimeout(saveTimer);
		saveTimer = setTimeout(flushSave, 600);
	}

	// Flush any pending save when leaving edit mode
	$effect(() => {
		if (!editMode && saveTimer) flushSave();
	});

	const rendered = $derived(marked.parse(localContent || '') as string);
</script>

<div class="flex h-full flex-col">
	<div class="mt-2.5 shrink-0 border-b"></div>

	{#if editMode}
		<textarea
			class="min-h-0 flex-1 resize-none bg-transparent px-4 py-3 font-mono text-sm focus:outline-none"
			value={localContent}
			oninput={onInput}
			onkeydown={(e) => e.stopPropagation()}
			placeholder="Write markdown here…"
		></textarea>
	{:else}
		<div class="prose prose-sm dark:prose-invert min-h-0 flex-1 max-w-none overflow-y-auto px-4 py-3">
			{@html rendered}
		</div>
	{/if}
</div>
