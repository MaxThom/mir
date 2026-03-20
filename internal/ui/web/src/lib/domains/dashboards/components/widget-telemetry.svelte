<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
	import { DeviceTarget } from '@mir/sdk';
	import type { QueryData } from '@mir/sdk';
	import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import type { TelemetryWidgetConfig } from '../api/dashboard-api';

	let { config }: { config: TelemetryWidgetConfig } = $props();

	let queryData = $state<QueryData | null>(null);
	let isQuerying = $state(false);
	let error = $state<string | null>(null);

	let chartConfig = $derived.by<ChartConfig>(() => {
		const cfg: ChartConfig = {};
		config.fields.forEach((f, i) => {
			cfg[f] = { label: f, color: CHART_COLORS[i % CHART_COLORS.length] };
		});
		return cfg;
	});

	async function query() {
		const mir = mirStore.mir;
		if (!mir || !config.measurement) return;
		isQuerying = true;
		error = null;
		try {
			const minutes = config.timeMinutes ?? 60;
			const end = new Date();
			const start = new Date(end.getTime() - minutes * 60 * 1000);
			const target = new DeviceTarget({ ids: config.target.ids ?? [] });
			queryData = await mir
				.client()
				.queryTelemetry()
				.request(target, config.measurement, config.fields, start, end);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to query telemetry';
		} finally {
			isQuerying = false;
		}
	}

	$effect(() => {
		if (mirStore.mir) {
			query();
		} else {
			queryData = null;
		}
	});
</script>

{#if isQuerying && !queryData}
	<div class="flex h-full items-center justify-center">
		<Spinner />
	</div>
{:else if error}
	<p class="text-destructive text-xs">{error}</p>
{:else if queryData}
	<TlmChart
		data={queryData}
		selectedFields={config.fields}
		{chartConfig}
		chartClass="h-full"
	/>
{:else}
	<p class="text-muted-foreground text-xs">No data</p>
{/if}
