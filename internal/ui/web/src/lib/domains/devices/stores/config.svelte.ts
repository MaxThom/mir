import type { Mir } from '@mir/sdk';
import { DeviceTarget } from '@mir/sdk';
import type { ConfigGroup, SendConfigResult } from '@mir/sdk';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

class ConfigStore {
	configs = $state<ConfigGroup[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	isSending = $state(false);
	sendError = $state<string | null>(null);
	response = $state<SendConfigResult | null>(null);

	async loadConfigs(mir: Mir, deviceId: string) {
		this.isLoading = true;
		this.error = null;

		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir.client().listConfigs().request(target);
			this.configs = result;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load configs';
		} finally {
			this.isLoading = false;
		}
	}

	async sendConfig(
		mir: Mir,
		deviceId: string,
		name: string,
		payload: string,
		dryRun: boolean
	) {
		this.isSending = true;
		this.sendError = null;

		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir.client().sendConfig().request(target, name, payload, dryRun);
			this.response = result;
			activityStore.add({
				kind: 'success',
				category: 'Config',
				title: name,
				request: { deviceId, name, payload, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send config';
			this.sendError = message;
			activityStore.add({
				kind: 'error',
				category: 'Config',
				title: name,
				request: { deviceId, name, payload, dryRun },
				error: message
			});
		} finally {
			this.isSending = false;
		}
	}

	clearResponse() {
		this.response = null;
		this.sendError = null;
	}

	reset() {
		this.clearResponse();
		this.configs = [];
	}
}

export const configStore = new ConfigStore();
