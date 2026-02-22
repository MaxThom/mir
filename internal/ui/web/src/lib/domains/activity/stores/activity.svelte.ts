export type ActivityKind = 'success' | 'error' | 'info';
export type ActivityCategory = 'Device' | 'Command' | 'Config' | 'Connection';

export type ActivityEntry = {
	id: string;
	timestamp: Date;
	kind: ActivityKind;
	category: ActivityCategory;
	title: string;
	request?: unknown;
	response?: unknown;
	error?: string;
};

class ActivityStore {
	entries = $state<ActivityEntry[]>([]);

	add(entry: Omit<ActivityEntry, 'id' | 'timestamp'>) {
		this.entries.unshift({ id: crypto.randomUUID(), timestamp: new Date(), ...entry });
	}

	clear() {
		this.entries = [];
	}
}

export const activityStore = new ActivityStore();
