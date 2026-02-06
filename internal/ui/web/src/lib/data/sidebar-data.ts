import AudioWaveformIcon from '@lucide/svelte/icons/audio-waveform';
import BookOpenIcon from '@lucide/svelte/icons/book-open';
import BotIcon from '@lucide/svelte/icons/bot';
import CommandIcon from '@lucide/svelte/icons/command';
import FileBraces from '@lucide/svelte/icons/file-braces';
import CalendarSearch from '@lucide/svelte/icons/calendar-search';
import GalleryVerticalEndIcon from '@lucide/svelte/icons/gallery-vertical-end';
import SquareTerminalIcon from '@lucide/svelte/icons/square-terminal';
import type { SidebarData } from '$lib/components/app-sidebar/types';

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
			url: '#',
			icon: SquareTerminalIcon,
			isActive: true,
			items: []
		},
		{
			title: 'Devices',
			url: '#',
			icon: BotIcon,
			isActive: true,
			items: [
				{
					title: 'Telemetry',
					url: '#'
				},
				{
					title: 'Commands',
					url: '#'
				},
				{
					title: 'Configuration',
					url: '#'
				}
			]
		},
		{
			title: 'Schemas',
			url: '#',
			icon: FileBraces,
			items: [
				{
					title: 'Explorer',
					url: '#'
				}
			]
		},
		{
			title: 'Events',
			url: '#',
			icon: CalendarSearch,
			items: [
				{
					title: 'List',
					url: '#'
				}
			]
		},
		{
			title: 'Documentation',
			url: '#',
			icon: BookOpenIcon,
			items: [
				{
					title: 'Introduction',
					url: '#'
				},
				{
					title: 'Get Started',
					url: '#'
				},
				{
					title: 'Tutorials',
					url: '#'
				},
				{
					title: 'Changelog',
					url: '#'
				}
			]
		}
	]
};
