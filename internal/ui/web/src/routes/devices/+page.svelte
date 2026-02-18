<script lang="ts">
	import type { Device } from '@mir/sdk';

	let devices = $state<Device[]>([
		{
			apiVersion: 'v1alpha',
			kind: 'Device',
			meta: {
				name: 'sensor-01',
				namespace: 'default'
			},
			spec: {
				deviceId: 'sensor-01',
				disabled: false
			},
			status: {
				online: true
			}
		} as unknown as Device
	]);
</script>

<div class="flex flex-col gap-2 p-4">
	<ul>
		{#each devices as device (device?.spec?.deviceId)}
			<li>🛰️ {device.meta?.name}/{device.meta?.namespace}</li>
		{/each}
	</ul>
</div>
