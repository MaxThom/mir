<script lang="ts">
	import MirAlphaLogo from '$lib/assets/mir_alpha.png';
	import NavSection from './nav-section.svelte';
	import NavUser from './nav-user.svelte';
	import ContextSwitcher from './context-switcher.svelte';
	import * as Sidebar from '$lib/shared/components/shadcn/sidebar/index.js';
	import type { ComponentProps } from 'svelte';
	import type { User } from '../../user/types/user';
	import type { NavItem } from '../types/types';

	type SidebarProps = ComponentProps<typeof Sidebar.Root> & {
		user: User;
		navMain: NavItem[];
	};

	let {
		user,
		navMain,
		ref = $bindable(null),
		collapsible = 'icon',
		...restProps
	}: SidebarProps = $props();
</script>

<Sidebar.Root {collapsible} {...restProps}>
	<Sidebar.Header>
		<div class="flex flex-col gap-0">
			<Sidebar.GroupLabel class="group-data-[collapsible=icon]:hidden">Contexts</Sidebar.GroupLabel>
			<ContextSwitcher />
		</div>
	</Sidebar.Header>
	<Sidebar.Content>
		<NavSection label="Platform" items={navMain} />
	</Sidebar.Content>
	<Sidebar.Footer>
		<div class="flex flex-col items-center justify-center group-data-[collapsible=icon]:hidden">
			<img src={MirAlphaLogo} alt="Mir Logo" class="w-48" />
			<span class="text-sm font-semibold tracking-widest text-sidebar-foreground/40 uppercase">Mir Cockpit</span>
		</div>
		<div class="hidden items-center justify-center group-data-[collapsible=icon]:flex">
			<img src={MirAlphaLogo} alt="Mir Logo" class="w-8" />
		</div>
		<NavUser {user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
