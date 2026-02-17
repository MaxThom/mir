<script lang="ts">
	import MirAlphaLogo from '$lib/assets/mir_alpha.png';
	import NavSection from './nav-section.svelte';
	import NavUser from './nav-user.svelte';
	import ContextSwitcher from './context-switcher.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
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
		<ContextSwitcher />
	</Sidebar.Header>
	<Sidebar.Content>
		<NavSection label="Platform" items={navMain} />
	</Sidebar.Content>
	<Sidebar.Footer>
		<div class="flex items-center justify-center">
			<img src={MirAlphaLogo} alt="Mir Logo" class="w-48 items-center justify-center" />
		</div>
		<NavUser {user} />
	</Sidebar.Footer>
	<Sidebar.Rail />
</Sidebar.Root>
