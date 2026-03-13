<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import TlmChart from './tlm-chart.svelte';
	import { CHART_COLORS, MAX_AUTO_FIELDS, getDeviceFieldColor } from '$lib/domains/devices/utils/tlm-time';

	let {
		devices,
		measurementFields,
		mergedData,
		chartConfig,
		queryStart = null,
		queryEnd = null,
		useUtc = false,
		initialDeviceId = undefined,
		chartClass = 'h-52',
		forcedFields = undefined,
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
		chartClass?: string;
		forcedFields?: string[];
		onBrushSelect?: (start: Date, end: Date) => void;
	} = $props();

	// Ordered array — index = selection order, drives color assignment
	let selectedDeviceIds = $state<string[]>(
		[initialDeviceId ?? devices[0]?.id ?? ''].filter(Boolean)
	);
	let selectedFields = $state<string[]>(measurementFields.slice(0, MAX_AUTO_FIELDS));

	let activeFields = $derived(forcedFields ?? selectedFields);

	let selectedDevices = $derived(
		selectedDeviceIds
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

	function toggleDevice(deviceId: string, multi: boolean) {
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
<div class="mb-1 flex flex-wrap gap-1">
	{#each measurementFields as field, i (field)}
		<button
			onclick={(e) => toggleField(field, e.shiftKey || e.ctrlKey)}
			class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
				{selectedFields.includes(field)
					? 'border-transparent text-white'
					: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
			style={selectedFields.includes(field)
				? `background: ${CHART_COLORS[i % CHART_COLORS.length]};`
				: ''}
		>
			{field}
		</button>
	{/each}
</div>
{/if}

<!-- Device selector -->
<div class="mb-2 flex flex-wrap gap-1">
	{#each devices as dev (dev.id)}
		<button
			onclick={(e) => toggleDevice(dev.id, e.shiftKey || e.ctrlKey)}
			class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
				{selectedDeviceIds.includes(dev.id)
					? 'border-transparent bg-foreground text-background'
					: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
		>
			{dev.name}
		</button>
	{/each}
</div>

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
