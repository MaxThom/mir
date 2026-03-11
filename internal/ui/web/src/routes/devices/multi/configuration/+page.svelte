<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
	import { DeviceTarget, ConfigResponseStatus } from '@mir/sdk';
	import type { ConfigGroup, ConfigDescriptor, SendConfigResult } from '@mir/sdk';
	import {
		DescriptorPanel,
		ResponsePanel
	} from '$lib/domains/devices/components/commands';
	import { CfgPayloadEditor } from '$lib/domains/devices/components/configurations';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
	import type { Descriptor } from '$lib/domains/devices/types/types';

	type Selection = {
		groupIdx: number;
		config: ConfigDescriptor;
	} | null;

	let configGroups = $state<ConfigGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let selection = $state<Selection>(null);
	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let response = $state<SendConfigResult | null>(null);

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
					? configGroups[selection.groupIdx]?.values
							.map((v) => v.deviceId)
							.sort()
							.join(',')
					: null;

				configGroups = configGroups
					.map((g) => ({ ...g, values: g.values.filter((v) => currentIds.has(v.deviceId)) }))
					.filter((g) => g.values.length > 0);

				if (selGroupKey) {
					const newIdx = configGroups.findIndex(
						(g) => g.values.map((v) => v.deviceId).sort().join(',') === selGroupKey
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
			loadConfigs();
		}
	});

	async function loadConfigs() {
		if (!mirStore.mir) return;
		isLoading = true;
		error = null;
		try {
			const allIds = selectionStore.activeDevices.map((d) => d.spec.deviceId);
			const target = new DeviceTarget({ ids: allIds });
			const groups = await mirStore.mir.client().listConfigs().request(target);
			configGroups = groups;
			_lastLoadedIds = new Set(allIds);
			selection = null;
			response = null;
			sendError = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load configurations';
		} finally {
			isLoading = false;
		}
	}

	let deviceLookup = $derived(
		new Map(selectionStore.selectedDevices.map((d) => [d.spec.deviceId, d.meta]))
	);

	let groups = $derived(
		configGroups.map((g, idx) => ({
			label:
				g.values
					.map((v) => {
						const meta = deviceLookup.get(v.deviceId);
						return meta ? `${meta.name}/${meta.namespace}` : v.deviceId;
					})
					.join(', ') || `Group ${idx + 1}`,
			items: g.descriptors.map((d) => ({
				name: d.name,
				labels: d.labels,
				template: d.template,
				error: d.error
			})),
			errors: g.error ? [g.error] : []
		}))
	);

	let selectedKey = $derived(selection ? `${selection.groupIdx}:${selection.config.name}` : null);

	let deviceValues = $derived.by(() => {
		if (!selection) return undefined;
		const configName = selection.config.name;
		const group = configGroups[selection.groupIdx];
		return group.values.map((v) => {
			const meta = deviceLookup.get(v.deviceId);
			const label = meta ? `${meta.name}/${meta.namespace}` : v.deviceId;
			return {
				label,
				deviceId: v.deviceId,
				values: prettyJson(v.values?.[configName] ?? '{}')
			};
		});
	});

	let template = $derived(prettyJson(selection?.config.template ?? '{}'));

	function prettyJson(raw: string): string {
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	function statusLabel(status: number): string {
		switch (status) {
			case ConfigResponseStatus.SUCCESS:
				return 'SUCCESS';
			case ConfigResponseStatus.ERROR:
				return 'ERROR';
			case ConfigResponseStatus.VALIDATED:
				return 'VALIDATED';
			case ConfigResponseStatus.NOCHANGE:
				return 'NOCHANGE';
			case ConfigResponseStatus.PENDING:
				return 'PENDING';
			default:
				return 'UNKNOWN';
		}
	}

	function statusClass(status: number): string {
		switch (status) {
			case ConfigResponseStatus.SUCCESS:
				return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400';
			case ConfigResponseStatus.ERROR:
				return 'bg-destructive/15 text-destructive';
			case ConfigResponseStatus.VALIDATED:
			case ConfigResponseStatus.NOCHANGE:
				return 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400';
			default:
				return 'bg-muted text-muted-foreground';
		}
	}

	function selectConfig(groupIdx: number, desc: Descriptor) {
		const full = configGroups[groupIdx].descriptors.find((d) => d.name === desc.name);
		if (!full) return;
		selection = { groupIdx, config: full };
		response = null;
		sendError = null;
	}

	async function sendConfig(dryRun: boolean, text: string) {
		if (!mirStore.mir || !selection) return;
		isSending = true;
		sendError = null;
		try {
			const ids = configGroups[selection.groupIdx].values.map((v) => v.deviceId);
			const target = new DeviceTarget({ ids });
			const result = await mirStore.mir
				.client()
				.sendConfig()
				.request(target, selection.config.name, text, dryRun);
			response = result;
			activityStore.add({
				kind: 'success',
				category: 'Config',
				title: selection.config.name,
				request: { ids, name: selection.config.name, payload: text, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send configuration';
			sendError = message;
			activityStore.add({
				kind: 'error',
				category: 'Config',
				title: selection.config.name,
				request: { name: selection.config.name },
				error: message
			});
		} finally {
			isSending = false;
		}
	}

	async function sendConfigMulti(dryRun: boolean, payloads: Map<string, string>) {
		if (!mirStore.mir || !selection) return;
		isSending = true;
		sendError = null;
		const combined: SendConfigResult = new Map();
		try {
			for (const [deviceId, text] of payloads) {
				const target = new DeviceTarget({ ids: [deviceId] });
				const result = await mirStore.mir
					.client()
					.sendConfig()
					.request(target, selection.config.name, text, dryRun);
				for (const [id, resp] of result) combined.set(id, resp);
			}
			response = combined;
			activityStore.add({
				kind: 'success',
				category: 'Config',
				title: selection.config.name,
				request: {
					ids: [...payloads.keys()],
					name: selection.config.name,
					payloads: Object.fromEntries(payloads),
					dryRun
				},
				response: Object.fromEntries(combined)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send configuration';
			sendError = message;
			activityStore.add({
				kind: 'error',
				category: 'Config',
				title: selection.config.name,
				request: { name: selection.config.name },
				error: message
			});
		} finally {
			isSending = false;
		}
	}
</script>

<div class="-m-4 flex h-[calc(100svh-13rem)] overflow-hidden rounded-none border-y">
	<DescriptorPanel
		title="Configurations"
		items={[]}
		{groups}
		isLoading={isLoading}
		error={error}
		groupErrors={[]}
		{selectedKey}
		emptyText="No configurations found."
		onSelect={() => {}}
		onSelectGrouped={selectConfig}
	/>

	<div class="flex min-w-0 flex-1 overflow-hidden">
		{#if !selection}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<SlidersHorizontalIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a configuration to get started</p>
			</div>
		{:else}
			<CfgPayloadEditor
				name={selection.config.name}
				nameError={selection.config.error}
				{deviceValues}
				{template}
				hasResponse={true}
				{isSending}
				{sendError}
				onSend={sendConfig}
				onSendMulti={sendConfigMulti}
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
