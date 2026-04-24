<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { CfgPayloadEditor } from '$lib/domains/devices/components/configurations';
	import { DeviceTarget } from '@mir/sdk';
	import type { ConfigDescriptor, ConfigGroup } from '@mir/sdk';
	import { ConfigResponseStatus } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import type { ConfigWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XCircleIcon from '@lucide/svelte/icons/x-circle';

	let {
		config,
		widgetId,
		onDevicesReady,
		refreshTick = 0
	}: {
		config: ConfigWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
		refreshTick?: number;
	} = $props();

	// ─── Types ────────────────────────────────────────────────────────────────

	type CfgResponseEntry = {
		idx: number;
		deviceId: string;
		deviceName: string;
		status: 'success' | 'error';
		statusLabel: string;
		message: string;
		durationMs: number;
		expanded: boolean;
	};

	// ─── State ────────────────────────────────────────────────────────────────

	let isLoading = $state(false);
	let loadError = $state<string | null>(null);
	let hasLoaded = $state(false);
	let emptyMessage = $state<string | null>(null);

	let activeDescriptor = $state<ConfigDescriptor | null>(null);
	let activeDevices = $state<{ id: string; name: string; namespace: string }[]>([]);
	let editorContent = $state('{}');
	let deviceContents = $state<Map<string, string>>(new Map());
	let serverDeviceValues = $state<Map<string, string>>(new Map());
	let editorHandle: { getContent: () => string } | undefined = $state(undefined);

	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let responses = $state<CfgResponseEntry[]>([]);
	let hasResponses = $state(false);
	let showResponses = $state(true);

	let fullscreen = $state(false);

	// ─── Derived ──────────────────────────────────────────────────────────────

	const deviceValues = $derived.by(() => {
		if (activeDevices.length <= 1) return undefined;
		return activeDevices.map((d) => ({
			label: `${d.name}/${d.namespace}`,
			deviceId: d.id,
			values: serverDeviceValues.get(d.id) ?? prettyJson(activeDescriptor?.template ?? '{}')
		}));
	});

	const configKey = $derived(
		JSON.stringify({ target: config.target ?? {}, cfg: config.selectedConfig ?? '' })
	);

	// ─── Save payload when exiting edit mode ──────────────────────────────────

	let editModeDirty = $state(false);

	$effect(() => {
		if (dashboardStore.editMode && hasLoaded) editModeDirty = true;
	});

	$effect(() => {
		if (!hasLoaded || dashboardStore.editMode || !editModeDirty) return;
		untrack(() => {
			editModeDirty = false;
			if (!dashboardStore.activeDashboard) return;
			const content = editorHandle?.getContent() ?? editorContent;
			const perDevice = parsePerDeviceBlocks(content);
			if (perDevice) {
				dashboardStore.updateWidgetConfigInMemory(dashboardStore.activeDashboard, widgetId, {
					...config,
					savedPayload: undefined,
					savedPayloads: perDevice
				});
			} else {
				try {
					JSON.parse(content);
					dashboardStore.updateWidgetConfigInMemory(dashboardStore.activeDashboard, widgetId, {
						...config,
						savedPayload: content,
						savedPayloads: undefined
					});
				} catch {
					// Don't save invalid JSON
				}
			}
		});
	});

	// Flush editor content into draft before create() snapshots draftWidgets
	$effect(() => {
		if (!dashboardStore.isSaving || !dashboardStore.isCreatingNew || !hasLoaded) return;
		untrack(() => {
			if (!dashboardStore.activeDashboard) return;
			const content = editorHandle?.getContent() ?? editorContent;
			const perDevice = parsePerDeviceBlocks(content);
			if (perDevice) {
				dashboardStore.updateWidgetConfigInMemory(dashboardStore.activeDashboard, widgetId, {
					...config,
					savedPayload: undefined,
					savedPayloads: perDevice
				});
			} else {
				try {
					JSON.parse(content);
					dashboardStore.updateWidgetConfigInMemory(dashboardStore.activeDashboard, widgetId, {
						...config,
						savedPayload: content,
						savedPayloads: undefined
					});
				} catch {
					// Don't save invalid JSON
				}
			}
		});
	});

	function parsePerDeviceBlocks(content: string): Record<string, string> | null {
		if (!content.trimStart().startsWith('// ')) return null;
		const result: Record<string, string> = {};
		const blocks = content.split(/\n(?=\/\/ )/);
		for (const block of blocks) {
			const nlIdx = block.indexOf('\n');
			if (nlIdx === -1) continue;
			const label = block.slice(0, nlIdx).trim().replace(/^\/\/ /, '');
			const jsonText = block.slice(nlIdx + 1).trim();
			const dev = activeDevices.find((d) => `${d.name}/${d.namespace}` === label);
			if (dev && jsonText) result[dev.id] = jsonText;
		}
		return Object.keys(result).length > 0 ? result : null;
	}

	// ─── Startup ──────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadConfigs);
		} else {
			activeDescriptor = null;
			activeDevices = [];
			loadError = null;
			isLoading = false;
		}
	});

	$effect(() => {
		// eslint-disable-next-line @typescript-eslint/no-unused-expressions
		configKey;
		if (mirStore.mir && untrack(() => hasLoaded)) {
			untrack(loadConfigs);
		}
	});

	// Reset editor to saved config when entering edit mode
	$effect(() => {
		if (dashboardStore.editMode && mirStore.mir && hasLoaded) {
			untrack(loadConfigs);
		}
	});

	// Auto-refresh: reload config values when dashboard refresh fires
	$effect(() => {
		if (refreshTick > 0 && mirStore.mir && hasLoaded) {
			untrack(loadConfigs);
		}
	});

	// ─── Load ─────────────────────────────────────────────────────────────────

	async function loadConfigs() {
		const mir = mirStore.mir;
		if (!mir || !config.selectedConfig) return;

		if (!hasLoaded) isLoading = true;
		loadError = null;
		emptyMessage = null;

		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			const groups: ConfigGroup[] = await mir.client().listConfigs().request(target);

			if (groups.length === 0) {
				emptyMessage = 'No devices found for this target.';
				hasLoaded = true;
				return;
			}

			let found: { group: ConfigGroup; desc: ConfigDescriptor } | null = null;
			for (const g of groups) {
				const desc = g.descriptors.find((d) => d.name === config.selectedConfig);
				if (desc) {
					found = { group: g, desc };
					break;
				}
			}

			if (!found) {
				emptyMessage = 'Configuration not found for this target.';
				hasLoaded = true;
				return;
			}

			activeDescriptor = found.desc;
			activeDevices = found.group.ids;

			// Always load current device values from server for per-device display
			serverDeviceValues = new Map(
				found.group.ids.map((d) => {
					const cv = found!.group.values.find((v) => v.deviceId === d.id);
					return [d.id, prettyJson(cv?.values[config.selectedConfig!] ?? '{}')];
				})
			);

			if (config.savedPayloads && Object.keys(config.savedPayloads).length > 0) {
				deviceContents = new Map(Object.entries(config.savedPayloads));
				const firstDev = found.group.ids[0];
				const cv = firstDev
					? found.group.values.find((v) => v.deviceId === firstDev.id)
					: undefined;
				editorContent = prettyJson(cv?.values[config.selectedConfig!] ?? '{}');
			} else if (config.savedPayload) {
				editorContent = config.savedPayload;
				deviceContents = new Map();
			} else {
				// Default: current values from first device for single-slot editor
				const firstDev = found.group.ids[0];
				const firstCv = firstDev
					? found.group.values.find((v) => v.deviceId === firstDev.id)
					: undefined;
				editorContent = prettyJson(firstCv?.values[config.selectedConfig!] ?? '{}');
				deviceContents = new Map();
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
			const msg = err instanceof Error ? err.message : 'Failed to load configs';
			if (msg === 'no device found with current targets criteria') {
				emptyMessage = 'No devices found for this target.';
				hasLoaded = true;
			} else {
				loadError = msg;
			}
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

	// ─── Response helpers ─────────────────────────────────────────────────────

	function isResponseSuccess(status: ConfigResponseStatus): boolean {
		return (
			status === ConfigResponseStatus.SUCCESS ||
			status === ConfigResponseStatus.VALIDATED ||
			status === ConfigResponseStatus.NOCHANGE
		);
	}

	function responseStatusLabel(status: ConfigResponseStatus): string {
		return ConfigResponseStatus[status] ?? 'UNKNOWN';
	}

	function responseMessage(status: ConfigResponseStatus, error: string): string {
		if (error) return error;
		return responseStatusLabel(status);
	}

	// ─── Send helpers ─────────────────────────────────────────────────────────

	async function sendToDevice(
		mir: NonNullable<typeof mirStore.mir>,
		deviceId: string,
		deviceName: string,
		text: string,
		dryRun: boolean
	): Promise<Omit<CfgResponseEntry, 'idx'>> {
		const start = performance.now();
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendConfig()
				.request(target, activeDescriptor!.name, text, dryRun);
			const durationMs = Math.round(performance.now() - start);
			const resp = result.get(deviceId) ?? [...result.values()][0];
			const success = resp ? isResponseSuccess(resp.status) : true;
			const statusLabel = resp ? responseStatusLabel(resp.status) : 'OK';
			const message = resp ? responseMessage(resp.status, resp.error) : 'OK';
			activityStore.add({
				kind: success ? 'success' : 'error',
				category: 'Config',
				title: activeDescriptor!.name,
				request: { deviceId, name: activeDescriptor!.name, text, dryRun },
				...(success ? { response: Object.fromEntries(result) } : { error: message })
			});
			return {
				deviceId,
				deviceName,
				status: success ? 'success' : 'error',
				statusLabel,
				message,
				durationMs,
				expanded: false
			};
		} catch (err) {
			const durationMs = Math.round(performance.now() - start);
			const message = err instanceof Error ? err.message : 'Failed';
			activityStore.add({
				kind: 'error',
				category: 'Config',
				title: activeDescriptor!.name,
				request: { deviceId, name: activeDescriptor!.name, text, dryRun },
				error: message
			});
			return {
				deviceId,
				deviceName,
				status: 'error',
				statusLabel: 'ERROR',
				message,
				durationMs,
				expanded: false
			};
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
		showResponses = true;

		try {
			const results = await Promise.allSettled(
				activeDevices.map((dev) => sendToDevice(mir, dev.id, dev.name, text, dryRun))
			);
			responses = results.map((r, i) =>
				r.status === 'fulfilled'
					? { ...r.value, idx: i }
					: {
							idx: i,
							deviceId: '',
							deviceName: 'unknown',
							status: 'error' as const,
							statusLabel: 'ERROR',
							message: String(r.reason),
							durationMs: 0,
							expanded: false
						}
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
		showResponses = true;

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
					: {
							idx: i,
							deviceId: '',
							deviceName: 'unknown',
							status: 'error' as const,
							statusLabel: 'ERROR',
							message: String(r.reason),
							durationMs: 0,
							expanded: false
						}
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
	<div class="mt-2.5 border-b"></div>
	{#if isLoading}
		<div class="flex flex-1 items-center justify-center">
			<span class="text-xs text-muted-foreground">Loading…</span>
		</div>
	{:else if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else if !config.selectedConfig}
		<p class="p-4 text-xs text-muted-foreground">No configuration selected.</p>
	{:else if emptyMessage}
		<p class="p-4 text-xs text-muted-foreground">{emptyMessage}</p>
	{:else if activeDescriptor}
		<!-- Config editor -->
		<div class="flex min-h-16 flex-1 overflow-hidden">
			<CfgPayloadEditor
				bind:this={editorHandle}
				name={activeDescriptor.name}
				currentValues={editorContent}
				template={prettyJson(activeDescriptor.template ?? '{}')}
				hasResponse={false}
				{isSending}
				{sendError}
				{deviceValues}
				onSend={handleSend}
				onSendMulti={handleSendMulti}
			>
				{#snippet headerEnd()}
					<button
						onclick={() => (fullscreen = !fullscreen)}
						title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
						class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
					>
						{#if fullscreen}
							<MinimizeIcon class="size-3.5" />
						{:else}
							<MaximizeIcon class="size-3.5" />
						{/if}
					</button>
				{/snippet}
				{#snippet footerEnd()}
					{#if hasResponses}
						<button
							onclick={() => (showResponses = !showResponses)}
							class="ml-auto rounded px-2 py-0.5 font-mono text-[10px] text-muted-foreground transition-colors hover:text-foreground"
						>
							{showResponses ? 'Hide Response' : 'Show Response'}
						</button>
					{/if}
				{/snippet}
			</CfgPayloadEditor>
		</div>

		<!-- Response log (appears after first send) -->
		{#if hasResponses && showResponses}
			<div class="max-h-[45%] min-h-0 shrink overflow-y-auto border-t">
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
								>{entry.statusLabel}</span
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
