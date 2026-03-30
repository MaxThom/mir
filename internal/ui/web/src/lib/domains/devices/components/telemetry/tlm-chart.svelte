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
		useUtc = false,
		start = null,
		end = null,
		chartClass = 'h-72',
		onBrushSelect
	}: {
		data: QueryData;
		selectedFields: string[];
		chartConfig: ChartConfig;
		useUtc?: boolean;
		start?: Date | null;
		end?: Date | null;
		chartClass?: string;
		onBrushSelect?: (start: Date, end: Date) => void;
	} = $props();

	// Flat chart rows: { _time: Date, __id: string, [field]: value }
	// Gap sentinel rows ({ __gap: true }) are inserted where consecutive points are
	// more than 3× the median interval apart so the line breaks instead of connecting.
	let chartRows = $derived.by(() => {
		const rows = data.rows.map((row) => ({ ...row.values }));
		if (rows.length < 2) return rows;

		const intervals: number[] = [];
		for (let i = 1; i < rows.length; i++) {
			intervals.push(
				(rows[i]._time as Date).getTime() - (rows[i - 1]._time as Date).getTime()
			);
		}
		const sorted = [...intervals].sort((a, b) => a - b);
		const median = sorted[Math.floor(sorted.length / 2)];
		const threshold = median * 3;

		const result: (typeof rows)[number][] = [];
		for (let i = 0; i < rows.length; i++) {
			result.push(rows[i]);
			if (i < rows.length - 1 && intervals[i] > threshold) {
				const mid = ((rows[i]._time as Date).getTime() + (rows[i + 1]._time as Date).getTime()) / 2;
				result.push({ _time: new Date(mid), __gap: true } as (typeof rows)[number]);
			}
		}
		return result;
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

	const rangeHours = $derived(
		start && end ? (end.getTime() - start.getTime()) / (1000 * 60 * 60) : 0
	);

	// X-axis tick labels: range-aware format
	//   ≤ 6 h  → "14:30"
	//   ≤ 48 h → "Jan 15 14:30"
	//   > 48 h → "Jan 15"
	const formatAxisTime = $derived((d: Date): string => {
		if (rangeHours > 48) {
			return d.toLocaleDateString([], { month: 'short', day: 'numeric', timeZone: tz });
		}
		const time = d.toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
			hour12: false,
			timeZone: tz
		});
		if (rangeHours > 6) {
			const date = d.toLocaleDateString([], { month: 'short', day: 'numeric', timeZone: tz });
			return `${date} ${time}`;
		}
		return time;
	});

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

<div class="relative {chartClass}">
	<ChartContainer config={chartConfig} class="h-full w-full">
		<LineChart
			data={chartRows}
			x="_time"
			xScale={useUtc ? scaleUtc() : scaleTime()}
			xDomain={start && end ? [start, end] : undefined}
			{series}
			{yDomain}
			padding={{ top: 8, right: 16, bottom: 32, left: 56 }}
			brush={{
				axis: 'x',
				resetOnEnd: true,
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				onBrushEnd: (detail: any) => {
					const [s, e] = detail.xDomain as [Date, Date];
					if (s && e && s.getTime() < e.getTime()) onBrushSelect?.(s, e);
				}
			}}
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
</div>
