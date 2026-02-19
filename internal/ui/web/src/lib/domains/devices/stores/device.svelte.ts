import type { Device } from '@mir/sdk';
import { DeviceTargetSchema } from '@mir/sdk';
import { create } from '@bufbuild/protobuf';
import type { Mir } from '@mir/sdk';

class DeviceStore {
	devices = $state<Device[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	private requestId = 0;

	async loadDevices(mir: Mir) {
		const id = ++this.requestId;

		this.isLoading = true;
		this.error = null;
		this.devices = [];

		try {
			const target = create(DeviceTargetSchema, {});
			const devices = await mir.client().listDevices().request(target, false);

			if (id !== this.requestId) return;

			this.devices = devices;
		} catch (err) {
			if (id === this.requestId) {
				this.error = err instanceof Error ? err.message : 'Failed to load devices';
			}
		} finally {
			if (id === this.requestId) {
				this.isLoading = false;
			}
		}
	}
}

export const deviceStore = new DeviceStore();
