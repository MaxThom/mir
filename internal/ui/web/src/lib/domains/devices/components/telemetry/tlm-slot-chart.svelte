<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import TlmChart from './tlm-chart.svelte';
	import TlmFieldToggles from './tlm-field-toggles.svelte';
	import { MAX_AUTO_FIELDS, getDeviceFieldColor } from '$lib/domains/devices/utils/tlm-time';
	import { untrack } from 'svelte';

	let {
		devices,
		measurementFields,
		mergedData,
		chartConfig,
		queryStart = null,
		queryEnd = null,
		useUtc = false,
		initialDeviceId = undefined,
		initialFields = undefined,
		chartClass = 'h-52',
		forcedFields = undefined,
		forcedDeviceIds = undefined,
		onToggleDevice = undefined,
		onBrushSelect
	}: {
		devices: { id: string; name: string }[];
		measurementFields: string[];
		mergedData: QueryData;
		chartConfig: ChartConfig;
		queryStart?: Date | null;
		queryEnd?: Date | null;
		useUtc?: boolean;
		initialDeviceId?: string;
		initialFields?: string[];
		chartClass?: string;
		forcedFields?: string[];
		forcedDeviceIds?: string[];
		onToggleDevice?: (id: string, multi: boolean) => void;
		onBrushSelect?: (start: Date, end: Date) => void;
	} = $props();

	// Ordered array — index = selection order, drives color assignment
	let selectedDeviceIds = $state<string[]>(
		untrack(() => [initialDeviceId ?? devices[0]?.id ?? ''].filter(Boolean))
	);
	let selectedFields = $state<string[]>(untrack(() => initialFields ?? measurementFields.slice(0, MAX_AUTO_FIELDS)));

	let activeFields = $derived(forcedFields ?? selectedFields);
	let activeDeviceIds = $derived(forcedDeviceIds ?? selectedDeviceIds);

	let selectedDevices = $derived(
		activeDeviceIds
			.map((id) => devices.find((d) => d.id === id))
			.filter((d): d is { id: string; name: string } => d !== undefined)
	);

	let visibleFields = $derived(
		selectedDevices.flatMap((d) => activeFields.map((f) => `${d.name}_${f}`))
	);

	let localChartConfig = $derived.by(() => {
		if (selectedDevices.length <= 1) return chartConfig;
		const result = { ...chartConfig };
		selectedDevices.forEach((dev, devIdx) => {
			activeFields.forEach((field) => {
				const fieldIdx = measurementFields.indexOf(field);
				const key = `${dev.name}_${field}`;
				if (key in result) result[key] = { ...result[key], color: getDeviceFieldColor(fieldIdx, devIdx) } as typeof result[string];
			});
		});
		return result;
	});

	const fieldUnits = $derived(
		Object.fromEntries(
			measurementFields.map((f) => [
				f,
				chartConfig[`${devices[0]?.name ?? ''}_${f}`]?.unit ?? ''
			])
		)
	);

	function toggleDevice(deviceId: string, multi: boolean) {
		if (onToggleDevice) {
			onToggleDevice(deviceId, multi);
			return;
		}
		if (multi) {
			if (selectedDeviceIds.includes(deviceId)) {
				if (selectedDeviceIds.length > 1) selectedDeviceIds = selectedDeviceIds.filter((id) => id !== deviceId);
			} else {
				selectedDeviceIds = [...selectedDeviceIds, deviceId];
			}
		} else {
			selectedDeviceIds = [deviceId];
		}
	}

	function toggleField(field: string, shift: boolean) {
		if (shift) {
			if (selectedFields.includes(field)) {
				if (selectedFields.length > 1) selectedFields = selectedFields.filter((f) => f !== field);
			} else {
				selectedFields = [...selectedFields, field];
			}
		} else {
			selectedFields = [field];
		}
	}
</script>

<!-- Field selector (hidden when fields are synced externally) -->
{#if !forcedFields}
<TlmFieldToggles fields={measurementFields} {selectedFields} ontoggle={toggleField} {fieldUnits} class="mb-1" />
{/if}

<!-- Chart -->
<TlmChart
	data={mergedData}
	selectedFields={visibleFields}
	chartConfig={localChartConfig}
	{useUtc}
	start={queryStart}
	end={queryEnd}
	{chartClass}
	{onBrushSelect}
/>

<!-- Device legend -->
{#if devices.length > 0 && activeFields.length > 0}
	<div class="mt-0.5 flex flex-wrap gap-x-3 gap-y-0.5 px-4">
		{#each devices as dev (dev.id)}
			{@const enabled = activeDeviceIds.includes(dev.id)}
			{@const color = localChartConfig[`${dev.name}_${activeFields[0]}`]?.color ?? 'var(--chart-1)'}
			<button
				onclick={(e) => toggleDevice(dev.id, e.shiftKey || e.ctrlKey)}
				class="flex items-center gap-1 font-mono text-[11px] transition-colors {enabled ? 'text-foreground' : 'text-muted-foreground/40'}"
			>
				<span
					class="inline-block size-2 shrink-0 rounded-full transition-opacity {enabled ? '' : 'opacity-30'}"
					style="background: {color}"
				></span>
				{dev.name}
			</button>
		{/each}
	</div>
{/if}
