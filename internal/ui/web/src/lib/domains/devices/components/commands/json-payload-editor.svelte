<script lang="ts">
	import { untrack } from 'svelte';
	import { EditorView, basicSetup } from 'codemirror';
	import { Compartment } from '@codemirror/state';
	import { json as jsonLang } from '@codemirror/lang-json';
	import { oneDark } from '@codemirror/theme-one-dark';
	import { vim, Vim } from '@replit/codemirror-vim';
	import { mode } from 'mode-watcher';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import SendIcon from '@lucide/svelte/icons/send';
	import FlaskConicalIcon from '@lucide/svelte/icons/flask-conical';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';

	let {
		name,
		nameError,
		value,
		hasResponse,
		isSending,
		sendError,
		showDryRun = true,
		onSend
	}: {
		name: string;
		nameError?: string;
		value: string;
		hasResponse: boolean;
		isSending: boolean;
		sendError: string | null;
		showDryRun?: boolean;
		onSend: (dryRun: boolean, text: string) => void;
	} = $props();

	let isVimMode = $derived(editorPrefs.vim);
	let copied = $state(false);
	let localError = $state<string | null>(null);

	let cmEl = $state<HTMLElement | null>(null);
	let cmView: EditorView | null = null;
	const themeCompartment = new Compartment();
	const vimCompartment = new Compartment();

	Vim.defineEx('write', 'w', () => submit(false));

	function toggleVim() {
		editorPrefs.setVim(!isVimMode);
		cmView?.dispatch({ effects: vimCompartment.reconfigure(!isVimMode ? vim() : []) });
	}

	async function handleCopy() {
		const text = cmView ? cmView.state.doc.toString() : value;
		try {
			await navigator.clipboard.writeText(text);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			// clipboard unavailable (non-HTTPS or permission denied)
		}
	}

	function submit(dryRun: boolean) {
		const text = cmView ? cmView.state.doc.toString() : value;
		try {
			JSON.parse(text);
		} catch {
			localError = 'Invalid JSON payload';
			return;
		}
		localError = null;
		onSend(dryRun, text);
	}

	// Mount / recreate editor when cmEl or value changes
	$effect(() => {
		if (!cmEl) return;
		const doc = value;
		const view = new EditorView({
			doc,
			extensions: [
				vimCompartment.of(untrack(() => editorPrefs.vim) ? vim() : []),
				basicSetup,
				jsonLang(),
				themeCompartment.of(untrack(() => mode.current === 'dark') ? oneDark : [])
			],
			parent: cmEl
		});
		cmView = view;
		return () => {
			view.destroy();
			cmView = null;
		};
	});

	// Update theme without recreating the view
	$effect(() => {
		const isDark = mode.current === 'dark';
		if (cmView) {
			cmView.dispatch({ effects: themeCompartment.reconfigure(isDark ? oneDark : []) });
		}
	});

	let displayError = $derived(sendError ?? localError);
</script>

<div class="flex flex-1 flex-col overflow-hidden {hasResponse ? 'border-r' : ''}">
	<!-- Header: name + error badge + vim toggle + copy -->
	<div class="flex items-center gap-2 border-b px-4 py-2">
		<span class="font-mono text-sm font-medium">{name}</span>
		{#if nameError}
			<Badge variant="destructive" class="text-xs">{nameError}</Badge>
		{/if}
		<div class="ml-auto flex items-center gap-1">
			<button
				onclick={toggleVim}
				class="rounded px-2 py-0.5 font-mono text-[10px] transition-colors {isVimMode
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			>
				VIM
			</button>
			<button
				onclick={handleCopy}
				class="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
				title="Copy"
			>
				{#if copied}
					<CheckIcon class="size-3.5 text-emerald-500" />
				{:else}
					<CopyIcon class="size-3.5" />
				{/if}
			</button>
		</div>
	</div>

	<!-- CodeMirror editor -->
	<div
		bind:this={cmEl}
		class="flex-1 overflow-auto [&_.cm-editor]:h-full [&_.cm-editor]:outline-none [&_.cm-scroller]:font-mono [&_.cm-scroller]:text-xs"
	></div>

	<!-- Action buttons -->
	<div class="flex items-center gap-1 border-t px-4 py-2">
		<Button
			variant="ghost"
			size="sm"
			disabled={isSending}
			onclick={() => submit(false)}
			class="gap-1.5"
		>
			{#if isSending}
				<Spinner class="size-3" />
			{:else}
				<SendIcon class="size-3.5" />
			{/if}
			Send
		</Button>
		{#if showDryRun}
			<Button
				variant="ghost"
				size="sm"
				disabled={isSending}
				onclick={() => submit(true)}
				class="gap-1.5 text-muted-foreground"
			>
				{#if isSending}
					<Spinner class="size-3" />
				{:else}
					<FlaskConicalIcon class="size-3.5" />
				{/if}
				Dry Run
			</Button>
		{/if}
		{#if displayError}
			<p class="ml-2 text-xs text-destructive">{displayError}</p>
		{/if}
	</div>
</div>
