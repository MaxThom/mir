import BotIcon from '@lucide/svelte/icons/bot';
import CalendarSearch from '@lucide/svelte/icons/calendar-search';
import HomeIcon from '@lucide/svelte/icons/home';
import SquareTerminalIcon from '@lucide/svelte/icons/square-terminal';
import type { SidebarData } from '../types/types';

export const sidebarData: SidebarData = {
	user: {
		name: 'maxthom',
		email: 'maxthomassin@hotmail.com',
		avatar: ''
	},
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
			icon: SquareTerminalIcon,
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
