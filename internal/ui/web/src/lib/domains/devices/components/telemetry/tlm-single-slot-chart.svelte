<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import TlmChart from './tlm-chart.svelte';
	import TlmFieldToggles from './tlm-field-toggles.svelte';
	import { MAX_AUTO_FIELDS } from '$lib/domains/devices/utils/tlm-time';
	import { untrack } from 'svelte';

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

	let selectedFields = $state<string[]>(untrack(() => initialFields ?? measurementFields.slice(0, MAX_AUTO_FIELDS)));

	const fieldUnits = $derived(
		Object.fromEntries(measurementFields.map((f) => [f, chartConfig[f]?.unit ?? '']))
	);

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

<TlmFieldToggles
	fields={measurementFields}
	{selectedFields}
	ontoggle={toggleField}
	{fieldUnits}
	class="shrink-0 px-0 py-1.5"
/>

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
