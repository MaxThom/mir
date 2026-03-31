<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DescriptorPanel, JsonPayloadEditor } from '$lib/domains/devices/components/commands';
	import { DeviceTarget } from '@mir/sdk';
	import type { CommandDescriptor, CommandGroup } from '@mir/sdk';
	import { CommandResponseStatus } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import type { CommandWidgetConfig } from '../api/dashboard-api';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XCircleIcon from '@lucide/svelte/icons/x-circle';

	let {
		config,
		widgetId,
		onDevicesReady
	}: {
		config: CommandWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	// ─── Types ────────────────────────────────────────────────────────────────

	type CmdResponseEntry = {
		deviceId: string;
		deviceName: string;
		status: 'success' | 'error';
		message: string;
		durationMs: number;
		expanded: boolean;
	};

	// ─── State ────────────────────────────────────────────────────────────────

	let groups = $state<CommandGroup[]>([]);
	let isLoading = $state(false);
	let loadError = $state<string | null>(null);
	let hasLoaded = $state(false);

	let selectedGroupIdx = $state<number | null>(null);
	let selectedCommand = $state<CommandDescriptor | null>(null);
	let editorContent = $state('{}');

	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let responses = $state<CmdResponseEntry[]>([]);
	let hasResponses = $state(false);

	let fullscreen = $state(false);

	// ─── Derived ──────────────────────────────────────────────────────────────

	const selectedKey = $derived(
		selectedGroupIdx !== null && selectedCommand
			? `${selectedGroupIdx}:${selectedCommand.name}`
			: null
	);

	const descriptorGroups = $derived(
		groups.map((g) => ({
			label: g.ids.map((d) => `${d.name}/${d.namespace}`).join(', '),
			items: g.descriptors,
			errors: g.error ? [g.error] : []
		}))
	);

	const groupErrors = $derived(groups.map((g) => g.error).filter(Boolean) as string[]);

	const selectedGroupDevices = $derived(
		selectedGroupIdx !== null ? (groups[selectedGroupIdx]?.ids ?? []) : []
	);

	const deviceValues = $derived.by(() => {
		if (selectedGroupDevices.length <= 1) return undefined;
		return selectedGroupDevices.map((d) => ({
			label: `${d.name}/${d.namespace}`,
			deviceId: d.id,
			values: editorContent
		}));
	});

	// ─── Startup ──────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadCommands);
		} else {
			groups = [];
			loadError = null;
		}
	});

	async function loadCommands() {
		const mir = mirStore.mir;
		if (!mir) return;

		isLoading = true;
		loadError = null;
		groups = [];
		selectedCommand = null;
		selectedGroupIdx = null;
		responses = [];
		hasResponses = false;

		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			groups = await mir.client().listCommands().request(target);

			const allDevices = groups.flatMap((g) => g.ids);
			onDevicesReady?.(
				allDevices.map((d, i) => ({
					id: d.id,
					name: d.name,
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);

			// Restore selected command from view state
			if (config.selectedCommand) {
				for (let gi = 0; gi < groups.length; gi++) {
					const desc = groups[gi].descriptors.find((d) => d.name === config.selectedCommand);
					if (desc) {
						selectCommand(gi, desc);
						break;
					}
				}
			}
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
			hasLoaded = true;
		}
	}

	// ─── Selection ────────────────────────────────────────────────────────────

	function selectCommand(groupIdx: number, desc: CommandDescriptor) {
		selectedGroupIdx = groupIdx;
		selectedCommand = desc;
		try {
			editorContent = JSON.stringify(JSON.parse(desc.template || '{}'), null, 2);
		} catch {
			editorContent = desc.template || '{}';
		}
		isSending = false;
		sendError = null;
		responses = [];
		hasResponses = false;
	}

	// ─── View state ───────────────────────────────────────────────────────────

	$effect(() => {
		if (!hasLoaded) return;
		const name = selectedCommand?.name;
		untrack(() => {
			dashboardStore.saveWidgetViewState(widgetId, {
				...config,
				selectedCommand: name
			});
		});
	});

	// ─── Send helpers ─────────────────────────────────────────────────────────

	function isResponseSuccess(status: CommandResponseStatus): boolean {
		return status === CommandResponseStatus.SUCCESS || status === CommandResponseStatus.VALIDATED;
	}

	function responseMessage(status: CommandResponseStatus, error: string): string {
		if (error) return error;
		return CommandResponseStatus[status] ?? 'OK';
	}

	async function sendToDevice(
		mir: NonNullable<typeof mirStore.mir>,
		deviceId: string,
		deviceName: string,
		text: string,
		dryRun: boolean
	): Promise<CmdResponseEntry> {
		const start = performance.now();
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendCommand()
				.request(target, selectedCommand!.name, text, dryRun);
			const durationMs = Math.round(performance.now() - start);
			const resp = result.get(deviceId) ?? [...result.values()][0];
			const success = resp ? isResponseSuccess(resp.status) : true;
			const message = resp ? responseMessage(resp.status, resp.error) : 'OK';
			activityStore.add({
				kind: success ? 'success' : 'error',
				category: 'Command',
				title: selectedCommand!.name,
				request: { deviceId, name: selectedCommand!.name, text, dryRun },
				...(success ? { response: Object.fromEntries(result) } : { error: message })
			});
			return { deviceId, deviceName, status: success ? 'success' : 'error', message, durationMs, expanded: false };
		} catch (err) {
			const durationMs = Math.round(performance.now() - start);
			const message = err instanceof Error ? err.message : 'Failed';
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: selectedCommand!.name,
				request: { deviceId, name: selectedCommand!.name, text, dryRun },
				error: message
			});
			return { deviceId, deviceName, status: 'error', message, durationMs, expanded: false };
		}
	}

	// ─── Send (broadcast) ─────────────────────────────────────────────────────

	async function handleSend(dryRun: boolean, text: string) {
		const mir = mirStore.mir;
		if (!mir || !selectedCommand || selectedGroupDevices.length === 0) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		const results = await Promise.allSettled(
			selectedGroupDevices.map((dev) => sendToDevice(mir, dev.id, dev.name, text, dryRun))
		);

		responses = results.map((r) =>
			r.status === 'fulfilled'
				? r.value
				: {
						deviceId: '',
						deviceName: 'unknown',
						status: 'error' as const,
						message: String(r.reason),
						durationMs: 0,
						expanded: false
					}
		);

		isSending = false;
	}

	// ─── Send (per-device) ────────────────────────────────────────────────────

	async function handleSendMulti(dryRun: boolean, payloads: Map<string, string>) {
		const mir = mirStore.mir;
		if (!mir || !selectedCommand) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		const results = await Promise.allSettled(
			[...payloads.entries()].map(([deviceId, text]) => {
				const dev =
					selectedGroupDevices.find((d) => d.id === deviceId) ??
					({ id: deviceId, name: deviceId, namespace: 'default' } as const);
				return sendToDevice(mir, dev.id, dev.name, text, dryRun);
			})
		);

		responses = results.map((r) =>
			r.status === 'fulfilled'
				? r.value
				: {
						deviceId: '',
						deviceName: 'unknown',
						status: 'error' as const,
						message: String(r.reason),
						durationMs: 0,
						expanded: false
					}
		);

		isSending = false;
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && fullscreen) fullscreen = false;
	}}
/>

<div
	class="{fullscreen
		? 'fixed inset-0 z-50 bg-background'
		: 'h-full'} flex flex-col overflow-hidden"
>
	{#if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else}
		<div class="flex min-h-0 flex-1 overflow-hidden">
			<!-- Left: command list -->
			<DescriptorPanel
				title="Commands"
				items={[]}
				{isLoading}
				error={null}
				{groupErrors}
				groups={descriptorGroups}
				onSelect={() => {}}
				onSelectGrouped={(gi, desc) => selectCommand(gi, desc)}
				selectedKey={selectedKey}
				emptyText="No commands."
			/>

			<!-- Right: editor + responses -->
			<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
				{#if selectedCommand}
					<!-- Fullscreen toggle -->
					<div class="flex shrink-0 items-center justify-end px-2 py-0.5">
						<button
							onclick={() => (fullscreen = !fullscreen)}
							title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
							class="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
						>
							{#if fullscreen}
								<MinimizeIcon class="size-3.5" />
							{:else}
								<MaximizeIcon class="size-3.5" />
							{/if}
						</button>
					</div>

					<!-- JSON editor -->
					<div class="flex min-h-0 flex-1 overflow-hidden">
						<JsonPayloadEditor
							name={selectedCommand.name}
							value={editorContent}
							hasResponse={false}
							{isSending}
							{sendError}
							{deviceValues}
							onSend={handleSend}
							onSendMulti={handleSendMulti}
						/>
					</div>

					<!-- Response log (appears after first send) -->
					{#if hasResponses}
						<div class="max-h-48 shrink-0 overflow-y-auto border-t">
							<div class="border-b px-3 py-1">
								<span
									class="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
									>Responses</span
								>
							</div>
							{#each responses as entry (entry.deviceId + entry.deviceName)}
								<div class="border-b last:border-0">
									<button
										class="flex w-full items-center gap-2 px-3 py-1.5 text-left hover:bg-accent/50"
										onclick={() => (entry.expanded = !entry.expanded)}
									>
										{#if entry.status === 'success'}
											<CheckIcon class="size-3 shrink-0 text-emerald-500" />
										{:else}
											<XCircleIcon class="size-3 shrink-0 text-destructive" />
										{/if}
										<span class="min-w-0 flex-1 truncate font-mono text-xs"
											>{entry.deviceName}</span
										>
										<span class="shrink-0 font-mono text-[10px] text-muted-foreground"
											>{entry.durationMs}ms</span
										>
										<span
											class="max-w-32 truncate font-mono text-[10px] text-muted-foreground"
											>{entry.message}</span
										>
									</button>
									{#if entry.expanded}
										<pre
											class="overflow-x-auto bg-muted/40 px-3 py-2 font-mono text-[11px] break-all whitespace-pre-wrap">{entry.message}</pre>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				{:else}
					<p class="p-4 text-xs text-muted-foreground">Select a command</p>
				{/if}
			</div>
		</div>
	{/if}
</div>
