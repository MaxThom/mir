import type { User } from '../../user/types/user';
import type { Context } from '../../contexts/types/types';

export type NavSubItem = {
	title: string;
	url: string;
};

export type NavItem = {
	title: string;
	url: string;
	// This should be `Component` after @lucide/svelte updates types
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	icon?: any;
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
	contexts: Context[];
	navMain: NavItem[];
};
