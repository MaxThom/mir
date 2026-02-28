<script lang="ts">
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import { ChartContainer, ChartTooltip } from '$lib/shared/components/shadcn/chart';
	import { LineChart } from 'layerchart';
	import { scaleUtc, scaleTime } from 'd3-scale';
	import type { QueryData } from '@mir/sdk';

	let {
		data,
		selectedFields,
		chartConfig,
		useUtc = false
	}: {
		data: QueryData;
		selectedFields: string[];
		chartConfig: ChartConfig;
		useUtc?: boolean;
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

	const tz = $derived(useUtc ? 'UTC' : undefined);

	// X-axis tick labels: just HH:MM(:SS if seconds vary)
	function formatAxisTime(d: Date): string {
		return d.toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
			hour12: false,
			timeZone: tz
		});
	}

	// Tooltip header: "Feb 27 · 14:23:45 UTC" or "Feb 27 · 14:23:45"
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	function formatTooltipLabel(value: any): string {
		if (!(value instanceof Date)) return String(value);
		const date = value.toLocaleDateString([], {
			month: 'short',
			day: 'numeric',
			timeZone: tz
		});
		const time = value.toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			hour12: false,
			timeZone: tz
		});
		return useUtc ? `${date} · ${time} UTC` : `${date} · ${time}`;
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
			xScale={useUtc ? scaleUtc() : scaleTime()}
			{series}
			{yDomain}
			padding={{ top: 8, right: 16, bottom: 32, left: 56 }}
			props={{
				spline: { strokeWidth: 2 },
				xAxis: { format: formatAxisTime, tickSpacing: 100 }
			}}
		>
			{#snippet tooltip()}
				<ChartTooltip labelFormatter={formatTooltipLabel} />
			{/snippet}
		</LineChart>
	</ChartContainer>
{/if}
