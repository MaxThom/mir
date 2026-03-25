<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { EventDataTable } from '$lib/domains/events/components/event-table';
	import { EventTarget, DateFilter, MirEvent } from '@mir/sdk';
	import type { EventsWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';

	let { config, refreshTick = 0 }: { config: EventsWidgetConfig; refreshTick?: number } = $props();

	let events = $state<MirEvent[]>([]);
	let isLoading = $state(false);
	let hasLoaded = $state(false);
	let error = $state<string | null>(null);
	let isInRefresh = false;

	async function loadEvents(from?: Date, to?: Date) {
		const mir = mirStore.mir;
		if (!mir) return;
		isLoading = true;
		error = null;
		try {
			const target = new EventTarget({
				names: config.target.names ?? [],
				namespaces: config.target.namespaces ?? [],
				limit: config.limit ?? 50,
				dateFilter: new DateFilter({ from, to })
			});
			events = await mir.client().listEvents().request(target);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			isLoading = false;
			hasLoaded = true;
			if (isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
		}
	}

	$effect(() => {
		if (refreshTick > 0) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			loadEvents();
		}
	});

	$effect(() => {
		if (mirStore.mir) {
			events = [];
			hasLoaded = false;
			loadEvents();
		} else {
			events = [];
			hasLoaded = false;
		}
	});
</script>

<EventDataTable
	{events}
	{isLoading}
	{hasLoaded}
	{error}
	onrefetch={(from, to) => loadEvents(from, to)}
/>
