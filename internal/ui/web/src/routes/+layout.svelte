<script lang="ts">
	import './layout.css';
	import { ModeWatcher } from 'mode-watcher';
	import SunIcon from '@lucide/svelte/icons/sun';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import FlameIcon from '@lucide/svelte/icons/flame';
	import MoonStarIcon from '@lucide/svelte/icons/moon-star';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SparklesIcon from '@lucide/svelte/icons/sparkles';
	import CoffeeIcon from '@lucide/svelte/icons/coffee';
	import UnplugIcon from '@lucide/svelte/icons/unplug';
	import BookOpenIcon from '@lucide/svelte/icons/book-open';
	import { ActivityLog } from '$lib/domains/activity/components/activity-log';
	import { Toaster } from '$lib/shared/components/shadcn/sonner';
	import * as Empty from '$lib/shared/components/shadcn/empty';

	import { Button } from '$lib/shared/components/shadcn/button/index.js';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu/index.js';
	import { onMount, untrack } from 'svelte';
	import { themeStore } from '$lib/shared/stores/theme.svelte';

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
		themeStore.init();
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
				<ActivityLog />
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
				<DropdownMenu.Root onOpenChange={(open) => { if (!open) themeStore.revert(); }}>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="ghost" size="icon" class="size-7">
								{#if themeStore.current === 'dusk'}
									<MoonIcon class="size-4" />
								{:else if themeStore.current === 'aurora'}
									<SparklesIcon class="size-4" />
								{:else if themeStore.current === 'midnight'}
									<MoonStarIcon class="size-4" />
								{:else if themeStore.current === 'rust'}
									<FlameIcon class="size-4" />
								{:else if themeStore.current === 'hacker'}
									<TerminalIcon class="size-4" />
								{:else if themeStore.current === 'mocha'}
									<CoffeeIcon class="size-4" />
								{:else}
									<SunIcon class="size-4" />
								{/if}
								<span class="sr-only">Theme</span>
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end" class="min-w-32">
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('dawn')} onclick={() => themeStore.set('dawn')} class="gap-2">
							<SunIcon class="size-3.5" /> Dawn
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('rust')} onclick={() => themeStore.set('rust')} class="gap-2">
							<FlameIcon class="size-3.5" /> Rust
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('mocha')} onclick={() => themeStore.set('mocha')} class="gap-2">
							<CoffeeIcon class="size-3.5" /> Mocha
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('dusk')} onclick={() => themeStore.set('dusk')} class="gap-2">
							<MoonIcon class="size-3.5" /> Dusk
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('aurora')} onclick={() => themeStore.set('aurora')} class="gap-2">
							<SparklesIcon class="size-3.5" /> Aurora
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('midnight')} onclick={() => themeStore.set('midnight')} class="gap-2">
							<MoonStarIcon class="size-3.5" /> Midnight
						</DropdownMenu.Item>
						<DropdownMenu.Item onmouseenter={() => themeStore.preview('hacker')} onclick={() => themeStore.set('hacker')} class="gap-2">
							<TerminalIcon class="size-3.5" /> Hacker
						</DropdownMenu.Item>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
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
<Toaster position="bottom-left" duration={8000} />
