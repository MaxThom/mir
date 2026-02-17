import BookOpenIcon from '@lucide/svelte/icons/book-open';
import BotIcon from '@lucide/svelte/icons/bot';
import FileBraces from '@lucide/svelte/icons/file-braces';
import CalendarSearch from '@lucide/svelte/icons/calendar-search';
import SquareTerminalIcon from '@lucide/svelte/icons/square-terminal';
import type { SidebarData } from '$lib/types';

export const sidebarData: SidebarData = {
	user: {
		name: 'shadcn',
		email: 'm@example.com',
		avatar: '/avatars/shadcn.jpg'
	},
	navMain: [
		{
			title: 'Dashboard',
			url: '/',
			icon: SquareTerminalIcon,
			isActive: true,
			items: []
		},
		{
			title: 'Devices',
			url: '/devices',
			icon: BotIcon,
			isActive: true,
			items: [
				{
					title: 'All Devices',
					url: '/devices'
				},
				{
					title: 'Telemetry',
					url: '/devices/telemetry'
				},
				{
					title: 'Commands',
					url: '/devices/commands'
				}
			]
		},
		{
			title: 'Schemas',
			url: '/schemas',
			icon: FileBraces,
			items: [
				{
					title: 'Explorer',
					url: '/schemas'
				}
			]
		},
		{
			title: 'Events',
			url: '/events',
			icon: CalendarSearch,
			items: [
				{
					title: 'List',
					url: '/events'
				}
			]
		},
		{
			title: 'Documentation',
			url: '/docs',
			icon: BookOpenIcon,
			items: [
				{
					title: 'Introduction',
					url: '/docs/introduction'
				},
				{
					title: 'Get Started',
					url: '/docs/get-started'
				},
				{
					title: 'Tutorials',
					url: '/docs/tutorials'
				}
			]
		}
	]
};
