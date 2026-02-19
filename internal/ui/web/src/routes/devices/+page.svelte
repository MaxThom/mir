<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import DeviceDataTable from '$lib/domains/devices/components/device-table/device-data-table.svelte';

	$effect(() => {
		if (mirStore.mir) {
			deviceStore.loadDevices(mirStore.mir, { reset: true });
		} else {
			deviceStore.reset();
		}
	});

	let isLoading = $derived(!mirStore.mir || mirStore.isConnecting || deviceStore.isLoading);

	function handleRefresh() {
		if (mirStore.mir) deviceStore.loadDevices(mirStore.mir);
	}
</script>

<div class="flex flex-col gap-4 p-4">
	<DeviceDataTable
		devices={deviceStore.devices}
		{isLoading}
		error={deviceStore.error}
		onRefresh={handleRefresh}
	/>
</div>
