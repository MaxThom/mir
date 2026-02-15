import type { User } from './user';

export type Context = {
	name: string;
	target: string; // NATS URL
	grafana: string; // Grafana URL
};

export type NavSubItem = {
	title: string;
	url: string;
};

export type NavItem = {
	title: string;
	url: string;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	icon?: any; // Component when @lucide/svelte updates types
	isActive?: boolean;
	items?: NavSubItem[];
};

export type NavMainProps = {
	label: string;
	items: NavItem[];
	collapsible?: boolean;
	showGroupLabel?: boolean;
	hideOnCollapse?: boolean;
};

export type SidebarData = {
	user: User;
	navMain: NavItem[];
};
