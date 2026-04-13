import { tick } from 'svelte';
import { SvelteMap } from 'svelte/reactivity';
import {
	dashboardApi,
	dashboardKey,
	type Dashboard,
	type Widget,
	type WidgetConfig,
	type WidgetType
} from '../api/dashboard-api';
import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
import { activityStore } from '$lib/domains/activity/stores/activity.svelte';

class DashboardStore {
	dashboards = $state<Dashboard[]>([]);
	activeDashboard = $state<Dashboard | null>(null);
	pinnedKeys = $state<string[]>(this._loadPinnedKeys());
	isLoading = $state(false);
	isSaving = $state(false);
	isRefreshing = $state(false);
	editMode = $state(false);
	isCreatingNew = $state(false);
	error = $state<string | null>(null);

	private _refreshCount = 0;
	private editSnapshot: Dashboard | null = null;
	private _preCreateDashboard: Dashboard | null = null;

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

	pinnedDashboards = $derived(
		this.pinnedKeys
			.map((k) => this.dashboards.find((d) => dashboardKey(d) === k))
			.filter(Boolean) as Dashboard[]
	);

	dashboardsByNamespace = $derived.by(() => {
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
	});

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
			if (this.activeDashboard) {
				editorPrefs.setRefreshInterval(this.activeDashboard.spec.refreshInterval ?? 10);
				editorPrefs.setTimeMinutes(this.activeDashboard.spec.timeMinutes ?? 60);
			}
			if (this.pinnedDashboards.length === 0 && this.activeDashboard) {
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
		editorPrefs.setRefreshInterval(d.spec.refreshInterval ?? 10);
		editorPrefs.setTimeMinutes(d.spec.timeMinutes ?? 60);
	}

	beginCreate() {
		this._preCreateDashboard = this.activeDashboard;
		this.activeDashboard = {
			meta: { name: '__draft__', namespace: 'default' },
			spec: { description: '', widgets: [], refreshInterval: 10, timeMinutes: 60 }
		} as unknown as Dashboard;
		this.isCreatingNew = true;
	}

	cancelCreate() {
		this.activeDashboard = this._preCreateDashboard;
		this._preCreateDashboard = null;
		this.isCreatingNew = false;
	}

	async create(name: string, namespace = 'default', description = '') {
		this.isSaving = true;
		// tick() lets widget $effects that watch isSaving flush their pending editor content
		// (e.g. command widget saves typed payload before we snapshot draftWidgets)
		await tick();
		try {
			// Use $state.snapshot to get a plain (non-proxy) copy for reliable JSON serialization
			const draftWidgets = this.isCreatingNew
				? ($state.snapshot(this.activeDashboard)?.spec.widgets ?? [])
				: [];
			const d = await dashboardApi.create(name, namespace, description, draftWidgets);
			this.dashboards = [...this.dashboards, d];
			this.pinnedKeys = [...this.pinnedKeys, dashboardKey(d)];
			this._savePinnedKeys();
			this.activeDashboard = d;
			localStorage.setItem('mir_active_dashboard_key', dashboardKey(d));
			editorPrefs.setRefreshInterval(d.spec.refreshInterval ?? 10);
			editorPrefs.setTimeMinutes(d.spec.timeMinutes ?? 60);
			this.isCreatingNew = false;
			this._preCreateDashboard = null;
			activityStore.add({ kind: 'success', category: 'Dashboard', title: 'Created', request: { name, namespace } });
			return d;
		} catch (err) {
			activityStore.add({ kind: 'error', category: 'Dashboard', title: 'Create Failed', error: err instanceof Error ? err.message : String(err) });
			throw err;
		} finally {
			this.isSaving = false;
		}
	}

	async update(d: Dashboard, patch: { name?: string; namespace?: string; description?: string; widgets?: Widget[]; refreshInterval?: number; timeMinutes?: number }) {
		this.isSaving = true;
		try {
			const oldKey = dashboardKey(d);
			const serverResponse = await dashboardApi.update(d.meta.namespace, d.meta.name, patch);
			const keyChanged = dashboardKey(serverResponse) !== oldKey;
			if (keyChanged) {
				this.pinnedKeys = this.pinnedKeys.map((k) => (k === oldKey ? dashboardKey(serverResponse) : k));
				this._savePinnedKeys();
			}
			// If widgets were included in the patch, the server response is authoritative.
			// Otherwise (rename/description-only), preserve the in-memory spec to avoid wiping unsaved widget changes.
			const merged = patch.widgets !== undefined
				? serverResponse
				: (() => {
						const inMemorySpec = this.activeDashboard?.spec ?? d.spec;
						return { ...serverResponse, spec: { ...serverResponse.spec, widgets: inMemorySpec.widgets, refreshInterval: inMemorySpec.refreshInterval, timeMinutes: inMemorySpec.timeMinutes } };
					})();
			this._syncDashboard(d, merged);
			activityStore.add({ kind: 'success', category: 'Dashboard', title: 'Updated', request: { name: serverResponse.meta.name, namespace: serverResponse.meta.namespace } });
			return merged;
		} catch (err) {
			activityStore.add({ kind: 'error', category: 'Dashboard', title: 'Update Failed', error: err instanceof Error ? err.message : String(err) });
			throw err;
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
			activityStore.add({ kind: 'success', category: 'Dashboard', title: 'Deleted', request: { name: d.meta.name, namespace: d.meta.namespace } });
		} catch (err) {
			activityStore.add({ kind: 'error', category: 'Dashboard', title: 'Delete Failed', error: err instanceof Error ? err.message : String(err) });
			throw err;
		} finally {
			this.isSaving = false;
		}
	}

	addWidget(d: Dashboard, type: WidgetType, title: string, config: WidgetConfig) {
		const existing = d.spec.widgets ?? [];
		const bottomY = existing.reduce((max, w) => Math.max(max, w.y + w.h), 0);
		const newWidget: Widget = {
			id: crypto.randomUUID(),
			type,
			title,
			config,
			x: 0,
			y: bottomY,
			w: 4,
			h: 4
		};
		const updated = [...(d.spec.widgets ?? []), newWidget];
		if (this.isCreatingNew) {
			this.activeDashboard = { ...d, spec: { ...d.spec, widgets: updated } };
			return;
		}
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });
	}

	removeWidget(d: Dashboard, widgetId: string) {
		const updated = d.spec.widgets.filter((w) => w.id !== widgetId);
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });
	}

	updateWidgetConfigInMemory(d: Dashboard, widgetId: string, config: WidgetConfig) {
		const updated = (d.spec.widgets ?? []).map((w) =>
			w.id === widgetId ? { ...w, config } : w
		);
		if (this.isCreatingNew) {
			this.activeDashboard = { ...d, spec: { ...d.spec, widgets: updated } };
			return;
		}
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });
	}

	persistActiveDashboard() {
		if (!this.activeDashboard || this.isCreatingNew) return;
		return this._persistWidgets(
			this.activeDashboard,
			this.activeDashboard.spec.widgets,
			this.activeDashboard.spec.refreshInterval,
			this.activeDashboard.spec.timeMinutes
		);
	}

	saveWidgetConfig(d: Dashboard, widgetId: string, config: WidgetConfig) {
		const updated = (d.spec.widgets ?? []).map((w) =>
			w.id === widgetId ? { ...w, config } : w
		);
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });
	}

	saveWidgetViewState(widgetId: string, config: WidgetConfig) {
		if (!this.activeDashboard) return;
		const updated = (this.activeDashboard.spec.widgets ?? []).map((w) =>
			w.id === widgetId ? { ...w, config } : w
		);
		if (this.isCreatingNew) {
			this.activeDashboard = { ...this.activeDashboard, spec: { ...this.activeDashboard.spec, widgets: updated } };
			return;
		}
		this._syncDashboard(this.activeDashboard, { ...this.activeDashboard, spec: { ...this.activeDashboard.spec, widgets: updated } });
	}

	saveLayout(d: Dashboard, layoutItems: Pick<Widget, 'id' | 'x' | 'y' | 'w' | 'h'>[]) {
		const posMap = new Map(layoutItems.map((item) => [item.id, item]));
		const updated = (d.spec.widgets ?? []).map((w) => {
			const pos = posMap.get(w.id);
			return pos ? { ...w, ...pos } : w;
		});
		if (this.isCreatingNew) {
			this.activeDashboard = { ...d, spec: { ...d.spec, widgets: updated } };
			return;
		}
		this._syncDashboard(d, { ...d, spec: { ...d.spec, widgets: updated } });
	}

	async enterEditMode(): Promise<{ name: string; namespace: string }> {
		if (this.activeDashboard && !this.isCreatingNew) {
			try {
				const fresh = await dashboardApi.get(
					this.activeDashboard.meta.namespace,
					this.activeDashboard.meta.name
				);
				this._syncDashboard(this.activeDashboard, fresh);
			} catch {
				// If fetch fails, proceed with current in-memory state
			}
		}
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
		this.editMode = false;
	}

	saveRefreshInterval(d: Dashboard, interval: number) {
		this._syncDashboard(d, { ...d, spec: { ...d.spec, refreshInterval: interval } });
	}

	saveTimeMinutes(d: Dashboard, minutes: number) {
		this._syncDashboard(d, { ...d, spec: { ...d.spec, timeMinutes: minutes } });
	}

	private async _persistWidgets(d: Dashboard, widgets: Widget[], refreshInterval?: number, timeMinutes?: number) {
		this.isSaving = true;
		try {
			const updated = await dashboardApi.saveWidgets(d.meta.namespace, d.meta.name, widgets, refreshInterval, timeMinutes);
			this._syncDashboard(d, updated);
			return updated;
		} finally {
			this.isSaving = false;
		}
	}

	refreshStart() {
		this._refreshCount++;
		this.isRefreshing = true;
	}

	refreshDone() {
		this._refreshCount = Math.max(0, this._refreshCount - 1);
		if (this._refreshCount === 0) this.isRefreshing = false;
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
