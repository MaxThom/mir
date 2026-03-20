import {
	dashboardApi,
	dashboardKey,
	type Dashboard,
	type Widget,
	type WidgetConfig,
	type WidgetType
} from '../api/dashboard-api';

class DashboardStore {
	dashboards      = $state<Dashboard[]>([]);
	activeDashboard = $state<Dashboard | null>(null);
	isLoading       = $state(false);
	isSaving        = $state(false);
	editMode        = $state(false);
	error           = $state<string | null>(null);

	private saveTimer: ReturnType<typeof setTimeout> | null = null;

	async load() {
		this.isLoading = true;
		this.error = null;
		try {
			this.dashboards = (await dashboardApi.list()) ?? [];
			const savedKey = localStorage.getItem('mir_active_dashboard_key');
			const found = savedKey ? this.dashboards.find((d) => dashboardKey(d) === savedKey) : null;
			this.activeDashboard = found ?? this.dashboards[0] ?? null;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load dashboards';
		} finally {
			this.isLoading = false;
		}
	}

	setActive(d: Dashboard) {
		this.activeDashboard = d;
		localStorage.setItem('mir_active_dashboard_key', dashboardKey(d));
	}

	async create(name: string, namespace = 'default', description = '') {
		this.isSaving = true;
		try {
			const d = await dashboardApi.create(name, namespace, description);
			this.dashboards = [...this.dashboards, d];
			this.setActive(d);
			return d;
		} finally {
			this.isSaving = false;
		}
	}

	async rename(d: Dashboard, name: string) {
		this.isSaving = true;
		try {
			const updated = await dashboardApi.update(d.meta.namespace, name, d.spec.description);
			this._syncDashboard(d, updated);
			return updated;
		} finally {
			this.isSaving = false;
		}
	}

	async remove(d: Dashboard) {
		this.isSaving = true;
		try {
			await dashboardApi.delete(d.meta.namespace, d.meta.name);
			const key = dashboardKey(d);
			this.dashboards = this.dashboards.filter((x) => dashboardKey(x) !== key);
			if (this.activeDashboard && dashboardKey(this.activeDashboard) === key) {
				this.activeDashboard = this.dashboards[0] ?? null;
				if (this.activeDashboard) {
					localStorage.setItem('mir_active_dashboard_key', dashboardKey(this.activeDashboard));
				} else {
					localStorage.removeItem('mir_active_dashboard_key');
				}
			}
		} finally {
			this.isSaving = false;
		}
	}

	async addWidget(d: Dashboard, type: WidgetType, title: string, config: WidgetConfig) {
		const newWidget: Widget = {
			id: crypto.randomUUID(),
			type, title, config,
			x: 0, y: 0, w: 4, h: 4
		};
		return this._persistWidgets(d, [...(d.spec.widgets ?? []), newWidget]);
	}

	async removeWidget(d: Dashboard, widgetId: string) {
		return this._persistWidgets(d, d.spec.widgets.filter((w) => w.id !== widgetId));
	}

	saveLayout(d: Dashboard, layoutItems: Pick<Widget, 'id' | 'x' | 'y' | 'w' | 'h'>[]) {
		const posMap = new Map(layoutItems.map((item) => [item.id, item]));
		const updated = (d.spec.widgets ?? []).map((w) => {
			const pos = posMap.get(w.id);
			return pos ? { ...w, ...pos } : w;
		});
		// Optimistic update so the grid doesn't flicker during drag.
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });

		if (this.saveTimer) clearTimeout(this.saveTimer);
		this.saveTimer = setTimeout(
			() => this._persistWidgets({ ...d, spec: { ...d.spec, widgets: updated } }, updated),
			1000
		);
	}

	toggleEditMode() {
		this.editMode = !this.editMode;
	}

	private async _persistWidgets(d: Dashboard, widgets: Widget[]) {
		this.isSaving = true;
		try {
			const updated = await dashboardApi.saveWidgets(d.meta.namespace, d.meta.name, widgets);
			this._syncDashboard(d, updated);
			return updated;
		} finally {
			this.isSaving = false;
		}
	}

	private _syncDashboard(old: Dashboard, updated: Dashboard) {
		const key = dashboardKey(old);
		this.dashboards = this.dashboards.map((x) => (dashboardKey(x) === key ? updated : x));
		if (this.activeDashboard && dashboardKey(this.activeDashboard) === key) {
			this.activeDashboard = updated;
		}
	}
}

export const dashboardStore = new DashboardStore();
