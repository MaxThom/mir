import type { Mir } from '@mir/sdk';
import { DeviceTarget } from '@mir/sdk';
import type { CommandGroup, CommandResponse, SendCommandResult } from '@mir/sdk';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

class CommandStore {
	commands = $state<CommandGroup[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	isSending = $state(false);
	sendError = $state<string | null>(null);
	response = $state<SendCommandResult | null>(null);

	async loadCommands(mir: Mir, deviceId: string) {
		this.isLoading = true;
		this.error = null;

		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir.client().listCommands().request(target);
			this.commands = result;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			this.isLoading = false;
		}
	}

	async sendCommand(
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
			const result = await mir.client().sendCommand().request(target, name, payload, dryRun);
			this.response = result;
			activityStore.add({
				kind: 'success',
				category: 'Command',
				title: name,
				request: { deviceId, name, payload, dryRun },
				response: Object.fromEntries(result)
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to send command';
			this.sendError = message;
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: name,
				request: { deviceId, name, payload, dryRun },
				error: message
			});
		} finally {
			this.isSending = false;
		}
	}

	reset() {
		this.response = null;
		this.sendError = null;
	}
}

export const commandStore = new CommandStore();
