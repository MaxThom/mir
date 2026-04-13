import BotIcon from '@lucide/svelte/icons/bot';
import CalendarSearch from '@lucide/svelte/icons/calendar-search';
import HomeIcon from '@lucide/svelte/icons/home';
import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
import type { SidebarData } from '../types/types';

export const sidebarData: SidebarData = {
	contexts: [],
	navMain: [
		{
			title: 'Home',
			url: '/',
			icon: HomeIcon,
			isActive: true,
			items: []
		},
		{
			title: 'Dashboards',
			url: '/dashboards',
			icon: LayoutDashboardIcon,
			isActive: true,
			items: []
		},
		{
			title: 'Devices',
			url: '/devices',
			icon: BotIcon,
			isActive: true,
			items: []
		},
		{
			title: 'Events',
			url: '/events',
			icon: CalendarSearch,
			items: []
		}
	]
};
