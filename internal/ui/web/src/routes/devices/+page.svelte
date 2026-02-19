<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import DeviceDataTable from '$lib/domains/devices/components/device-data-table.svelte';

	$effect(() => {
		if (mirStore.mir) {
			deviceStore.loadDevices(mirStore.mir);
		}
	});

	let isLoading = $derived(!mirStore.mir || mirStore.isConnecting || deviceStore.isLoading);
</script>

<div class="flex flex-col gap-4 p-4">
	<DeviceDataTable
		devices={deviceStore.devices}
		{isLoading}
		error={deviceStore.error}
	/>
</div>
