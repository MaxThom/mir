<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { JsonPayloadEditor } from '$lib/domains/devices/components/commands';
	import { DeviceTarget } from '@mir/sdk';
	import type { CommandDescriptor, CommandGroup } from '@mir/sdk';
	import { CommandResponseStatus } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import type { CommandWidgetConfig } from '../api/dashboard-api';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XCircleIcon from '@lucide/svelte/icons/x-circle';

	let {
		config,
		widgetId: _widgetId,
		onDevicesReady
	}: {
		config: CommandWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	// ─── Types ────────────────────────────────────────────────────────────────

	type CmdResponseEntry = {
		idx: number;
		deviceId: string;
		deviceName: string;
		status: 'success' | 'error';
		message: string;
		durationMs: number;
		expanded: boolean;
	};

	// ─── State ────────────────────────────────────────────────────────────────

	let isLoading = $state(false);
	let loadError = $state<string | null>(null);
	let hasLoaded = $state(false);

	let activeDescriptor = $state<CommandDescriptor | null>(null);
	let activeDevices = $state<{ id: string; name: string; namespace: string }[]>([]);
	let editorContent = $state('{}');

	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let responses = $state<CmdResponseEntry[]>([]);
	let hasResponses = $state(false);

	let fullscreen = $state(false);

	// ─── Derived ──────────────────────────────────────────────────────────────

	const deviceValues = $derived.by(() => {
		if (activeDevices.length <= 1) return undefined;
		return activeDevices.map((d) => ({
			label: `${d.name}/${d.namespace}`,
			deviceId: d.id,
			values: editorContent
		}));
	});

	const configKey = $derived(JSON.stringify(config.target ?? {}));

	// ─── Startup ──────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadCommands);
		} else {
			activeDescriptor = null;
			activeDevices = [];
			loadError = null;
		}
	});

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		configKey;
		if (mirStore.mir && untrack(() => hasLoaded)) {
			untrack(loadCommands);
		}
	});

	async function loadCommands() {
		const mir = mirStore.mir;
		if (!mir || !config.selectedCommand) return;

		isLoading = true;
		loadError = null;
		responses = [];
		hasResponses = false;

		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			const groups: CommandGroup[] = await mir.client().listCommands().request(target);

			// Find the group + descriptor for the configured command
			let found: { group: CommandGroup; desc: CommandDescriptor } | null = null;
			for (const g of groups) {
				const desc = g.descriptors.find((d) => d.name === config.selectedCommand);
				if (desc) {
					found = { group: g, desc };
					break;
				}
			}

			if (!found) {
				loadError = `Command "${config.selectedCommand}" not found on target devices`;
				return;
			}

			activeDescriptor = found.desc;
			activeDevices = found.group.ids;
			try {
				editorContent = JSON.stringify(JSON.parse(found.desc.template || '{}'), null, 2);
			} catch {
				editorContent = found.desc.template || '{}';
			}

			const allDevices = groups.flatMap((g) => g.ids);
			onDevicesReady?.(
				allDevices.map((d, i) => ({
					id: d.id,
					name: d.name,
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);

			hasLoaded = true;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
		}
	}

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
	): Promise<Omit<CmdResponseEntry, 'idx'>> {
		const start = performance.now();
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendCommand()
				.request(target, activeDescriptor!.name, text, dryRun);
			const durationMs = Math.round(performance.now() - start);
			const resp = result.get(deviceId) ?? [...result.values()][0];
			const success = resp ? isResponseSuccess(resp.status) : true;
			const message = resp ? responseMessage(resp.status, resp.error) : 'OK';
			activityStore.add({
				kind: success ? 'success' : 'error',
				category: 'Command',
				title: activeDescriptor!.name,
				request: { deviceId, name: activeDescriptor!.name, text, dryRun },
				...(success ? { response: Object.fromEntries(result) } : { error: message })
			});
			return { deviceId, deviceName, status: success ? 'success' : 'error', message, durationMs, expanded: false };
		} catch (err) {
			const durationMs = Math.round(performance.now() - start);
			const message = err instanceof Error ? err.message : 'Failed';
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: activeDescriptor!.name,
				request: { deviceId, name: activeDescriptor!.name, text, dryRun },
				error: message
			});
			return { deviceId, deviceName, status: 'error', message, durationMs, expanded: false };
		}
	}

	// ─── Send (broadcast) ─────────────────────────────────────────────────────

	async function handleSend(dryRun: boolean, text: string) {
		const mir = mirStore.mir;
		if (!mir || !activeDescriptor || activeDevices.length === 0) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		try {
			const results = await Promise.allSettled(
				activeDevices.map((dev) => sendToDevice(mir, dev.id, dev.name, text, dryRun))
			);
			responses = results.map((r, i) =>
				r.status === 'fulfilled'
					? { ...r.value, idx: i }
					: { idx: i, deviceId: '', deviceName: 'unknown', status: 'error' as const, message: String(r.reason), durationMs: 0, expanded: false }
			);
		} catch (err) {
			sendError = err instanceof Error ? err.message : 'Send failed';
			hasResponses = false;
		} finally {
			isSending = false;
		}
	}

	// ─── Send (per-device) ────────────────────────────────────────────────────

	async function handleSendMulti(dryRun: boolean, payloads: Map<string, string>) {
		const mir = mirStore.mir;
		if (!mir || !activeDescriptor) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		try {
			const results = await Promise.allSettled(
				[...payloads.entries()].map(([deviceId, text]) => {
					const dev =
						activeDevices.find((d) => d.id === deviceId) ??
						({ id: deviceId, name: deviceId, namespace: 'default' } as const);
					return sendToDevice(mir, dev.id, dev.name, text, dryRun);
				})
			);
			responses = results.map((r, i) =>
				r.status === 'fulfilled'
					? { ...r.value, idx: i }
					: { idx: i, deviceId: '', deviceName: 'unknown', status: 'error' as const, message: String(r.reason), durationMs: 0, expanded: false }
			);
		} catch (err) {
			sendError = err instanceof Error ? err.message : 'Send failed';
			hasResponses = false;
		} finally {
			isSending = false;
		}
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
	{#if isLoading}
		<div class="flex flex-1 items-center justify-center">
			<span class="text-xs text-muted-foreground">Loading…</span>
		</div>
	{:else if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if !config.selectedCommand}
		<p class="p-4 text-xs text-muted-foreground">No command selected. Edit the widget to choose one.</p>
	{:else if activeDescriptor}
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
				name={activeDescriptor.name}
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
					<span class="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
						>Responses</span
					>
				</div>
				{#each responses as entry (entry.idx)}
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
							<span class="min-w-0 flex-1 truncate font-mono text-xs">{entry.deviceName}</span>
							<span class="shrink-0 font-mono text-[10px] text-muted-foreground"
								>{entry.durationMs}ms</span
							>
							<span class="max-w-32 truncate font-mono text-[10px] text-muted-foreground"
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
	{/if}
</div>
