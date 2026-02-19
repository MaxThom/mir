import { Mir } from '@mir/sdk';
import type { Context } from '../../contexts/types/types';

// Converts "nats://host:port" → "ws://host:9222"
function toWsUrl(natsTarget: string): string {
	return natsTarget.replace(/^nats:\/\//, 'ws://').replace(/:\d+$/, ':9222');
}

class MirStore {
	mir = $state<Mir | null>(null);
	isConnecting = $state(false);
	error = $state<string | null>(null);

	get isConnected(): boolean {
		return this.mir !== null;
	}

	async connect(ctx: Context) {
		if (this.isConnecting) return;
		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}
		this.isConnecting = true;
		this.error = null;
		try {
			const wsUrl = toWsUrl(ctx.target);
			this.mir = await Mir.connect('cockpit', wsUrl, {
				maxReconnectAttempts: 0
			});
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Connection failed';
		} finally {
			this.isConnecting = false;
		}
	}

	async disconnect() {
		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}
	}
}

export const mirStore = new MirStore();
