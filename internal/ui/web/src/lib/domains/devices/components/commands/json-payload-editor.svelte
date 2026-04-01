<script lang="ts">
	import { untrack } from 'svelte';
	import type { Snippet } from 'svelte';
	import { EditorView, basicSetup } from 'codemirror';
	import { Compartment } from '@codemirror/state';
	import { json as jsonLang } from '@codemirror/lang-json';
	import { vim, Vim } from '@replit/codemirror-vim';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { themeStore } from '$lib/shared/stores/theme.svelte';
	import { cmTheme } from '$lib/shared/stores/codemirror-themes';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import SendIcon from '@lucide/svelte/icons/send';
	import FlaskConicalIcon from '@lucide/svelte/icons/flask-conical';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { SvelteMap } from 'svelte/reactivity';
	import * as Popover from '$lib/shared/components/shadcn/popover';

	type DeviceValue = { label: string; deviceId: string; values: string };

	let {
		name,
		nameError,
		value,
		hasResponse,
		isSending,
		sendError,
		showDryRun = true,
		onSend,
		deviceValues,
		onSendMulti,
		headerEnd,
		footerEnd
	}: {
		name: string;
		nameError?: string;
		value: string;
		hasResponse: boolean;
		isSending: boolean;
		sendError: string | null;
		showDryRun?: boolean;
		onSend: (dryRun: boolean, text: string) => void;
		deviceValues?: DeviceValue[];
		onSendMulti?: (dryRun: boolean, payloads: Map<string, string>) => void;
		headerEnd?: Snippet;
		footerEnd?: Snippet;
	} = $props();

	let viewMode = $state<'template' | 'per-device'>('template');
	let isMultiValues = $derived(viewMode === 'per-device' && (deviceValues?.length ?? 0) > 0);

	let displayValue = $derived.by(() => {
		if (viewMode === 'per-device' && deviceValues && deviceValues.length > 0) {
			return deviceValues.map((dv) => `// ${dv.label}\n${dv.values}`).join('\n\n');
		}
		return value;
	});

	let isVimMode = $derived(editorPrefs.vim);
	let copied = $state(false);
	let localError = $state<string | null>(null);
	let headerWidth = $state(9999);
	const compact = $derived(headerWidth < 300);
	let overflowOpen = $state(false);

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
			const payloads = new SvelteMap<string, string>();
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
				themeCompartment.of(cmTheme(untrack(() => themeStore.current)))
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
			cmView.dispatch({
				effects: themeCompartment.reconfigure(cmTheme(t))
			});
		}
	});

	let displayError = $derived(sendError ?? localError);
</script>

<div class="flex flex-1 flex-col overflow-hidden {hasResponse ? 'border-r' : ''}">
	<!-- Header: name + error badge + vim toggle + copy -->
	<div class="flex items-center gap-2 border-b px-4 py-2" bind:clientWidth={headerWidth}>
		<span class="min-w-0 truncate font-mono text-sm font-medium">{name}</span>
		{#if nameError}
			<Badge variant="destructive" class="shrink-0 text-xs">{nameError}</Badge>
		{/if}
		<div class="ml-auto flex shrink-0 items-center gap-1">
			{#if !compact}
				{#if deviceValues && deviceValues.length > 0}
					<button
						onclick={() => (viewMode = viewMode === 'per-device' ? 'template' : 'per-device')}
						class="rounded px-2 py-0.5 font-mono text-[10px] transition-colors {viewMode ===
						'per-device'
							? 'bg-secondary text-secondary-foreground'
							: 'text-muted-foreground hover:text-foreground'}"
					>
						{viewMode === 'per-device' ? 'PER DEVICE' : 'TEMPLATE'}
					</button>
				{/if}
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
				{@render headerEnd?.()}
			{:else}
				<!-- Compact: collapse into popover row (same as tlm-toolbar) -->
				<Popover.Root bind:open={overflowOpen}>
					<Popover.Trigger>
						{#snippet child({ props })}
							<button
								{...props}
								title="More options"
								class="flex items-center rounded-md border border-border bg-background px-1.5 py-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground {overflowOpen
									? 'border-ring ring-1 ring-ring'
									: ''}"
							>
								<ChevronDownIcon class="size-3.5" />
							</button>
						{/snippet}
					</Popover.Trigger>
					<Popover.Content class="w-auto p-1.5 shadow-lg" align="end">
						<div class="flex items-center gap-1">
							{#if deviceValues && deviceValues.length > 0}
								<button
									onclick={() => {
										viewMode = viewMode === 'per-device' ? 'template' : 'per-device';
										overflowOpen = false;
									}}
									class="flex items-center gap-1 rounded-md border border-border bg-background px-1.5 py-1 font-mono text-[10px] text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground {viewMode === 'per-device' ? 'border-ring text-foreground ring-1 ring-ring' : ''}"
								>
									{viewMode === 'per-device' ? 'PER DEVICE' : 'TEMPLATE'}
								</button>
							{/if}
							<button
								onclick={() => { toggleVim(); overflowOpen = false; }}
								class="flex items-center gap-1 rounded-md border border-border bg-background px-1.5 py-1 font-mono text-[10px] text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground {isVimMode ? 'border-ring text-foreground ring-1 ring-ring' : ''}"
							>
								VIM
							</button>
							<button
								onclick={() => { handleCopy(); overflowOpen = false; }}
								class="flex items-center gap-1 rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
								title="Copy"
							>
								{#if copied}
									<CheckIcon class="size-3.5 text-emerald-500" />
								{:else}
									<CopyIcon class="size-3.5" />
								{/if}
							</button>
							{@render headerEnd?.()}
						</div>
					</Popover.Content>
				</Popover.Root>
			{/if}
		</div>
	</div>

	<!-- CodeMirror editor -->
	<div
		bind:this={cmEl}
		class="flex-1 overflow-auto [&_.cm-editor]:h-full [&_.cm-editor]:outline-none [&_.cm-scroller]:font-mono [&_.cm-scroller]:text-xs"
	></div>

	<!-- Action buttons -->
	<div class="flex items-center gap-1 border-t px-3 py-0.5">
		<Button
			variant="ghost"
			size="sm"
			disabled={isSending}
			onclick={() => submit(false)}
			class="h-7 gap-1 px-2 text-xs"
		>
			{#if isSending}
				<Spinner class="size-3" />
			{:else}
				<SendIcon class="size-3" />
			{/if}
			Send
		</Button>
		{#if showDryRun}
			<Button
				variant="ghost"
				size="sm"
				disabled={isSending}
				onclick={() => submit(true)}
				class="h-7 gap-1 px-2 text-xs text-muted-foreground"
			>
				{#if isSending}
					<Spinner class="size-3" />
				{:else}
					<FlaskConicalIcon class="size-3" />
				{/if}
				Dry Run
			</Button>
		{/if}
		{#if displayError}
			<p class="ml-2 text-xs text-destructive">{displayError}</p>
		{/if}
		{@render footerEnd?.()}
	</div>
</div>
