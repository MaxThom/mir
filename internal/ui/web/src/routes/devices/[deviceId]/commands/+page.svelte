<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { commandStore } from '$lib/domains/devices/stores/command.svelte';
	import { CommandResponseStatus } from '@mir/sdk';
	import type { CommandDescriptor } from '@mir/sdk';
	import {
		DescriptorPanel,
		JsonPayloadEditor,
		ResponsePanel
	} from '$lib/domains/devices/components/commands';
	import TerminalIcon from '@lucide/svelte/icons/terminal';

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedCommand = $state<CommandDescriptor | null>(null);
	let editorContent = $state('{}');

	let allDescriptors = $derived(commandStore.commands.flatMap((g) => g.descriptors));
	let groupErrors = $derived(commandStore.commands.map((g) => g.error).filter(Boolean));

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
		commandStore.clearResponse();
	}

	function handleSend(dryRun: boolean, text: string) {
		if (!mirStore.mir || !selectedCommand) return;
		commandStore.sendCommand(mirStore.mir, deviceId, selectedCommand.name, text, dryRun);
	}

	function statusLabel(status: number): string {
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

	function statusClass(status: number): string {
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
</script>

<div class="-m-4 flex h-[calc(100svh-14rem)] overflow-hidden rounded-none border-y">
	<DescriptorPanel
		title="Commands"
		items={allDescriptors}
		isLoading={commandStore.isLoading}
		error={commandStore.error}
		{groupErrors}
		selectedName={selectedCommand?.name ?? null}
		emptyText="No commands found."
		onSelect={selectCommand}
	/>

	<div class="flex min-w-0 flex-1 overflow-hidden">
		{#if !selectedCommand}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<TerminalIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a command to get started</p>
			</div>
		{:else}
			<JsonPayloadEditor
				name={selectedCommand.name}
				nameError={selectedCommand.error}
				value={editorContent}
				hasResponse={true}
				isSending={commandStore.isSending}
				sendError={commandStore.sendError}
				onSend={handleSend}
			/>
			<ResponsePanel
				response={commandStore.response}
				{statusLabel}
				{statusClass}
				onClear={() => commandStore.clearResponse()}
			/>
		{/if}
	</div>
</div>
