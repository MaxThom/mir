import type { Device } from '@mir/sdk';

class SelectionStore {
	selectedDevices = $state<Device[]>([]);

	select(device: Device) {
		if (!this.isSelected(device.spec.deviceId)) {
			this.selectedDevices = [...this.selectedDevices, device];
		}
	}

	deselect(deviceId: string) {
		this.selectedDevices = this.selectedDevices.filter((d) => d.spec.deviceId !== deviceId);
	}

	setAll(devices: Device[]) {
		this.selectedDevices = [...devices];
	}

	clearSelection() {
		this.selectedDevices = [];
	}

	reset() {
		this.clearSelection();
	}

	isSelected(deviceId: string): boolean {
		return this.selectedDevices.some((d) => d.spec.deviceId === deviceId);
	}

	get count() {
		return this.selectedDevices.length;
	}
}

export const selectionStore = new SelectionStore();
