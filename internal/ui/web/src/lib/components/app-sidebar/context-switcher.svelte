<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import { useSidebar } from '$lib/components/ui/sidebar/index.js';
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import type { Context } from '$lib/types';

	let { contexts }: { contexts: Context[] } = $props();
	const sidebar = useSidebar();

	// Initialize with first context (intentionally capturing initial value)
	// eslint-disable-next-line svelte/state-referenced-locally
	let activeContext = $state<Context>(contexts[0] || {
		name: 'Default',
		logo: null,
		plan: 'Free'
	});
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Sidebar.MenuButton
						{...props}
						size="lg"
						class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
					>
						<div
							class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground"
						>
							<activeContext.logo class="size-4" />
						</div>
						<div class="grid flex-1 text-start text-sm leading-tight">
							<span class="truncate font-medium">
								{activeContext.name}
							</span>
							<span class="truncate text-xs">{activeContext.plan}</span>
						</div>
						<ChevronsUpDownIcon class="ms-auto" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content
				class="w-(--bits-dropdown-menu-anchor-width) min-w-56 rounded-lg"
				align="start"
				side={sidebar.isMobile ? 'bottom' : 'right'}
				sideOffset={4}
			>
				<DropdownMenu.Label class="text-xs text-muted-foreground">Contexts</DropdownMenu.Label>
				{#each contexts as context, index (context.name)}
					<DropdownMenu.Item onSelect={() => (activeContext = context)} class="gap-2 p-2">
						<div class="flex size-6 items-center justify-center rounded-md border">
							<context.logo class="size-3.5 shrink-0" />
						</div>
						{context.name}
						<DropdownMenu.Shortcut>⌘{index + 1}</DropdownMenu.Shortcut>
					</DropdownMenu.Item>
				{/each}
				<DropdownMenu.Separator />
				<DropdownMenu.Item class="gap-2 p-2">
					<div class="flex size-6 items-center justify-center rounded-md border bg-transparent">
						<PlusIcon class="size-4" />
					</div>
					<div class="font-medium text-muted-foreground">Add context</div>
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
