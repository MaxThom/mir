<script lang="ts">
	import { EditorView, basicSetup } from 'codemirror';
	import { Compartment } from '@codemirror/state';
	import { yaml as yamlLang } from '@codemirror/lang-yaml';
	import { json as jsonLang } from '@codemirror/lang-json';
	import { vim, Vim } from '@replit/codemirror-vim';
	import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
	import { untrack } from 'svelte';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XIcon from '@lucide/svelte/icons/x';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { themeStore } from '$lib/shared/stores/theme.svelte';
	import { cmTheme } from '$lib/shared/stores/codemirror-themes';

	let {
		content,
		onSave,
		onCancel,
		isSaving = false,
		error = null
	}: {
		content: string;
		onSave: (text: string) => Promise<void>;
		onCancel: () => void;
		isSaving?: boolean;
		error?: string | null;
	} = $props();

	let isVimMode = $derived(editorPrefs.vim);
	let isJsonMode = $derived(editorPrefs.json);

	let cmEditorEl = $state<HTMLElement | null>(null);
	let cmView: EditorView | null = null;
	const vimCompartment = new Compartment();
	const langCompartment = new Compartment();
	const themeCompartment = new Compartment();

	// Internal content state — seeded from prop, updated on format toggle
	let localContent = $state(untrack(() => content));

	function toggleVim() {
		editorPrefs.setVim(!isVimMode);
		cmView?.dispatch({ effects: vimCompartment.reconfigure(!isVimMode ? vim() : []) });
	}

	function toggleFormat() {
		if (!cmView) return;
		const text = cmView.state.doc.toString();
		try {
			localContent = isJsonMode
				? stringifyYaml(JSON.parse(text), { lineWidth: 0 })
				: JSON.stringify(parseYaml(text), null, 2);
		} catch {
			localContent = text;
		}
		editorPrefs.setJson(!isJsonMode);
	}

	async function handleSave() {
		const text = cmView ? cmView.state.doc.toString() : localContent;
		await onSave(text);
	}

	$effect(() => {
		if (cmEditorEl) {
			Vim.defineEx('write', 'w', () => handleSave());
			Vim.defineEx('quit', 'q', () => onCancel());
			const view = new EditorView({
				doc: localContent,
				extensions: [
					vimCompartment.of(isVimMode ? vim() : []),
					langCompartment.of(isJsonMode ? jsonLang() : yamlLang()),
					themeCompartment.of(cmTheme(themeStore.current)),
					basicSetup
				],
				parent: cmEditorEl
			});
			cmView = view;
			view.focus();
			return () => {
				view.destroy();
				cmView = null;
			};
		}
	});

	$effect(() => {
		const t = themeStore.current;
		if (cmView) {
			cmView.dispatch({ effects: themeCompartment.reconfigure(cmTheme(t)) });
		}
	});
</script>

<div class="flex items-center justify-between">
	<div class="flex items-center gap-2">
		<div class="flex overflow-hidden rounded border border-input font-mono text-[10px]">
			<button
				onclick={() => isJsonMode && toggleFormat()}
				class="px-2 py-0.5 transition-colors {!isJsonMode
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}">YAML</button
			>
			<button
				onclick={() => !isJsonMode && toggleFormat()}
				class="px-2 py-0.5 transition-colors {isJsonMode
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}">JSON</button
			>
		</div>
		<button
			onclick={toggleVim}
			class="rounded px-2 py-0.5 font-mono text-[10px] transition-colors {isVimMode
				? 'bg-secondary text-secondary-foreground'
				: 'text-muted-foreground hover:text-foreground'}"
		>
			VIM
		</button>
	</div>
	<div class="flex items-center gap-1">
		<Button
			variant="ghost"
			size="sm"
			onclick={handleSave}
			disabled={isSaving}
			class="h-7 gap-1 text-xs"
		>
			{#if isSaving}<Spinner class="size-3" />{:else}<CheckIcon class="size-3" />{/if}
			Save
		</Button>
		<Button
			variant="ghost"
			size="sm"
			onclick={onCancel}
			disabled={isSaving}
			class="h-7 gap-1 text-xs"
		>
			<XIcon class="size-3" />
			Cancel
		</Button>
	</div>
</div>


{#if error}
	<p class="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">{error}</p>
{/if}

<div
	bind:this={cmEditorEl}
	class="overflow-hidden rounded-md border border-input [&_.cm-editor]:min-h-64 [&_.cm-editor]:outline-none [&_.cm-scroller]:font-mono [&_.cm-scroller]:text-xs"
></div>
