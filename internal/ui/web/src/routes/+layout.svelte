<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { ModeWatcher } from 'mode-watcher';
	import SunIcon from '@lucide/svelte/icons/sun';
	import MoonIcon from '@lucide/svelte/icons/moon';

	import { toggleMode } from 'mode-watcher';
	import { Button } from '$lib/components/ui/button/index.js';
	import { onMount } from 'svelte';

	let { children } = $props();

	import AppSidebar from '$lib/components/app-sidebar/app-sidebar.svelte';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import { sidebarData } from '$lib/data/sidebar-data';
	import { contextStore } from '$lib/stores/contexts.svelte';
	import { page } from '$app/stores';
	import { generateBreadcrumbs } from '$lib/utils/breadcrumbs';

	// Generate breadcrumbs from current pathname
	let breadcrumbs = $derived(generateBreadcrumbs($page.url.pathname));

	onMount(async () => {
		await contextStore.initialize();
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#if contextStore.error}
	<div class="fixed top-0 left-0 right-0 bg-destructive/90 text-destructive-foreground p-4 z-50">
		<div class="container flex items-center justify-between">
			<p class="text-sm font-medium">Failed to load contexts: {contextStore.error}</p>
			<Button onclick={() => contextStore.initialize()} variant="secondary" size="sm">
				Retry
			</Button>
		</div>
	</div>
{/if}

<Sidebar.Provider>
	<AppSidebar user={sidebarData.user} navMain={sidebarData.navMain} />
	<Sidebar.Inset>
		<header
			class="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12"
		>
			<div class="flex items-center gap-2 px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Button onclick={toggleMode} variant="link" size="icon">
					<SunIcon
						class="h-[1.2rem] w-[1.2rem] scale-100 rotate-0 transition-all! dark:scale-0 dark:-rotate-90"
					/>
					<MoonIcon
						class="absolute h-[1.2rem] w-[1.2rem] scale-0 rotate-90 transition-all! dark:scale-100 dark:rotate-0"
					/>
					<span class="sr-only">Toggle theme</span>
				</Button>
				<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						{#each breadcrumbs as crumb, index}
							<Breadcrumb.Item class="hidden md:block">
								{#if crumb.isCurrentPage}
									<Breadcrumb.Page>{crumb.label}</Breadcrumb.Page>
								{:else}
									<Breadcrumb.Link href={crumb.href}>{crumb.label}</Breadcrumb.Link>
								{/if}
							</Breadcrumb.Item>
							{#if index < breadcrumbs.length - 1}
								<Breadcrumb.Separator class="hidden md:block" />
							{/if}
						{/each}
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>
		</header>
		<div class="flex flex-1 flex-col gap-4 p-4 pt-0">
			<ModeWatcher />
			{@render children()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
