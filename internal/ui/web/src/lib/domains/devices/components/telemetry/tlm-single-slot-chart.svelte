<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import TlmChart from './tlm-chart.svelte';
	import { CHART_COLORS, MAX_AUTO_FIELDS } from '$lib/domains/devices/utils/tlm-time';

	let {
		measurementFields,
		data,
		chartConfig,
		queryStart = null,
		queryEnd = null,
		useUtc = false,
		chartClass = 'h-52',
		initialFields = undefined,
		onBrushSelect
	}: {
		measurementFields: string[];
		data: QueryData;
		chartConfig: ChartConfig;
		queryStart?: Date | null;
		queryEnd?: Date | null;
		useUtc?: boolean;
		chartClass?: string;
		initialFields?: string[];
		onBrushSelect?: (start: Date, end: Date) => void;
	} = $props();

	let selectedFields = $state<string[]>(initialFields ?? measurementFields.slice(0, MAX_AUTO_FIELDS));

	function toggleField(field: string, multi: boolean) {
		if (multi) {
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

<div class="mb-2 flex flex-wrap gap-1">
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

<TlmChart
	{data}
	{selectedFields}
	{chartConfig}
	{useUtc}
	start={queryStart}
	end={queryEnd}
	{chartClass}
	{onBrushSelect}
/>
