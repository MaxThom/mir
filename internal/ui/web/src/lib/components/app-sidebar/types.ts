export type User = {
	name: string;
	email: string;
	avatar: string;
};

export type Context = {
	name: string;
	// This should be `Component` after @lucide/svelte updates types
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	logo: any;
	plan: string;
};

export type NavItem = {
	title: string;
	url: string;
	// This should be `Component` after @lucide/svelte updates types
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	icon?: any;
	isActive?: boolean;
	items?: {
		title: string;
		url: string;
	}[];
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
	contexts: Context[];
	navMain: NavItem[];
};
