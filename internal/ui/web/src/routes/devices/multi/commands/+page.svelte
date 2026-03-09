<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { DeviceTarget, CommandResponseStatus } from '@mir/sdk';
	import type { CommandGroup, CommandDescriptor, SendCommandResult } from '@mir/sdk';
	import {
		DescriptorPanel,
		JsonPayloadEditor,
		ResponsePanel
	} from '$lib/domains/devices/components/commands';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import type { Descriptor } from '$lib/domains/devices/types/types';

	type GroupState = {
		group: CommandGroup;
		selectedCommand: CommandDescriptor | null;
		editorContent: string;
		isSending: boolean;
		sendError: string | null;
		response: SendCommandResult | null;
	};

	let commandGroups = $state<CommandGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let groupStates = $state<GroupState[]>([]);

	$effect(() => {
		if (!mirStore.mir || selectionStore.count === 0) return;
		loadCommands();
	});

	async function loadCommands() {
		if (!mirStore.mir) return;
		isLoading = true;
		error = null;
		try {
			const allIds = selectionStore.selectedDevices.map((d) => d.spec.deviceId);
			const target = new DeviceTarget({ ids: allIds });
			const groups = await mirStore.mir.client().listCommands().request(target);
			commandGroups = groups;
			groupStates = groups.map((g) => ({
				group: g,
				selectedCommand: null,
				editorContent: '{}',
				isSending: false,
				sendError: null,
				response: null
			}));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
		}
	}

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
		const gs = groupStates[groupIdx];
		const fullDesc = gs.group.descriptors.find((d) => d.name === desc.name);
		if (!fullDesc) return;
		groupStates[groupIdx] = {
			...gs,
			selectedCommand: fullDesc,
			editorContent: prettyJson(fullDesc.template || '{}'),
			response: null,
			sendError: null
		};
	}

	async function sendCommand(groupIdx: number, dryRun: boolean, text: string) {
		if (!mirStore.mir) return;
		const gs = groupStates[groupIdx];
		if (!gs.selectedCommand) return;
		groupStates[groupIdx] = { ...gs, isSending: true, sendError: null };
		try {
			const ids = gs.group.ids.map((id) => id.id);
			const target = new DeviceTarget({ ids });
			const result = await mirStore.mir
				.client()
				.sendCommand()
				.request(target, gs.selectedCommand.name, text, dryRun);
			groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, response: result };
			activityStore.add({
				kind: 'success',
				category: 'Command',
				title: gs.selectedCommand.name,
				request: { ids, name: gs.selectedCommand.name, payload: text, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send command';
			groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, sendError: message };
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: gs.selectedCommand.name,
				request: { name: gs.selectedCommand.name },
				error: message
			});
		}
	}
</script>

<div class="flex flex-col gap-6">
	{#if isLoading}
		<div class="flex items-center justify-center py-12 text-muted-foreground">
			<Spinner class="mr-2 size-4" />
			Loading commands...
		</div>
	{:else if error}
		<p class="text-sm text-destructive">{error}</p>
	{:else if commandGroups.length === 0}
		<div class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
			<TerminalIcon class="size-8 opacity-30" />
			<p class="text-sm">No commands found for selected devices</p>
		</div>
	{:else}
		{#each groupStates as gs, idx (idx)}
			<div>
				<div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
					<span class="font-mono font-medium text-foreground">
						{gs.group.ids.map((id) => `${id.name}/${id.namespace}`).join(', ')}
					</span>
					<span
						>({gs.group.ids.length} device{gs.group.ids.length > 1 ? 's' : ''})</span
					>
				</div>

				<div class="flex min-h-80 overflow-hidden rounded-lg border">
					<DescriptorPanel
						title="Commands"
						items={gs.group.descriptors.map((d) => ({
							name: d.name,
							labels: d.labels,
							template: d.template,
							error: d.error
						}))}
						isLoading={false}
						error={null}
						groupErrors={gs.group.error ? [gs.group.error] : []}
						selectedName={gs.selectedCommand?.name ?? null}
						emptyText="No commands."
						onSelect={(desc) => selectCommand(idx, desc)}
					/>

					<div class="flex min-w-0 flex-1 overflow-hidden">
						{#if !gs.selectedCommand}
							<div
								class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground"
							>
								<TerminalIcon class="size-8 opacity-30" />
								<p class="text-sm">Select a command</p>
							</div>
						{:else}
							<JsonPayloadEditor
								name={gs.selectedCommand.name}
								nameError={gs.selectedCommand.error}
								value={gs.editorContent}
								hasResponse={true}
								isSending={gs.isSending}
								sendError={gs.sendError}
								onSend={(dryRun, text) => sendCommand(idx, dryRun, text)}
							/>
							<ResponsePanel
								response={gs.response}
								{statusLabel}
								{statusClass}
								onClear={() => {
									groupStates[idx] = { ...groupStates[idx], response: null, sendError: null };
								}}
							/>
						{/if}
					</div>
				</div>
			</div>

			{#if idx < groupStates.length - 1}
				<Separator />
			{/if}
		{/each}
	{/if}
</div>
