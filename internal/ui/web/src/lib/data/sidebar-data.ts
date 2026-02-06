import AudioWaveformIcon from '@lucide/svelte/icons/audio-waveform';
import BookOpenIcon from '@lucide/svelte/icons/book-open';
import BotIcon from '@lucide/svelte/icons/bot';
import CommandIcon from '@lucide/svelte/icons/command';
import FileBraces from '@lucide/svelte/icons/file-braces';
import CalendarSearch from '@lucide/svelte/icons/calendar-search';
import GalleryVerticalEndIcon from '@lucide/svelte/icons/gallery-vertical-end';
import SquareTerminalIcon from '@lucide/svelte/icons/square-terminal';
import type { SidebarData } from '$lib/types';

export const sidebarData: SidebarData = {
	user: {
		name: 'shadcn',
		email: 'm@example.com',
		avatar: '/avatars/shadcn.jpg'
	},
	contexts: [
		{
			name: 'Prod',
			logo: GalleryVerticalEndIcon,
			plan: 'Mir'
		},
		{
			name: 'Acme Corp.',
			logo: AudioWaveformIcon,
			plan: 'Startup'
		},
		{
			name: 'Evil Corp.',
			logo: CommandIcon,
			plan: 'Free'
		}
	],
	navMain: [
		{
			title: 'Dashboard',
			url: '/dashboard',
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
