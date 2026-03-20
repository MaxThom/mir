import { SvelteMap } from 'svelte/reactivity';
import {
	dashboardApi,
	dashboardKey,
	type Dashboard,
	type Widget,
	type WidgetConfig,
	type WidgetType
} from '../api/dashboard-api';

class DashboardStore {
	dashboards = $state<Dashboard[]>([]);
	activeDashboard = $state<Dashboard | null>(null);
	pinnedKeys = $state<string[]>(this._loadPinnedKeys());
	isLoading = $state(false);
	isSaving = $state(false);
	editMode = $state(false);
	error = $state<string | null>(null);

	private saveTimer: ReturnType<typeof setTimeout> | null = null;
	private editSnapshot: Dashboard | null = null;

	private _loadPinnedKeys(): string[] {
		try {
			return JSON.parse(localStorage.getItem('mir_pinned_dashboards') ?? '[]');
		} catch {
			return [];
		}
	}

	private _savePinnedKeys() {
		localStorage.setItem('mir_pinned_dashboards', JSON.stringify(this.pinnedKeys));
	}

	get pinnedDashboards(): Dashboard[] {
		return this.pinnedKeys
			.map((k) => this.dashboards.find((d) => dashboardKey(d) === k))
			.filter(Boolean) as Dashboard[];
	}

	get dashboardsByNamespace(): Map<string, Dashboard[]> {
		const map = new SvelteMap<string, Dashboard[]>();
		const sorted = [...this.dashboards].sort(
			(a, b) =>
				a.meta.namespace.localeCompare(b.meta.namespace) || a.meta.name.localeCompare(b.meta.name)
		);
		for (const d of sorted) {
			const ns = d.meta.namespace;
			if (!map.has(ns)) map.set(ns, []);
			map.get(ns)!.push(d);
		}
		return map;
	}

	isPinned(d: Dashboard): boolean {
		return this.pinnedKeys.includes(dashboardKey(d));
	}

	isNamespaceFullyPinned(namespace: string): boolean {
		const group = this.dashboardsByNamespace.get(namespace) ?? [];
		return group.length > 0 && group.every((d) => this.isPinned(d));
	}

	isNamespacePartiallyPinned(namespace: string): boolean {
		const group = this.dashboardsByNamespace.get(namespace) ?? [];
		const count = group.filter((d) => this.isPinned(d)).length;
		return count > 0 && count < group.length;
	}

	togglePinned(d: Dashboard) {
		const key = dashboardKey(d);
		if (this.pinnedKeys.includes(key)) {
			this.pinnedKeys = this.pinnedKeys.filter((k) => k !== key);
			if (this.activeDashboard && dashboardKey(this.activeDashboard) === key) {
				this.activeDashboard = this.pinnedDashboards[0] ?? null;
			}
		} else {
			this.pinnedKeys = [...this.pinnedKeys, key];
		}
		this._savePinnedKeys();
	}

	toggleNamespace(namespace: string) {
		const group = this.dashboardsByNamespace.get(namespace) ?? [];
		const allPinned = group.every((d) => this.isPinned(d));
		const keys = group.map((d) => dashboardKey(d));
		if (allPinned) {
			this.pinnedKeys = this.pinnedKeys.filter((k) => !keys.includes(k));
			if (this.activeDashboard && keys.includes(dashboardKey(this.activeDashboard))) {
				this.activeDashboard = this.pinnedDashboards[0] ?? null;
			}
		} else {
			const toAdd = keys.filter((k) => !this.pinnedKeys.includes(k));
			this.pinnedKeys = [...this.pinnedKeys, ...toAdd];
		}
		this._savePinnedKeys();
	}

	async load() {
		this.isLoading = true;
		this.error = null;
		try {
			this.dashboards = (await dashboardApi.list()) ?? [];
			const savedKey = localStorage.getItem('mir_active_dashboard_key');
			const found = savedKey ? this.dashboards.find((d) => dashboardKey(d) === savedKey) : null;
			this.activeDashboard = found ?? this.dashboards[0] ?? null;
			if (this.pinnedKeys.length === 0 && this.activeDashboard) {
				this.pinnedKeys = [dashboardKey(this.activeDashboard)];
				this._savePinnedKeys();
			}
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
			this.pinnedKeys = [...this.pinnedKeys, dashboardKey(d)];
			this._savePinnedKeys();
			return d;
		} finally {
			this.isSaving = false;
		}
	}

	async update(d: Dashboard, patch: { name?: string; namespace?: string; description?: string }) {
		this.isSaving = true;
		try {
			const oldKey = dashboardKey(d);
			const updated = await dashboardApi.update(d.meta.namespace, d.meta.name, patch);
			const keyChanged = dashboardKey(updated) !== oldKey;
			if (keyChanged) {
				this.pinnedKeys = this.pinnedKeys.map((k) => (k === oldKey ? dashboardKey(updated) : k));
				this._savePinnedKeys();
			}
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
			this.pinnedKeys = this.pinnedKeys.filter((k) => k !== key);
			this._savePinnedKeys();
			if (this.activeDashboard && dashboardKey(this.activeDashboard) === key) {
				this.activeDashboard = this.pinnedDashboards[0] ?? this.dashboards[0] ?? null;
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
			type,
			title,
			config,
			x: 0,
			y: 0,
			w: 4,
			h: 4
		};
		return this._persistWidgets(d, [...(d.spec.widgets ?? []), newWidget]);
	}

	async removeWidget(d: Dashboard, widgetId: string) {
		return this._persistWidgets(
			d,
			d.spec.widgets.filter((w) => w.id !== widgetId)
		);
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

	enterEditMode(): { name: string; namespace: string } {
		this.editSnapshot = this.activeDashboard
			? structuredClone($state.snapshot(this.activeDashboard))
			: null;
		this.editMode = true;
		return {
			name: this.activeDashboard?.meta.name ?? '',
			namespace: this.activeDashboard?.meta.namespace ?? ''
		};
	}

	saveEditMode() {
		this.editSnapshot = null;
		this.editMode = false;
	}

	cancelEditMode() {
		if (this.editSnapshot) {
			this._syncDashboard(this.activeDashboard!, this.editSnapshot);
			this.editSnapshot = null;
		}
		if (this.saveTimer) {
			clearTimeout(this.saveTimer);
			this.saveTimer = null;
		}
		this.editMode = false;
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
