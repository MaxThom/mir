<script lang="ts">
	import { page } from '$app/state';
	import { getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { configStore } from '$lib/domains/devices/stores/config.svelte';
	import { ConfigResponseStatus } from '@mir/sdk';
	import type { ConfigDescriptor } from '@mir/sdk';
	import {
		DescriptorPanel,
		ResponsePanel
	} from '$lib/domains/devices/components/commands';
	import { CfgPayloadEditor } from '$lib/domains/devices/components/configurations';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';

	let deviceId = $derived(page.params.deviceId ?? '');
	let selectedConfig = $state<ConfigDescriptor | null>(null);

	let allDescriptors = $derived(
		configStore.configs.flatMap((g) => g.descriptors).sort((a, b) => a.name.localeCompare(b.name))
	);
	let groupErrors = $derived(configStore.configs.map((g) => g.error).filter(Boolean));

	// Assumes config names are unique across groups for a given device
	let currentValuesMap = $derived.by(() => {
		const map: Record<string, string> = {};
		for (const group of configStore.configs) {
			const dv = group.values.find((v) => v.deviceId === deviceId);
			if (dv) Object.assign(map, dv.values);
		}
		return map;
	});

	let currentValues = $derived(prettyJson(currentValuesMap[selectedConfig?.name ?? ''] ?? '{}'));
	let template = $derived(prettyJson(selectedConfig?.template ?? '{}'));

	const deviceCtx = getContext<{ setTabRefresh: (fn: (() => void) | null) => void }>('device');
	deviceCtx.setTabRefresh(() => {
		if (mirStore.mir && deviceId) {
			configStore.loadConfigs(mirStore.mir, deviceId);
		}
	});
	onDestroy(() => deviceCtx.setTabRefresh(null));

	$effect(() => {
		if (mirStore.mir && deviceId) {
			selectedConfig = null;
			configStore.reset();
			configStore.loadConfigs(mirStore.mir, deviceId);
		}
	});

	function prettyJson(raw: string): string {
		try {
			return JSON.stringify(JSON.parse(raw), null, 2);
		} catch {
			return raw;
		}
	}

	function selectConfig(desc: ConfigDescriptor) {
		selectedConfig = desc;
		configStore.reset();
	}

	function handleSend(dryRun: boolean, text: string) {
		if (!mirStore.mir || !selectedConfig) return;
		configStore.sendConfig(mirStore.mir, deviceId, selectedConfig.name, text, dryRun);
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
</script>

<div class="-m-4 flex min-h-125 overflow-hidden rounded-none border-y">
	<DescriptorPanel
		title="Configurations"
		items={allDescriptors}
		isLoading={configStore.isLoading}
		error={configStore.error}
		{groupErrors}
		selectedName={selectedConfig?.name ?? null}
		emptyText="No configurations found."
		onSelect={selectConfig}
	/>

	<div class="flex min-w-0 flex-1 overflow-hidden">
		{#if !selectedConfig}
			<div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
				<SlidersHorizontalIcon class="size-8 opacity-30" />
				<p class="text-sm">Select a configuration to get started</p>
			</div>
		{:else}
			<CfgPayloadEditor
				name={selectedConfig.name}
				nameError={selectedConfig.error}
				{currentValues}
				{template}
				hasResponse={true}
				isSending={configStore.isSending}
				sendError={configStore.sendError}
				onSend={handleSend}
			/>
			<ResponsePanel
				response={configStore.response}
				{statusLabel}
				{statusClass}
				onClear={() => configStore.reset()}
			/>
		{/if}
	</div>
</div>
