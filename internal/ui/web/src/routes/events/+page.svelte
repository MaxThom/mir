<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { eventStore } from '$lib/domains/events/stores/event.svelte';
	import { EventDataTable } from '$lib/domains/events/components/event-table';

	$effect(() => {
		if (mirStore.mir) {
			eventStore.reset();
			eventStore.loadAllEvents(mirStore.mir, 500);
		} else {
			eventStore.reset();
		}
	});
</script>

<EventDataTable
	events={eventStore.events}
	isLoading={eventStore.isLoading}
	hasLoaded={eventStore.hasLoaded}
	error={eventStore.error}
	onrefetch={(from, to) => {
		if (mirStore.mir) {
			eventStore.loadAllEvents(mirStore.mir, 500, from, to);
		}
	}}
/>
