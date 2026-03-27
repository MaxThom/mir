import type { Component } from 'svelte';
import type { DocTab, RouteKey } from '../types/docs';

import DocDevices from '../components/doc-devices.svelte';

export const docsContent: Record<RouteKey, Component> = {
	dashboard: DocDevices,
	devices: DocDevices,
	'devices/create': DocDevices,
	'devices/detail': DocDevices,
	'devices/telemetry': DocDevices,
	'devices/commands': DocDevices,
	'devices/configuration': DocDevices,
	'devices/events': DocDevices
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
		if (pathname === '/devices/create') return 'devices/create';
		if (pathname.includes('/telemetry')) return 'devices/telemetry';
		if (pathname.includes('/commands')) return 'devices/commands';
		if (pathname.includes('/configuration')) return 'devices/configuration';
		if (pathname.includes('/events')) return 'devices/events';
		if (pathname.startsWith('/devices/')) return 'devices/detail';
		if (pathname === '/devices') return 'devices';

		return 'dashboard';
	}
}

export const docsStore = new DocsStore();
