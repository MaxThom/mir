<script lang="ts">
	import * as Collapsible from '$lib/shared/components/shadcn/collapsible/index.js';
	import * as Sidebar from '$lib/shared/components/shadcn/sidebar/index.js';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { resolve } from '$app/paths';
	import type { NavMainProps, NavItem } from '../types/types';

	let {
		label,
		items,
		collapsible = true,
		showGroupLabel = true,
		hideOnCollapse = false
	}: NavMainProps = $props();

	function shouldBeCollapsible(item: NavItem): boolean {
		return collapsible && (item.items?.length ?? 0) > 0;
	}
</script>

<Sidebar.Group class={hideOnCollapse ? 'group-data-[collapsible=icon]:hidden' : ''}>
	{#if showGroupLabel}
		<Sidebar.GroupLabel>{label}</Sidebar.GroupLabel>
	{/if}
	<Sidebar.Menu>
		{#each items as item (item.title)}
			{#if shouldBeCollapsible(item)}
				<Collapsible.Root open={item.isActive} class="group/collapsible">
					{#snippet child({ props })}
						<Sidebar.MenuItem {...props}>
							<Collapsible.Trigger>
								{#snippet child({ props })}
									<Sidebar.MenuButton {...props} tooltipContent={item.title}>
										{#if item.icon}
											<item.icon />
										{/if}
										<span>{item.title}</span>
										<ChevronRightIcon
											class="ms-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
										/>
									</Sidebar.MenuButton>
								{/snippet}
							</Collapsible.Trigger>
							<Collapsible.Content>
								<Sidebar.MenuSub>
									{#each item.items ?? [] as subItem (subItem.title)}
										<Sidebar.MenuSubItem>
											<Sidebar.MenuSubButton>
												{#snippet child({ props })}
													<a href={resolve(subItem.url)} data-sveltekit-preload-data="" {...props}>
														<span>{subItem.title}</span>
													</a>
												{/snippet}
											</Sidebar.MenuSubButton>
										</Sidebar.MenuSubItem>
									{/each}
								</Sidebar.MenuSub>
							</Collapsible.Content>
						</Sidebar.MenuItem>
					{/snippet}
				</Collapsible.Root>
			{:else}
				<Sidebar.MenuItem>
					<Sidebar.MenuButton tooltipContent={item.title}>
						{#snippet child({ props })}
							<a href={resolve(item.url)} data-sveltekit-preload-data="" {...props}>
								{#if item.icon}
									<item.icon />
								{/if}
								<span>{item.title}</span>
							</a>
						{/snippet}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			{/if}
		{/each}
	</Sidebar.Menu>
</Sidebar.Group>
