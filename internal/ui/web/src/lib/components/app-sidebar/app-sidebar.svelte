<script lang="ts" module>
	import AudioWaveformIcon from '@lucide/svelte/icons/audio-waveform';
	import BookOpenIcon from '@lucide/svelte/icons/book-open';
	import BotIcon from '@lucide/svelte/icons/bot';
	import CommandIcon from '@lucide/svelte/icons/command';
	import FileBraces from '@lucide/svelte/icons/file-braces';
	import CalendarSearch from '@lucide/svelte/icons/calendar-search';
	import GalleryVerticalEndIcon from '@lucide/svelte/icons/gallery-vertical-end';
	import SquareTerminalIcon from '@lucide/svelte/icons/square-terminal';
	// import MirAlpha from '@lib/assets/mir_alpha_nocolor.svg';
	// import MirAlphaIcon from '$lib/components/icons/mir_logo.svelte';
	import MirAlphaLogo from '$lib/assets/mir_alpha.png';
	// This is sample data.
	const data = {
		user: {
			name: 'shadcn',
			email: 'm@example.com',
			avatar: '/avatars/shadcn.jpg'
		},
		teams: [
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
		// projects: [
		// 	{
		// 		name: 'Design Engineering',
		// 		url: '#',
		// 		icon: FrameIcon
		// 	},
		// 	{
		// 		name: 'Sales & Marketing',
		// 		url: '#',
		// 		icon: ChartPieIcon
		// 	},
		// 	{
		// 		name: 'Travel',
		// 		url: '#',
		// 		icon: MapIcon
		// 	}
		// ]
	};
</script>

<script lang="ts">
	import NavMain from './nav-main.svelte';
	import NavProjects from './nav-projects.svelte';
	import NavUser from './nav-user.svelte';
	import ContextSwitcher from './context-switcher.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	let {
		ref = $bindable(null),
		collapsible = 'icon',
		...restProps
	}: ComponentProps<typeof Sidebar.Root> = $props();
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<ContextSwitcher teams={data.teams} />
	</Sidebar.Header>
	<Sidebar.Content>
		<NavMain items={data.navMain} />
		<!-- <NavProjects projects={data.projects} /> -->
	</Sidebar.Content>
	<Sidebar.Footer>
		<div class="flex items-center justify-center">
			<img src={MirAlphaLogo} alt="Mir Logo" class="w-48 items-center justify-center" />
		</div>
		<NavUser user={data.user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
