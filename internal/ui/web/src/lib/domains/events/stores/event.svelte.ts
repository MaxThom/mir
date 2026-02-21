import type { Mir } from '@mir/sdk';
import { MirEvent, EventTarget } from '@mir/sdk';

class EventStore {
	events = $state<MirEvent[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);
	private requestId = 0;

	reset() {
		this.events = [];
		this.isLoading = false;
		this.error = null;
	}

	async loadEvents(mir: Mir, deviceName: string) {
		const id = ++this.requestId;
		this.isLoading = true;
		this.error = null;
		try {
			const target = new EventTarget({ names: [deviceName], limit: 50 });
			const events = await mir.client().listEvents().request(target);

			if (id !== this.requestId) return;
			this.events = events;
		} catch (err) {
			if (id === this.requestId)
				this.error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			if (id === this.requestId) this.isLoading = false;
		}
	}
}

export const eventStore = new EventStore();
