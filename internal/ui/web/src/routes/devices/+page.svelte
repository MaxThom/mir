<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';

	$effect(() => {
		if (mirStore.mir) {
			deviceStore.loadDevices(mirStore.mir);
		}
	});
</script>

<div class="flex flex-col gap-2 p-4">
	{#if deviceStore.isLoading}
		<p>Loading devices...</p>
	{:else if deviceStore.error}
		<p>Error: {deviceStore.error}</p>
	{:else}
		<ul>
			{#each deviceStore.devices as device (device?.spec?.deviceId)}
				<li>🛰️ {device.meta?.name}/{device.meta?.namespace}</li>
			{/each}
		</ul>
	{/if}
</div>
