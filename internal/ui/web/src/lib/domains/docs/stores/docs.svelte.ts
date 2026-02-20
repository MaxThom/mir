import type { Component } from 'svelte';
import type { DocTab, RouteKey } from '../types/docs';

import DocDevices from '../components/doc-devices.svelte';
import DocDashboard from '../components/doc-dashboard.svelte';

export const docsContent: Record<RouteKey, Component> = {
	dashboard: DocDevices,
	devices: DocDevices
};

class DocsStore {
	isOpen = $state(false);
	activeTab = $state<DocTab>('web');

	toggle() {
		this.isOpen = !this.isOpen;
	}

	setTab(tab: DocTab) {
		this.activeTab = tab;
	}

	getContent(pathname: string): Component {
		if (pathname === '/') return DocDashboard;
		if (pathname === '/devices') return DocDevices;
		return DocDashboard;
	}
}

export const docsStore = new DocsStore();
