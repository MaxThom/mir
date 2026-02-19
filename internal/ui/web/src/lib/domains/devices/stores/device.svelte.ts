import type { Device } from '@mir/sdk';
import { DeviceTargetSchema } from '@mir/sdk';
import { create } from '@bufbuild/protobuf';
import type { Mir } from '@mir/sdk';

class DeviceStore {
	devices = $state<Device[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	async loadDevices(mir: Mir) {
		this.isLoading = true;
		this.error = null;
		try {
			const target = create(DeviceTargetSchema, {});
			this.devices = await mir.client().listDevices().request(target, false);
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load devices';
		} finally {
			this.isLoading = false;
		}
	}
}

export const deviceStore = new DeviceStore();
