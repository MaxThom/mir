export type WidgetType = 'telemetry' | 'command' | 'config' | 'events';

export interface DeviceTargetConfig {
	ids?: string[];
	names?: string[];
	namespaces?: string[];
	labels?: Record<string, string>;
}

export interface TelemetryWidgetConfig {
	target: DeviceTargetConfig;
	measurement: string;
	fields: string[];
	timeMinutes: number;
	// View state (optional — absent in older saved dashboards, defaults applied on load)
	selectedFields?: string[];
	splitCount?: 1 | 2 | 3 | 4;
	syncFields?: boolean;
	enabledDeviceIds?: string[];
}

export interface CommandWidgetConfig {
	target: DeviceTargetConfig;
	selectedCommand?: string;
}

export interface ConfigWidgetConfig {
	target: DeviceTargetConfig;
}

export interface EventsWidgetConfig {
	target: DeviceTargetConfig;
	limit: number;
}

export type WidgetConfig =
	| TelemetryWidgetConfig
	| CommandWidgetConfig
	| ConfigWidgetConfig
	| EventsWidgetConfig;

export interface Widget {
	id: string;
	type: WidgetType;
	title: string;
	x: number;
	y: number;
	w: number;
	h: number;
	config: WidgetConfig;
}

export interface DashboardMeta {
	name: string;
	namespace: string;
	labels?: Record<string, string>;
	annotations?: Record<string, string>;
}

export interface DashboardSpec {
	description: string;
	refreshInterval?: number;
	timeMinutes?: number;
	widgets: Widget[];
	createdAt: string;
	updatedAt: string;
}

export interface Dashboard {
	apiVersion: string;
	kind: string;
	meta: DashboardMeta;
	spec: DashboardSpec;
}

/** Stable path key for a dashboard: "{namespace}/{name}" */
export function dashboardKey(d: Dashboard): string {
	return `${d.meta.namespace}/${d.meta.name}`;
}

const BASE = '/api/v1/dashboards';

async function request<T>(url: string, init?: RequestInit): Promise<T> {
	const res = await fetch(url, init);
	if (!res.ok) {
		const text = await res.text();
		throw new Error(text || res.statusText);
	}
	return res.json() as Promise<T>;
}

export const dashboardApi = {
	list: (): Promise<Dashboard[]> =>
		request<Dashboard[]>(BASE),

	get: (namespace: string, name: string): Promise<Dashboard> =>
		request<Dashboard>(`${BASE}/${namespace}/${name}`),

	create: (name: string, namespace = 'default', description = ''): Promise<Dashboard> =>
		request<Dashboard>(BASE, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				apiVersion: 'mir/v1alpha',
				kind: 'dashboard',
				meta: { name, namespace },
				spec: { description, widgets: [] }
			})
		}),

	update: (
		namespace: string,
		name: string,
		patch: { name?: string; namespace?: string; description?: string }
	): Promise<Dashboard> =>
		request<Dashboard>(`${BASE}/${namespace}/${name}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				...(patch.name || patch.namespace
					? {
							meta: {
								...(patch.name ? { name: patch.name } : {}),
								...(patch.namespace ? { namespace: patch.namespace } : {})
							}
						}
					: {}),
				spec: { description: patch.description ?? '' }
			})
		}),

	delete: (namespace: string, name: string): Promise<Dashboard> =>
		request<Dashboard>(`${BASE}/${namespace}/${name}`, { method: 'DELETE' }),

	saveWidgets: (namespace: string, name: string, widgets: Widget[], refreshInterval?: number, timeMinutes?: number): Promise<Dashboard> =>
		request<Dashboard>(`${BASE}/${namespace}/${name}/widgets`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				widgets,
				...(refreshInterval !== undefined ? { refreshInterval } : {}),
				...(timeMinutes !== undefined ? { timeMinutes } : {})
			})
		})
};
