import type { Component } from 'svelte';
import type { DocTab, RouteKey } from '../types/docs';

import DocDevices from '../components/doc-devices.svelte';
import DocDashboard from '../components/doc-dashboard.svelte';
import DocDashboards from '../components/doc-dashboards.svelte';
import DocCreateDevice from '../components/doc-create-device.svelte';
import DocDeviceOverview from '../components/doc-device-overview.svelte';
import DocDeviceTelemetry from '../components/doc-device-telemetry.svelte';
import DocDeviceCommands from '../components/doc-device-commands.svelte';
import DocDeviceConfiguration from '../components/doc-device-configuration.svelte';
import DocDeviceEvents from '../components/doc-device-events.svelte';

export const docsContent: Record<RouteKey, Component> = {
	dashboard: DocDashboard,
	dashboards: DocDashboards,
	devices: DocDevices,
	'devices/create': DocCreateDevice,
	'devices/detail': DocDeviceOverview,
	'devices/telemetry': DocDeviceTelemetry,
	'devices/commands': DocDeviceCommands,
	'devices/configuration': DocDeviceConfiguration,
	'devices/events': DocDeviceEvents
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
		if (pathname === '/dashboards') return DocDashboards;
		if (pathname === '/devices') return DocDevices;
		if (pathname === '/devices/create') return DocCreateDevice;
		if (pathname.includes('/telemetry')) return DocDeviceTelemetry;
		if (pathname.includes('/commands')) return DocDeviceCommands;
		if (pathname.includes('/configuration')) return DocDeviceConfiguration;
		if (pathname.includes('/events')) return DocDeviceEvents;
		if (pathname.startsWith('/devices/') && !pathname.endsWith('/bulk')) return DocDeviceOverview;
		return DocDashboard;
	}
}

export const docsStore = new DocsStore();
