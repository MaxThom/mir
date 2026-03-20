<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DescriptorPanel } from '$lib/domains/devices/components/commands';
	import { CfgPayloadEditor } from '$lib/domains/devices/components/configurations';
	import { DeviceTarget } from '@mir/sdk';
	import type { ConfigDescriptor, ConfigGroup } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import type { ConfigWidgetConfig } from '../api/dashboard-api';

	let { config }: { config: ConfigWidgetConfig } = $props();

	const deviceId = $derived((config.target.ids ?? [])[0] ?? '');

	// Local state
	let configs = $state<ConfigGroup[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let selectedConfig = $state<ConfigDescriptor | null>(null);

	let allDescriptors = $derived(configs.flatMap((g) => g.descriptors));
	let groupErrors = $derived(configs.map((g) => g.error).filter(Boolean) as string[]);

	let currentValuesMap = $derived.by(() => {
		const map: Record<string, string> = {};
		for (const group of configs) {
			const dv = group.values.find((v) => v.deviceId === deviceId);
			if (dv) Object.assign(map, dv.values);
		}
		return map;
	});

	function prettyJson(raw: string): string {
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	let currentValues = $derived(prettyJson(currentValuesMap[selectedConfig?.name ?? ''] ?? '{}'));
	let template = $derived(prettyJson(selectedConfig?.template ?? '{}'));

	async function loadConfigs() {
		const mir = mirStore.mir;
		if (!mir || !deviceId) return;
		isLoading = true;
		error = null;
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			configs = await mir.client().listConfigs().request(target);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load configs';
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedConfig = null;
			configs = [];
			loadConfigs();
		}
	});

	function selectConfig(desc: ConfigDescriptor) {
		selectedConfig = desc;
		isSending = false;
		sendError = null;
	}

	async function handleSend(dryRun: boolean, text: string) {
		const mir = mirStore.mir;
		if (!mir || !selectedConfig) return;
		isSending = true;
		sendError = null;
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendConfig()
				.request(target, selectedConfig.name, text, dryRun);
			activityStore.add({
				kind: 'success',
				category: 'Config',
				title: selectedConfig.name,
				request: { deviceId, name: selectedConfig.name, text, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			sendError = err instanceof Error ? err.message : 'Failed to send config';
			activityStore.add({
				kind: 'error',
				category: 'Config',
				title: selectedConfig?.name ?? '',
				request: { deviceId, name: selectedConfig?.name ?? '', text, dryRun },
				error: sendError
			});
		} finally {
			isSending = false;
		}
	}
</script>

<div class="flex h-full overflow-hidden">
	<DescriptorPanel
		title="Configurations"
		items={allDescriptors}
		{isLoading}
		{error}
		{groupErrors}
		selectedName={selectedConfig?.name ?? null}
		emptyText="No configurations."
		onSelect={selectConfig}
	/>
	<div class="min-w-0 flex-1 overflow-auto">
		{#if selectedConfig}
			<CfgPayloadEditor
				name={selectedConfig.name}
				{template}
				{currentValues}
				hasResponse={false}
				{isSending}
				{sendError}
				onSend={handleSend}
			/>
		{:else}
			<p class="text-muted-foreground p-4 text-xs">Select a configuration</p>
		{/if}
	</div>
</div>
