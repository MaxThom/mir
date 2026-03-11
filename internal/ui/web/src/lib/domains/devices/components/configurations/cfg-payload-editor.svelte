<script lang="ts">
	import { untrack } from 'svelte';
	import { EditorView, basicSetup } from 'codemirror';
	import { Compartment } from '@codemirror/state';
	import { json as jsonLang } from '@codemirror/lang-json';
	import { oneDark } from '@codemirror/theme-one-dark';
	import { vim, Vim } from '@replit/codemirror-vim';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { themeStore } from '$lib/shared/stores/theme.svelte';
	import { rustTheme, midnightTheme } from '$lib/shared/stores/codemirror-themes';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import SendIcon from '@lucide/svelte/icons/send';
	import FlaskConicalIcon from '@lucide/svelte/icons/flask-conical';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';

	type DeviceValue = { label: string; deviceId: string; values: string };

	let {
		name,
		nameError,
		currentValues = '{}',
		template,
		hasResponse,
		isSending,
		sendError,
		showDryRun = true,
		onSend,
		deviceValues,
		onSendMulti
	}: {
		name: string;
		nameError?: string;
		currentValues?: string;
		template: string;
		hasResponse: boolean;
		isSending: boolean;
		sendError: string | null;
		showDryRun?: boolean;
		onSend: (dryRun: boolean, text: string) => void;
		deviceValues?: DeviceValue[];
		onSendMulti?: (dryRun: boolean, payloads: Map<string, string>) => void;
	} = $props();

	let viewMode = $state<'values' | 'template'>('values');

	let isMultiValues = $derived(viewMode === 'values' && (deviceValues?.length ?? 0) > 0);

	let displayValue = $derived.by(() => {
		if (viewMode === 'template') return template;
		if (deviceValues && deviceValues.length > 0) {
			return deviceValues.map((dv) => `// ${dv.label}\n${dv.values}`).join('\n\n');
		}
		return currentValues;
	});

	let isVimMode = $derived(editorPrefs.vim);
	let copied = $state(false);
	let localError = $state<string | null>(null);

	let cmEl = $state<HTMLElement | null>(null);
	let cmView: EditorView | null = null;
	const themeCompartment = new Compartment();
	const vimCompartment = new Compartment();

	Vim.defineEx('write', 'w', () => submit(false));
	Vim.defineEx('wq', 'wq', () => submit(false));

	function toggleVim() {
		const newVim = !isVimMode;
		editorPrefs.setVim(newVim);
		cmView?.dispatch({ effects: vimCompartment.reconfigure(newVim ? vim() : []) });
	}

	async function handleCopy() {
		const text = cmView ? cmView.state.doc.toString() : displayValue;
		try {
			await navigator.clipboard.writeText(text);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {
			// clipboard unavailable (non-HTTPS or permission denied)
		}
	}

	function submit(dryRun: boolean) {
		const text = cmView ? cmView.state.doc.toString() : displayValue;

		if (isMultiValues && onSendMulti) {
			const payloads = new Map<string, string>();
			const errors: string[] = [];
			const blocks = text.split(/\n(?=\/\/ )/);
			for (const block of blocks) {
				const nlIdx = block.indexOf('\n');
				if (nlIdx === -1) continue;
				const labelLine = block.slice(0, nlIdx).trim();
				const label = labelLine.startsWith('// ') ? labelLine.slice(3) : labelLine;
				const jsonText = block.slice(nlIdx + 1).trim();
				try {
					JSON.parse(jsonText);
				} catch {
					errors.push(`Invalid JSON for ${label}`);
					continue;
				}
				const dv = deviceValues!.find((d) => d.label === label);
				if (dv) payloads.set(dv.deviceId, jsonText);
			}
			if (errors.length > 0) {
				localError = errors.join('; ');
				return;
			}
			localError = null;
			onSendMulti(dryRun, payloads);
			return;
		}

		try {
			JSON.parse(text);
		} catch {
			localError = 'Invalid JSON payload';
			return;
		}
		localError = null;
		onSend(dryRun, text);
	}

	// Mount / recreate editor when cmEl or displayValue changes
	$effect(() => {
		if (!cmEl) return;
		const doc = displayValue;
		const view = new EditorView({
			doc,
			extensions: [
				vimCompartment.of(untrack(() => editorPrefs.vim) ? vim() : []),
				basicSetup,
				jsonLang(),
				themeCompartment.of(untrack(() => themeStore.current === 'dark') ? oneDark : untrack(() => themeStore.current === 'midnight') ? midnightTheme : untrack(() => themeStore.current === 'rust') ? rustTheme : [])
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
		const t = themeStore.current;
		if (cmView) {
			cmView.dispatch({ effects: themeCompartment.reconfigure(t === 'dark' ? oneDark : t === 'midnight' ? midnightTheme : t === 'rust' ? rustTheme : []) });
		}
	});

	let displayError = $derived(sendError ?? localError);
</script>

<div class="flex flex-1 flex-col overflow-hidden {hasResponse ? 'border-r' : ''}">
	<!-- Header: name + error badge + TPL toggle + vim toggle + copy -->
	<div class="flex items-center gap-2 border-b px-4 py-2">
		<span class="font-mono text-sm font-medium">{name}</span>
		{#if nameError}
			<Badge variant="destructive" class="text-xs">{nameError}</Badge>
		{/if}
		<div class="ml-auto flex items-center gap-1">
			<button
				onclick={() => (viewMode = viewMode === 'values' ? 'template' : 'values')}
				class="rounded px-2 py-0.5 font-mono text-[10px] transition-colors {viewMode === 'template'
					? 'bg-secondary text-secondary-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			>
				{viewMode === 'template' ? 'TEMPLATE' : isMultiValues ? 'VALUES (per device)' : 'VALUES'}
			</button>
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
