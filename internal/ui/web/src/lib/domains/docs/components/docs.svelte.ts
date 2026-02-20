import type { Component } from 'svelte';
import type { DocTab, RouteKey } from '../types/docs';

import DocDevices from '../components/doc-devices.svelte';

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
		const key = this.resolveRouteKey(pathname);
		return docsContent[key];
	}

	private resolveRouteKey(pathname: string): RouteKey {
		if (pathname === '/') return 'dashboard';
		if (pathname === '/devices') return 'devices';

		return 'dashboard';
	}
}

export const docsStore = new DocsStore();
