<script lang="ts">
	import { getContext, onDestroy } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { eventStore } from '$lib/domains/events/stores/event.svelte';
	import { EventDataTable } from '$lib/domains/events/components/event-table';
	import type { Device } from '@mir/sdk';

	const deviceCtx = getContext<{
		device: Device | null;
		setTabRefresh: (fn: (() => void) | null) => void;
	}>('device');

	let device = $derived(deviceCtx.device);

	// Derive stable primitives so the $effect only re-runs when the device identity
	// actually changes, not when the device object gets a new reference (e.g. after
	// the layout force-refreshes the device on the refresh button click).
	let deviceName = $derived(device?.meta?.name ?? '');
	let deviceNamespace = $derived(device?.meta?.namespace ?? '');

	deviceCtx.setTabRefresh(() => {
		if (mirStore.mir && deviceName) {
			eventStore.loadEvents(mirStore.mir, deviceName, deviceNamespace, 200);
		}
	});
	onDestroy(() => deviceCtx.setTabRefresh(null));

	$effect(() => {
		if (mirStore.mir && deviceName) {
			eventStore.reset();
			eventStore.loadEvents(mirStore.mir, deviceName, deviceNamespace, 200);
		}
	});
</script>

<EventDataTable
	events={eventStore.events}
	isLoading={eventStore.isLoading}
	error={eventStore.error}
/>
