import type { Mir } from '@mir/sdk';
import { Device, DeviceTarget } from '@mir/sdk';

class DeviceStore {
	devices = $state<Device[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	selectedDevice = $state<Device | null>(null);
	isLoadingDevice = $state(false);
	deviceError = $state<string | null>(null);

	isUpdating = $state(false);
	updateError = $state<string | null>(null);

	isDeleting = $state(false);
	deleteError = $state<string | null>(null);

	private requestId = 0;
	private deviceRequestId = 0;

	reset() {
		this.devices = [];
		this.isLoading = false;
		this.error = null;
	}

	resetDevice() {
		this.selectedDevice = null;
		this.isLoadingDevice = false;
		this.deviceError = null;
	}

	async loadDevice(mir: Mir, deviceId: string) {
		const id = ++this.deviceRequestId;
		this.isLoadingDevice = true;
		this.deviceError = null;

		try {
			const cached = this.devices.find((d) => d.spec.deviceId === deviceId);
			if (cached) {
				this.selectedDevice = cached;
				this.isLoadingDevice = false;
				return;
			}

			const devices = await mir.client().listDevices().request(new DeviceTarget(), false);

			if (id !== this.deviceRequestId) return;

			this.devices = devices;
			this.selectedDevice = devices.find((d) => d.spec.deviceId === deviceId) ?? null;
			if (!this.selectedDevice) this.deviceError = 'Device not found';
		} catch (err) {
			if (id === this.deviceRequestId) {
				this.deviceError = err instanceof Error ? err.message : 'Failed to load device';
			}
		} finally {
			if (id === this.deviceRequestId) {
				this.isLoadingDevice = false;
			}
		}
	}

	async updateDevice(mir: Mir, device: Device) {
		this.isUpdating = true;
		this.updateError = null;

		try {
			const updated = await mir.client().updateDevices().requestSingle(device);
			const first = updated[0];
			if (first) {
				this.selectedDevice = first;
				this.devices = this.devices.map((d) =>
					d.spec.deviceId === first.spec.deviceId ? first : d
				);
			}
		} catch (err) {
			this.updateError = err instanceof Error ? err.message : 'Failed to update device';
			throw err;
		} finally {
			this.isUpdating = false;
		}
	}

	async deleteDevice(mir: Mir, deviceId: string) {
		this.isDeleting = true;
		this.deleteError = null;

		try {
			await mir.client().deleteDevices().request(new DeviceTarget({ ids: [deviceId] }));
			this.devices = this.devices.filter((d) => d.spec.deviceId !== deviceId);
			this.selectedDevice = null;
		} catch (err) {
			this.deleteError = err instanceof Error ? err.message : 'Failed to delete device';
			throw err;
		} finally {
			this.isDeleting = false;
		}
	}

	async loadDevices(mir: Mir, { reset = false } = {}) {
		const id = ++this.requestId;

		this.isLoading = true;
		this.error = null;
		if (reset) this.devices = [];

		try {
			const devices = await mir.client().listDevices().request(new DeviceTarget(), false);

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
