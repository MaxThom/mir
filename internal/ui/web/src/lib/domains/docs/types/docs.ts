import type { Component } from 'svelte';

export type DocTab = 'web' | 'cli' | 'gosdk' | 'tssdk';
export type PageDocContent = {
	title: string;
	overview: string;
	tabs: Component<DocContentProps>;
};
interface DocContentProps {
	tab: DocTab;
}
export type RouteKey = 'dashboard' | 'devices' | 'devices/create' | 'devices/detail' | 'devices/telemetry' | 'devices/commands' | 'devices/configuration' | 'devices/events';
// | 'devices/detail'
// | 'devices/commands'
// | 'devices/configuration'
// | 'devices/events'
// | 'devices/schema'
// | 'devices/telemetry';
