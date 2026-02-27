<script lang="ts">
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import { ChartContainer, ChartTooltip } from '$lib/shared/components/shadcn/chart';
	import { LineChart } from 'layerchart';
	import { scaleUtc } from 'd3-scale';
	import type { QueryData } from '@mir/sdk';

	let {
		data,
		selectedFields,
		chartConfig
	}: {
		data: QueryData;
		selectedFields: string[];
		chartConfig: ChartConfig;
	} = $props();

	// Flat chart rows: { _time: Date, __id: string, [field]: value }
	let chartRows = $derived.by(() => {
		return data.rows.map((row) => ({ ...row.values }));
	});

	// Y-domain spanning all selected numeric fields
	let yDomain = $derived.by(() => {
		if (!chartRows.length || !selectedFields.length) return [0, 1];
		const values = selectedFields.flatMap((f) =>
			chartRows
				.map((r) => {
					const v = (r as Record<string, unknown>)[f];
					return typeof v === 'number' ? v : null;
				})
				.filter((v): v is number => v !== null)
		);
		if (!values.length) return [0, 1];
		const min = Math.min(0, ...values);
		const max = Math.max(...values);
		return [min, max === min ? max + 1 : max];
	});

	let series = $derived(
		selectedFields.map((field) => ({
			key: field,
			label: chartConfig[field]?.label ?? field,
			color: chartConfig[field]?.color ?? 'hsl(var(--chart-1))'
		}))
	);

	function formatTime(d: Date): string {
		return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}
</script>

{#if !chartRows.length}
	<div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
		No data in this time range.
	</div>
{:else}
	<ChartContainer config={chartConfig} class="h-72 w-full">
		<LineChart
			data={chartRows}
			x="_time"
			xScale={scaleUtc()}
			{series}
			{yDomain}
			padding={{ top: 8, right: 16, bottom: 32, left: 56 }}
			props={{
				spline: { strokeWidth: 2 },
				xAxis: { format: formatTime, tickSpacing: 100 }
			}}
		>
			{#snippet tooltip()}
				<ChartTooltip />
			{/snippet}
		</LineChart>
	</ChartContainer>
{/if}
