<script lang="ts">
	import { untrack, getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { DeviceTarget, CommandResponseStatus } from '@mir/sdk';
	import { SvelteMap } from 'svelte/reactivity';
	import type { CommandGroup, CommandDescriptor, SendCommandResult } from '@mir/sdk';
	import {
		DescriptorPanel,
		JsonPayloadEditor,
		ResponsePanel
	} from '$lib/domains/devices/components/commands';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import type { Descriptor } from '$lib/domains/devices/types/types';

	type Selection = {
		groupIdx: number;
		command: CommandDescriptor;
		editorContent: string;
	} | null;

	let commandGroups = $state<CommandGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let selection = $state<Selection>(null);
	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let response = $state<SendCommandResult | null>(null);

	// Plain JS var — not reactive, not tracked by effects
	let _lastLoadedIds = new Set<string>();

	$effect(() => {
		if (!mirStore.mir || selectionStore.activeCount === 0) return;
		const currentIds = new Set(selectionStore.activeDevices.map((d) => d.spec.deviceId));

		// Pure removal: every current ID was already loaded — filter in place, no network call
		const isPureRemoval =
			_lastLoadedIds.size > 0 && [...currentIds].every((id) => _lastLoadedIds.has(id));

		if (isPureRemoval) {
			untrack(() => {
				// Capture identity key of selected group before filtering
				const selGroupKey = selection
					? commandGroups[selection.groupIdx]?.ids
							.map((id) => id.id)
							.sort()
							.join(',')
					: null;

				commandGroups = commandGroups
					.map((g) => ({ ...g, ids: g.ids.filter((id) => currentIds.has(id.id)) }))
					.filter((g) => g.ids.length > 0);

				if (selGroupKey) {
					const newIdx = commandGroups.findIndex(
						(g) =>
							g.ids
								.map((id) => id.id)
								.sort()
								.join(',') === selGroupKey
					);
					if (newIdx === -1) {
						selection = null;
						response = null;
						sendError = null;
					} else {
						selection = { ...selection!, groupIdx: newIdx };
					}
				}
			});
			_lastLoadedIds = currentIds;
		} else {
			loadCommands();
		}
	});

	async function loadCommands(preserveSelection = false) {
		if (!mirStore.mir) return;
		isLoading = true;
		error = null;
		try {
			const allIds = selectionStore.activeDevices.map((d) => d.spec.deviceId);
			const target = new DeviceTarget({ ids: allIds });
			const groups = await mirStore.mir.client().listCommands().request(target);
			commandGroups = groups;
			_lastLoadedIds = new Set(allIds);
			if (!preserveSelection) {
				selection = null;
				response = null;
				sendError = null;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
		}
	}

	const multiCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('multi');
	multiCtx.setTabRefresh(() => loadCommands(true));
	onDestroy(() => multiCtx.setTabRefresh(null));

	let groups = $derived(
		commandGroups.map((g) => ({
			label: g.ids.map((id) => `${id.name}/${id.namespace}`).join(', '),
			items: g.descriptors.map((d) => ({
				name: d.name,
				labels: d.labels,
				template: d.template,
				error: d.error
			})),
			errors: g.error ? [g.error] : []
		}))
	);

	let selectedKey = $derived(selection ? `${selection.groupIdx}:${selection.command.name}` : null);

	let deviceValues = $derived.by(() => {
		if (!selection) return undefined;
		const group = commandGroups[selection.groupIdx];
		const template = prettyJson(selection.command.template || '{}');
		return group.ids.map((id) => ({
			label: `${id.name}/${id.namespace}`,
			deviceId: id.id,
			values: template
		}));
	});

	function prettyJson(raw: string): string {
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
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

	function selectCommand(groupIdx: number, desc: Descriptor) {
		const full = commandGroups[groupIdx].descriptors.find((d) => d.name === desc.name);
		if (!full) return;
		selection = { groupIdx, command: full, editorContent: prettyJson(full.template || '{}') };
		response = null;
		sendError = null;
	}

	async function sendCommand(dryRun: boolean, text: string) {
		if (!mirStore.mir || !selection) return;
		isSending = true;
		sendError = null;
		try {
			const ids = commandGroups[selection.groupIdx].ids.map((id) => id.id);
			const target = new DeviceTarget({ ids });
			const result = await mirStore.mir
				.client()
				.sendCommand()
				.request(target, selection.command.name, text, dryRun);
			response = result;
			activityStore.add({
				kind: 'success',
				category: 'Command',
				title: selection.command.name,
				request: { ids, name: selection.command.name, payload: text, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send command';
			sendError = message;
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: selection.command.name,
				request: { name: selection.command.name },
				error: message
			});
		} finally {
			isSending = false;
		}
	}

	async function sendCommandMulti(dryRun: boolean, payloads: Map<string, string>) {
		if (!mirStore.mir || !selection) return;
		isSending = true;
		sendError = null;
		const merged = new SvelteMap<string, unknown>();
		const failures: string[] = [];

		for (const [deviceId, json] of payloads) {
			try {
				const target = new DeviceTarget({ ids: [deviceId] });
				const result = await mirStore.mir
					.client()
					.sendCommand()
					.request(target, selection.command.name, json, dryRun);
				for (const [k, v] of result) merged.set(k, v);
			} catch (err) {
				const label = deviceValues?.find((dv) => dv.deviceId === deviceId)?.label ?? deviceId;
				failures.push(`${label}: ${err instanceof Error ? err.message : 'failed'}`);
			}
		}

		if (merged.size > 0) {
			response = merged as SendCommandResult;
			activityStore.add({
				kind: 'success',
				category: 'Command',
				title: selection.command.name,
				request: { name: selection.command.name, dryRun },
				response: Object.fromEntries(merged)
			});
		}
		if (failures.length > 0) {
			sendError = failures.join('; ');
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: selection.command.name,
				request: { name: selection.command.name },
				error: sendError
			});
		}

		isSending = false;
	}
</script>

<div class="-m-4 flex h-[calc(100svh-13rem)] overflow-hidden rounded-none border-y">
	<DescriptorPanel
		title="Commands"
		items={[]}
		{groups}
		{isLoading}
		{error}
		groupErrors={[]}
		{selectedKey}
		emptyText="No commands found."
		onSelect={() => {}}
		onSelectGrouped={selectCommand}
	/>

	<div class="flex min-w-0 flex-1 overflow-hidden">
		{#if !selection}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<TerminalIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a command to get started</p>
			</div>
		{:else}
			<JsonPayloadEditor
				name={selection.command.name}
				nameError={selection.command.error}
				value={selection.editorContent}
				hasResponse={true}
				{isSending}
				{sendError}
				onSend={sendCommand}
				{deviceValues}
				onSendMulti={sendCommandMulti}
			/>
			<ResponsePanel
				{response}
				{statusLabel}
				{statusClass}
				onClear={() => {
					response = null;
					sendError = null;
				}}
			/>
		{/if}
	</div>
</div>
