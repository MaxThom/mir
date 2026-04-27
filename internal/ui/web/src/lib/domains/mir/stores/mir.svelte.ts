import { Mir, credsAuthenticator } from '@mir/sdk';
import type { Context } from '../../contexts/types/types';
import { contextService } from '../../contexts/services/contexts';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

// Returns the WebSocket URL for a context.
// Uses ctx.webTarget if set; otherwise derives ws[s]://host:9222 from ctx.target.
// Uses wss:// when the page is served over HTTPS.
function toWsUrl(ctx: Context): string {
	if (ctx.webTarget) return ctx.webTarget;
	const scheme =
		typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss' : 'ws';
	return ctx.target.replace(/^nats:\/\//, `${scheme}://`).replace(/:\d+$/, ':9222');
}

class MirStore {
	mir = $state<Mir | null>(null);
	isConnecting = $state(false);
	error = $state<string | null>(null);

	private connectionId = 0;

	get isConnected(): boolean {
		return this.mir !== null;
	}

	async connect(ctx: Context, password?: string | null) {
		const id = ++this.connectionId;

		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}

		this.isConnecting = true;
		this.error = null;

		try {
			const wsUrl = toWsUrl(ctx);

			const creds = await contextService.getCredentials(ctx.name, password ?? null);

			const opts = creds
				? {
						maxReconnectAttempts: 0,
						authenticator: credsAuthenticator(new TextEncoder().encode(creds))
					}
				: { maxReconnectAttempts: 0 };

			const mir = await Mir.connect('cockpit', wsUrl, opts);

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
