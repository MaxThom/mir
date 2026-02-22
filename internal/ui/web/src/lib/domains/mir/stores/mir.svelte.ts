import { Mir } from '@mir/sdk';
import type { Context } from '../../contexts/types/types';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

// Converts "nats://host:port" → "ws://host:9222"
function toWsUrl(natsTarget: string): string {
	return natsTarget.replace(/^nats:\/\//, 'ws://').replace(/:\d+$/, ':9222');
}

class MirStore {
	mir = $state<Mir | null>(null);
	isConnecting = $state(false);
	error = $state<string | null>(null);

	private connectionId = 0;

	get isConnected(): boolean {
		return this.mir !== null;
	}

	async connect(ctx: Context) {
		const id = ++this.connectionId;

		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}

		this.isConnecting = true;
		this.error = null;

		try {
			const wsUrl = toWsUrl(ctx.target);
			const mir = await Mir.connect('cockpit', wsUrl, { maxReconnectAttempts: 0 });

			if (id !== this.connectionId) {
				await mir.disconnect();
				return;
			}

			this.mir = mir;
			activityStore.add({
				kind: 'success',
				category: 'Connection',
				title: 'Connected',
				request: { context: ctx.name }
			});
		} catch (err) {
			if (id === this.connectionId) {
				this.error = err instanceof Error ? err.message : 'Connection failed';
			}
			activityStore.add({
				kind: 'error',
				category: 'Connection',
				title: 'Connection Failed',
				error: err instanceof Error ? err.message : String(err)
			});
		} finally {
			if (id === this.connectionId) {
				this.isConnecting = false;
			}
		}
	}

	async disconnect() {
		++this.connectionId;
		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
			activityStore.add({ kind: 'info', category: 'Connection', title: 'Disconnected' });
		}
	}
}

export const mirStore = new MirStore();
