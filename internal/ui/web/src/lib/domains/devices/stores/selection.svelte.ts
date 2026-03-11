import type { Device } from '@mir/sdk';

class SelectionStore {
	selectedDevices = $state<Device[]>([]);
	disabledDeviceIds = $state<Set<string>>(new Set());

	select(device: Device) {
		if (!this.isSelected(device.spec.deviceId)) {
			this.selectedDevices = [...this.selectedDevices, device];
		}
	}

	deselect(deviceId: string) {
		this.selectedDevices = this.selectedDevices.filter((d) => d.spec.deviceId !== deviceId);
		if (this.disabledDeviceIds.has(deviceId)) {
			this.disabledDeviceIds = new Set([...this.disabledDeviceIds].filter((id) => id !== deviceId));
		}
	}

	toggleDisabled(deviceId: string) {
		const next = new Set(this.disabledDeviceIds);
		if (next.has(deviceId)) {
			next.delete(deviceId);
		} else {
			next.add(deviceId);
		}
		this.disabledDeviceIds = next;
	}

	isDisabled(deviceId: string): boolean {
		return this.disabledDeviceIds.has(deviceId);
	}

	setAll(devices: Device[]) {
		this.selectedDevices = [...devices];
		this.disabledDeviceIds = new Set();
	}

	clearSelection() {
		this.selectedDevices = [];
		this.disabledDeviceIds = new Set();
	}

	reset() {
		this.clearSelection();
	}

	isSelected(deviceId: string): boolean {
		return this.selectedDevices.some((d) => d.spec.deviceId === deviceId);
	}

	get activeDevices(): Device[] {
		return this.selectedDevices.filter((d) => !this.disabledDeviceIds.has(d.spec.deviceId));
	}

	get count() {
		return this.selectedDevices.length;
	}

	get activeCount() {
		return this.selectedDevices.length - this.disabledDeviceIds.size;
	}
}

export const selectionStore = new SelectionStore();
