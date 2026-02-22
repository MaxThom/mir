<script lang="ts">
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu/index.js';
	import * as Sidebar from '$lib/shared/components/shadcn/sidebar/index.js';
	import { useSidebar } from '$lib/shared/components/shadcn/sidebar/index.js';
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import ServerIcon from '@lucide/svelte/icons/server';
	import CheckIcon from '@lucide/svelte/icons/check';
	import { contextStore } from '../../contexts/stores/contexts.svelte';
	import { formatNatsUrl } from '../../../shared/utils/url-formatters';

	const sidebar = useSidebar();
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		{#if contextStore.isLoading}
			<!-- Loading skeleton -->
			<Sidebar.MenuButton size="lg">
				<div
					class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary/20 animate-pulse"
				></div>
				<div class="grid flex-1 gap-1 text-start text-sm">
					<div class="h-4 w-24 bg-sidebar-primary/20 rounded animate-pulse"></div>
					<div class="h-3 w-16 bg-sidebar-primary/20 rounded animate-pulse"></div>
				</div>
			</Sidebar.MenuButton>
		{:else if contextStore.error}
			<!-- Error state -->
			<Sidebar.MenuButton size="lg">
				<div
					class="flex aspect-square size-8 items-center justify-center rounded-lg bg-destructive/10 text-destructive"
				>
					<ServerIcon class="size-4" />
				</div>
				<div class="grid flex-1 text-start text-sm leading-tight">
					<span class="truncate font-medium text-destructive">Error</span>
					<span class="truncate text-xs text-muted-foreground">Failed to load</span>
				</div>
			</Sidebar.MenuButton>
		{:else if contextStore.activeContext}
			<!-- Active context display -->
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
								<ServerIcon class="size-4" />
							</div>
							<div class="grid flex-1 text-start text-sm leading-tight">
								<span class="truncate font-medium">
									{contextStore.activeContext?.name ?? ''}
								</span>
								<span class="truncate text-xs"
									>{formatNatsUrl(contextStore.activeContext?.target ?? '')}</span
								>
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
					{#each contextStore.contexts as context, index (context.name)}
						<DropdownMenu.Item
							onSelect={() => contextStore.setActiveContext(context.name)}
							class="gap-2 p-2"
						>
							<div class="flex size-6 items-center justify-center rounded-md border">
								<ServerIcon class="size-3.5 shrink-0" />
							</div>
							<div class="flex flex-1 flex-col gap-0">
								<span class="text-sm font-medium">{context.name}</span>
								<span class="text-xs text-muted-foreground">{formatNatsUrl(context.target)}</span>
							</div>
							{#if contextStore.activeContext?.name === context.name}
								<CheckIcon class="size-4 text-primary" />
							{/if}
							<DropdownMenu.Shortcut>⌘{index + 1}</DropdownMenu.Shortcut>
						</DropdownMenu.Item>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		{/if}
	</Sidebar.MenuItem>
</Sidebar.Menu>
