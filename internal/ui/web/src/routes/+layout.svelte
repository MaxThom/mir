<script lang="ts">
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import SunIcon from '@lucide/svelte/icons/sun';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import UnplugIcon from '@lucide/svelte/icons/unplug';
	import BookOpenIcon from '@lucide/svelte/icons/book-open';
	import * as Empty from '$lib/shared/components/shadcn/empty';

	import { toggleMode } from 'mode-watcher';
	import { Button } from '$lib/shared/components/shadcn/button/index.js';
	import { onMount, untrack } from 'svelte';

	let { children } = $props();

	import AppSidebar from '$lib/domains/sidebar/components/app-sidebar.svelte';
	import * as Breadcrumb from '$lib/shared/components/shadcn/breadcrumb/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import * as Sidebar from '$lib/shared/components/shadcn/sidebar/index.js';
	import { sidebarData } from '$lib/domains/sidebar/data/sidebar-data';
	import { contextStore } from '$lib/domains/contexts/stores/contexts.svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { page } from '$app/state';
	import { generateBreadcrumbs } from '$lib/domains/breadcrumbs/utils/breadcrumbs';
	import { docsStore } from '$lib/domains/docs/stores/docs.svelte';
	import DocDrawer from '$lib/domains/docs/components/doc-drawer.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	// Generate breadcrumbs from current pathname
	let breadcrumbs = $derived(generateBreadcrumbs(page.url.pathname));
	let docsContent = $derived(docsStore.getContent(page.url.pathname));

	onMount(async () => {
		await contextStore.initialize();
	});

	$effect(() => {
		const ctx = contextStore.activeContext;
		if (ctx) {
			untrack(() => mirStore.connect(ctx));
		}
	});
</script>

{#if contextStore.error}
	<div class="text-destructive-foreground fixed top-0 right-0 left-0 z-50 bg-destructive/90 p-4">
		<div class="container flex items-center justify-between">
			<p class="text-sm font-medium">Failed to load contexts: {contextStore.error}</p>
			<Button onclick={() => contextStore.initialize()} variant="secondary" size="sm">Retry</Button>
		</div>
	</div>
{/if}

<Sidebar.Provider class="h-svh">
	<AppSidebar user={sidebarData.user} navMain={sidebarData.navMain} />
	<Sidebar.Inset>
		<header
			class="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12"
		>
			<div class="flex flex-1 items-center gap-2 px-4">
				<Sidebar.Trigger class="-ms-1" />
				<Separator orientation="vertical" class="me-2 data-[orientation=vertical]:h-4" />
				<Breadcrumb.Root>
					<Breadcrumb.List>
						{#each breadcrumbs as crumb, i (i)}
							<Breadcrumb.Item class="hidden md:block">
								{#if crumb.isCurrentPage}
									<Breadcrumb.Page>{crumb.label}</Breadcrumb.Page>
								{:else}
									<Breadcrumb.Link href={crumb.href}>{crumb.label}</Breadcrumb.Link>
								{/if}
							</Breadcrumb.Item>
							{#if i < breadcrumbs.length - 1}
								<Breadcrumb.Separator class="hidden md:block" />
							{/if}
						{/each}
					</Breadcrumb.List>
				</Breadcrumb.Root>
			</div>
			<div class="flex items-center gap-2 px-4">
				<Separator orientation="vertical" class="data-[orientation=vertical]:h-4" />
				<Button onclick={() => editorPrefs.setUtc(!editorPrefs.utc)} variant="ghost" class="h-7 w-12 px-2 font-mono text-[10px]">
					{editorPrefs.utc ? 'UTC' : 'LOCAL'}
				</Button>
				<Button onclick={toggleMode} variant="ghost" size="icon" class="size-7">
					<SunIcon
						class="h-[1.2rem] w-[1.2rem] scale-100 rotate-0 transition-all! dark:scale-0 dark:-rotate-90"
					/>
					<MoonIcon
						class="absolute h-[1.2rem] w-[1.2rem] scale-0 rotate-90 transition-all! dark:scale-100 dark:rotate-0"
					/>
					<span class="sr-only">Toggle theme</span>
				</Button>
				<Button onclick={() => docsStore.toggle()} variant="ghost" size="icon" class="size-7">
					<BookOpenIcon class="size-4" />
					<span class="sr-only">Toggle documentation</span>
				</Button>
			</div>
		</header>
		<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 pt-0">
			<ModeWatcher />
			{#if mirStore.error}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon">
							<UnplugIcon class="text-destructive" />
						</Empty.Media>
						<Empty.Title>Failed to connect to Mir Context</Empty.Title>
						<Empty.Description>{mirStore.error}</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				{@render children()}
			{/if}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
<DocDrawer Content={docsContent} />
