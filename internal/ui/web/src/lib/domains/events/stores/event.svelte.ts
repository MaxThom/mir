import type { Mir } from '@mir/sdk';
import { MirEvent, EventTarget, DateFilter } from '@mir/sdk';

class EventStore {
	events = $state<MirEvent[]>([]);
	isLoading = $state(false);
	hasLoaded = $state(false);
	error = $state<string | null>(null);
	lastFrom = $state<Date | undefined>(undefined);
	lastTo = $state<Date | undefined>(undefined);
	private requestId = 0;

	reset() {
		this.events = [];
		this.isLoading = false;
		this.hasLoaded = false;
		this.error = null;
		this.lastFrom = undefined;
		this.lastTo = undefined;
	}

	async loadAllEvents(mir: Mir, limit = 200, from?: Date, to?: Date) {
		this.lastFrom = from;
		this.lastTo = to;
		const id = ++this.requestId;
		this.isLoading = true;
		this.error = null;
		try {
			const target = new EventTarget({
				names: [],
				namespaces: [],
				limit,
				dateFilter: new DateFilter({ from, to })
			});
			const events = await mir.client().listEvents().request(target);
			if (id !== this.requestId) return;
			this.events = events;
		} catch (err) {
			if (id === this.requestId)
				this.error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			if (id === this.requestId) {
				this.hasLoaded = true;
				this.isLoading = false;
			}
		}
	}

	async loadEvents(mir: Mir, name: string, namespace: string, limit = 50, from?: Date, to?: Date) {
		this.lastFrom = from;
		this.lastTo = to;
		const id = ++this.requestId;
		this.isLoading = true;
		this.error = null;
		try {
			const target = new EventTarget({
				names: [name + '-*'],
				namespaces: [namespace],
				limit,
				dateFilter: new DateFilter({ from, to })
			});
			const events = await mir.client().listEvents().request(target);

			if (id !== this.requestId) return;
			this.events = events;
		} catch (err) {
			if (id === this.requestId)
				this.error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			if (id === this.requestId) {
				this.hasLoaded = true;
				this.isLoading = false;
			}
		}
	}
}

export const eventStore = new EventStore();
