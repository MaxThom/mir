<script lang="ts">
	import { page } from '$app/state';
	import { untrack, getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { commandStore } from '$lib/domains/devices/stores/command.svelte';
	import { CommandResponseStatus } from '@mir/sdk';
	import type { CommandDescriptor, CommandResponse } from '@mir/sdk';
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
	import XIcon from '@lucide/svelte/icons/x';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SendIcon from '@lucide/svelte/icons/send';
	import FlaskConicalIcon from '@lucide/svelte/icons/flask-conical';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';
	import SearchIcon from '@lucide/svelte/icons/search';
	import { getHighlighter } from '$lib/shared/utils/highlighter';

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedCommand = $state<CommandDescriptor | null>(null);
	let editorContent = $state('{}');
	let copied = $state(false);
	let copiedResponseId = $state<string | null>(null);
	let searchQuery = $state('');
	let responseHtml = $state(new Map<string, string>());

	let cmEl = $state<HTMLElement | null>(null);
	let cmView: EditorView | null = null;
	const themeCompartment = new Compartment();
	const vimCompartment = new Compartment();

	let isVimMode = $derived(editorPrefs.vim);
	let allDescriptors = $derived(commandStore.commands.flatMap((g) => g.descriptors));
	let filteredDescriptors = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return allDescriptors;
		return allDescriptors.filter((desc) => {
			if (desc.name.toLowerCase().includes(q)) return true;
			return Object.entries(desc.labels ?? {}).some(
				([k, v]) => k.toLowerCase().includes(q) || v.toLowerCase().includes(q) || `${k}=${v}`.toLowerCase().includes(q)
			);
		});
	});

	const deviceCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('device');
	deviceCtx.setTabRefresh(() => {
		if (mirStore.mir && deviceId) {
			commandStore.loadCommands(mirStore.mir, deviceId);
		}
	});
	onDestroy(() => deviceCtx.setTabRefresh(null));

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedCommand = null;
			editorContent = '{}';
			commandStore.reset();
			commandStore.loadCommands(mirStore.mir, deviceId);
		}
	});

	function prettyJson(raw: string): string {
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	function selectCommand(desc: CommandDescriptor) {
		selectedCommand = desc;
		editorContent = prettyJson(desc.template || '{}');
		commandStore.reset();
	}

	function toggleVim() {
		editorPrefs.setVim(!isVimMode);
		cmView?.dispatch({ effects: vimCompartment.reconfigure(!isVimMode ? vim() : []) });
	}

	async function handleCopy() {
		const text = cmView ? cmView.state.doc.toString() : editorContent;
		await navigator.clipboard.writeText(text);
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 1500);
	}

	async function handleCopyResponse(devId: string, text: string) {
		await navigator.clipboard.writeText(text);
		copiedResponseId = devId;
		setTimeout(() => {
			copiedResponseId = null;
		}, 1500);
	}

	// Mount / recreate editor when cmEl or editorContent changes
	$effect(() => {
		if (!cmEl) return;
		const doc = editorContent;
		Vim.defineEx('write', 'w', () => handleSend(false));
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

	async function handleSend(dryRun: boolean) {
		if (!mirStore.mir || !selectedCommand) return;
		const text = cmView ? cmView.state.doc.toString() : editorContent;
		await commandStore.sendCommand(mirStore.mir, deviceId, selectedCommand.name, text, dryRun);
	}

	function statusLabel(status: CommandResponseStatus): string {
		switch (status) {
			case CommandResponseStatus.SUCCESS:
				return 'SUCCESS';
			case CommandResponseStatus.ERROR:
				return 'ERROR';
			case CommandResponseStatus.VALIDATED:
				return 'VALIDATED';
			case CommandResponseStatus.PENDING:
				return 'PENDING';
			default:
				return 'UNKNOWN';
		}
	}

	function statusClass(status: CommandResponseStatus): string {
		switch (status) {
			case CommandResponseStatus.SUCCESS:
				return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400';
			case CommandResponseStatus.ERROR:
				return 'bg-destructive/15 text-destructive';
			case CommandResponseStatus.VALIDATED:
				return 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function decodePayload(payload: Uint8Array): string {
		if (!payload || payload.length === 0) return '';
		try {
			return JSON.stringify(JSON.parse(new TextDecoder().decode(payload)), null, 2);
		} catch {
			return new TextDecoder().decode(payload);
		}
	}

	$effect(() => {
		const resp = commandStore.response;
		if (!resp) {
			responseHtml = new Map();
			return;
		}
		const entries = [...resp.entries()];
		getHighlighter().then((hl) => {
			const next = new Map<string, string>();
			for (const [devId, r] of entries) {
				const decoded = decodePayload(r.payload);
				if (decoded) {
					next.set(
						devId,
						hl.codeToHtml(decoded, {
							lang: 'json',
							themes: { light: 'github-light', dark: 'github-dark' },
							defaultColor: false
						})
					);
				}
			}
			responseHtml = next;
		});
	});
</script>

<div class="-m-4 flex min-h-[500px] overflow-hidden rounded-none border-y">
	<!-- Left panel: command list -->
	<div class="flex w-64 shrink-0 flex-col overflow-hidden border-r">
		<!-- Panel header -->
		<div class="flex items-center border-b px-3 py-[11px]">
			<span class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Commands</span>
		</div>

		<!-- Search -->
		<div class="flex items-center gap-2 border-b px-3 py-1.5">
			<SearchIcon class="size-3.5 shrink-0 text-muted-foreground" />
			<input
				bind:value={searchQuery}
				placeholder="name or label…"
				class="w-full bg-transparent py-1 text-xs outline-none placeholder:text-muted-foreground/60"
			/>
		</div>

		<!-- Loading / error / list -->
		<div class="flex-1 overflow-y-auto">
			{#if commandStore.isLoading && allDescriptors.length === 0}
				<div class="flex items-center gap-2 px-3 py-4 text-xs text-muted-foreground">
					<Spinner class="size-3" />
					Loading commands…
				</div>
			{:else if commandStore.error}
				<p class="px-3 py-4 text-xs text-destructive">{commandStore.error}</p>
			{:else if allDescriptors.length === 0}
				<p class="px-3 py-4 text-xs text-muted-foreground">No commands found.</p>
			{:else if filteredDescriptors.length === 0}
				<p class="px-3 py-4 text-xs text-muted-foreground">No match for "{searchQuery}".</p>
			{:else}
				{#each filteredDescriptors as desc (desc.name)}
					<button
						onclick={() => selectCommand(desc)}
						class="flex w-full flex-col gap-1.5 border-b px-3 py-2.5 text-left transition-colors last:border-0 hover:bg-accent
							{selectedCommand?.name === desc.name ? 'bg-accent' : ''}"
					>
						<span class="truncate font-mono text-xs font-medium">
							{#if desc.error}
								<span class="mr-1 inline-block size-1.5 translate-y-[-1px] rounded-full bg-destructive"></span>
							{/if}{desc.name}</span
						>
						<div class="flex flex-wrap gap-1">
							{#each Object.entries(desc.labels ?? {}) as [k, v] (k)}
								<span class="rounded-sm border border-border/60 bg-muted/60 px-1.5 py-px font-mono text-[11px] text-muted-foreground">
									{k}={v}
								</span>
							{/each}
						</div>
					</button>
				{/each}
			{/if}
		</div>
	</div>

	<!-- Right panel: editor + response side by side -->
	<div class="flex min-w-0 flex-1 overflow-hidden">
		{#if !selectedCommand}
			<!-- Empty state -->
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<TerminalIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a command to get started</p>
			</div>
		{:else}
			<!-- Editor panel -->
			<div class="flex flex-1 flex-col overflow-hidden {commandStore.response !== null ? 'border-r' : ''}">
				<!-- Command header + editor toolbar -->
				<div class="flex items-center gap-2 border-b px-4 py-2">
					<span class="font-mono text-sm font-medium">{selectedCommand.name}</span>
					{#if selectedCommand.error}
						<Badge variant="destructive" class="text-xs">{selectedCommand.error}</Badge>
					{/if}

					<!-- Toolbar: vim + copy -->
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
						disabled={commandStore.isSending}
						onclick={() => handleSend(false)}
						class="gap-1.5"
					>
						{#if commandStore.isSending}
							<Spinner class="size-3" />
						{:else}
							<SendIcon class="size-3.5" />
						{/if}
						Send
					</Button>
					<Button
						variant="ghost"
						size="sm"
						disabled={commandStore.isSending}
						onclick={() => handleSend(true)}
						class="gap-1.5 text-muted-foreground"
					>
						{#if commandStore.isSending}
							<Spinner class="size-3" />
						{:else}
							<FlaskConicalIcon class="size-3.5" />
						{/if}
						Dry Run
					</Button>
					{#if commandStore.sendError}
						<p class="ml-2 text-xs text-destructive">{commandStore.sendError}</p>
					{/if}
				</div>
			</div>

			<!-- Response panel (right side, when available) -->
			{#if commandStore.response !== null}
				<div class="flex flex-1 flex-col overflow-hidden">
					<!-- Response header with clear button -->
					<div class="flex items-center gap-2 border-b px-4 py-2">
						<span class="text-sm text-muted-foreground">Response</span>
						<button
							onclick={() => commandStore.reset()}
							class="ml-auto rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
							title="Clear response"
						>
							<XIcon class="size-3.5" />
						</button>
					</div>

					<!-- Response entries -->
					<div class="flex-1 overflow-y-auto p-4">
						<div class="flex flex-col gap-3">
							{#each [...commandStore.response.entries()] as [devId, resp] (devId)}
								<div class="rounded-lg border p-3">
									<div class="mb-2 flex items-center gap-2">
										<span class="font-mono text-xs text-muted-foreground">{devId}</span>
										<span
											class="rounded px-1.5 py-0.5 text-[10px] font-medium uppercase {statusClass(resp.status)}"
										>
											{statusLabel(resp.status)}
										</span>
									</div>
									{#if resp.error}
										<p class="rounded bg-destructive/10 px-2 py-1.5 text-xs text-destructive">
											{resp.error}
										</p>
									{/if}
									{#if resp.payload && resp.payload.length > 0}
										{@const decoded = decodePayload(resp.payload)}
										{#if decoded}
											<div class="relative mt-2 overflow-hidden rounded-md">
												{#if responseHtml.has(devId)}
													<!-- eslint-disable-next-line svelte/no-at-html-tags -->
													{@html responseHtml.get(devId)}
												{:else}
													<pre class="overflow-x-auto bg-muted px-3 py-2 font-mono text-xs leading-relaxed">{decoded}</pre>
												{/if}
												<button
													onclick={() => handleCopyResponse(devId, decoded)}
													class="absolute right-1.5 top-1.5 rounded p-1 text-muted-foreground/60 transition-colors hover:text-foreground"
													title="Copy"
												>
													{#if copiedResponseId === devId}
														<CheckIcon class="size-3 text-emerald-500" />
													{:else}
														<CopyIcon class="size-3" />
													{/if}
												</button>
											</div>
										{/if}
									{/if}
								</div>
							{/each}
						</div>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>
