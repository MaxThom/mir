<script lang="ts">
	import { getContext } from 'svelte';
	import { Device } from '@mir/sdk';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { eventStore } from '$lib/domains/events/stores/event.svelte';
	import { DeviceDataCard, DevicePropertiesCard, DeviceEventsCard } from '$lib/domains/devices/components/device-detail';

	const ctx = getContext<{ device: Device | null }>('device');
	let device = $derived(ctx.device);

	$effect(() => {
		if (mirStore.mir && device?.spec?.deviceId) {
			eventStore.loadEvents(mirStore.mir, device.meta.name, device.meta.namespace);
		} else {
			eventStore.reset();
		}
	});
</script>

{#if device}
	<div class="flex flex-col gap-4">
		<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
			<DeviceDataCard {device} />
			<DevicePropertiesCard {device} />
		</div>
		<DeviceEventsCard />
	</div>
{/if}
