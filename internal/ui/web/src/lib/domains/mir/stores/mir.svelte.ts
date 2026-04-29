import { Mir, credsAuthenticator } from '@mir/sdk';
import type { Context } from '../../contexts/types/types';
import { contextService } from '../../contexts/services/contexts';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

class MirStore {
	mir = $state<Mir | null>(null);
	isConnecting = $state(false);
	error = $state<string | null>(null);

	private connectionId = 0;

	get isConnected(): boolean {
		return this.mir !== null;
	}

	// preloadedCreds: pass already-fetched creds to avoid a second round-trip (e.g. from the password dialog).
	// undefined = fetch now using password, null = no creds (already confirmed by caller), string = use directly.
	async connect(ctx: Context, password?: string | null, preloadedCreds?: string | null) {
		const id = ++this.connectionId;

		if (this.mir) {
			await this.mir.disconnect();
			this.mir = null;
		}

		this.isConnecting = true;
		this.error = null;

		try {
			const wsUrl = ctx.target;

			const creds = preloadedCreds !== undefined
				? preloadedCreds
				: await contextService.getCredentials(ctx.name, password ?? null);

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
