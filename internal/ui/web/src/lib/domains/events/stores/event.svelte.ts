import type { Event } from '@mir/sdk';
import {
	ListEventsRequestSchema,
	ListEventsResponseSchema,
	EventTargetSchema,
	TargetsSchema
} from '@mir/sdk';
import { create, toBinary, fromBinary } from '@bufbuild/protobuf';
import type { Mir } from '@mir/sdk';

class EventStore {
	events = $state<Event[]>([]);
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
			const subject = `client.${mir.getInstanceName()}.evt.v1alpha.list`;
			const req = create(ListEventsRequestSchema, {
				target: create(EventTargetSchema, {
					targets: create(TargetsSchema, { names: [deviceName] }),
					filterLimit: 50
				})
			});
			const msg = await mir.request(subject, toBinary(ListEventsRequestSchema, req));
			if (id !== this.requestId) return;
			const response = fromBinary(ListEventsResponseSchema, msg.data);
			if (response.response.case === 'ok') {
				this.events = response.response.value.events;
			} else if (response.response.case === 'error') {
				throw new Error(response.response.value);
			}
		} catch (err) {
			if (id === this.requestId)
				this.error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			if (id === this.requestId) this.isLoading = false;
		}
	}
}

export const eventStore = new EventStore();
