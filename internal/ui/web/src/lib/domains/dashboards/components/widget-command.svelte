<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import {
		DescriptorPanel,
		JsonPayloadEditor
	} from '$lib/domains/devices/components/commands';
	import { DeviceTarget } from '@mir/sdk';
	import type { CommandDescriptor, CommandGroup } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import type { CommandWidgetConfig } from '../api/dashboard-api';

	let { config }: { config: CommandWidgetConfig } = $props();

	const deviceId = $derived((config.target.ids ?? [])[0] ?? '');

	// Local state (not a singleton)
	let commands = $state<CommandGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let selectedCommand = $state<CommandDescriptor | null>(null);
	let editorContent = $state('{}');

	let allDescriptors = $derived(commands.flatMap((g) => g.descriptors));
	let groupErrors = $derived(commands.map((g) => g.error).filter(Boolean) as string[]);

	async function loadCommands() {
		const mir = mirStore.mir;
		if (!mir || !deviceId) return;
		isLoading = true;
		error = null;
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			commands = await mir.client().listCommands().request(target);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedCommand = null;
			editorContent = '{}';
			commands = [];
			loadCommands();
		}
	});

	function selectCommand(desc: CommandDescriptor) {
		selectedCommand = desc;
		try {
			editorContent = JSON.stringify(JSON.parse(desc.template || '{}'), null, 2);
		} catch {
			editorContent = desc.template || '{}';
		}
		isSending = false;
		sendError = null;
	}

	async function handleSend(dryRun: boolean, text: string) {
		const mir = mirStore.mir;
		if (!mir || !selectedCommand) return;
		isSending = true;
		sendError = null;
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendCommand()
				.request(target, selectedCommand.name, text, dryRun);
			activityStore.add({
				kind: 'success',
				category: 'Command',
				title: selectedCommand.name,
				request: { deviceId, name: selectedCommand.name, text, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			sendError = err instanceof Error ? err.message : 'Failed to send command';
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: selectedCommand.name,
				request: { deviceId, name: selectedCommand.name, text, dryRun },
				error: sendError
			});
		} finally {
			isSending = false;
		}
	}
</script>

<div class="flex h-full overflow-hidden">
	<DescriptorPanel
		title="Commands"
		items={allDescriptors}
		{isLoading}
		{error}
		{groupErrors}
		selectedName={selectedCommand?.name ?? null}
		emptyText="No commands."
		onSelect={selectCommand}
	/>
	<div class="min-w-0 flex-1 overflow-auto">
		{#if selectedCommand}
			<JsonPayloadEditor
				name={selectedCommand.name}
				value={editorContent}
				hasResponse={false}
				{isSending}
				{sendError}
				onSend={handleSend}
			/>
		{:else}
			<p class="text-muted-foreground p-4 text-xs">Select a command</p>
		{/if}
	</div>
</div>
