import { toast } from 'svelte-sonner';

export type ActivityKind = 'success' | 'error' | 'info';
export type ActivityCategory = 'Device' | 'Command' | 'Config' | 'Connection' | 'Telemetry' | 'Dashboard';

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

		const title = `${entry.category} · ${entry.title}`;
		const opts = entry.error ? { description: entry.error } : undefined;
		if (entry.kind === 'success') toast.success(title, opts);
		else if (entry.kind === 'error') toast.error(title, opts);
		else toast(title, opts);
	}

	clear() {
		this.entries = [];
	}
}

export const activityStore = new ActivityStore();
