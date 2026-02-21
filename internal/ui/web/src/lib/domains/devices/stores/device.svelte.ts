import type { Device } from '@mir/sdk';
import {
	DeviceTargetSchema,
	UpdateDeviceRequestSchema,
	UpdateDeviceResponseSchema,
	type UpdateDeviceRequest
} from '@mir/sdk';
import { create, toBinary, fromBinary } from '@bufbuild/protobuf';
import type { Mir } from '@mir/sdk';

class DeviceStore {
	devices = $state<Device[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	selectedDevice = $state<Device | null>(null);
	isLoadingDevice = $state(false);
	deviceError = $state<string | null>(null);

	isUpdating = $state(false);
	updateError = $state<string | null>(null);

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
			const cached = this.devices.find((d) => d.spec?.deviceId === deviceId);
			if (cached) {
				this.selectedDevice = cached;
				this.isLoadingDevice = false;
				return;
			}

			const target = create(DeviceTargetSchema, {});
			const devices = await mir.client().listDevices().request(target, false);

			if (id !== this.deviceRequestId) return;

			this.devices = devices;
			this.selectedDevice = devices.find((d) => d.spec?.deviceId === deviceId) ?? null;
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

	async updateDevice(mir: Mir, request: UpdateDeviceRequest) {
		this.isUpdating = true;
		this.updateError = null;

		try {
			const subject = `client.${mir.getInstanceName()}.core.v1alpha.update`;
			const payload = toBinary(UpdateDeviceRequestSchema, request);
			const msg = await mir.request(subject, payload);
			const response = fromBinary(UpdateDeviceResponseSchema, msg.data);

			if (response.response.case === 'ok') {
				const updated = response.response.value.devices[0];
				if (updated) {
					this.selectedDevice = updated;
					this.devices = this.devices.map((d) =>
						d.spec?.deviceId === updated.spec?.deviceId ? updated : d
					);
				}
			} else if (response.response.case === 'error') {
				throw new Error(response.response.value);
			}
		} catch (err) {
			this.updateError = err instanceof Error ? err.message : 'Failed to update device';
			throw err;
		} finally {
			this.isUpdating = false;
		}
	}

	async loadDevices(mir: Mir, { reset = false } = {}) {
		const id = ++this.requestId;

		this.isLoading = true;
		this.error = null;
		if (reset) this.devices = [];

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
