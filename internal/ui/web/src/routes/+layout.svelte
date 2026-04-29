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
	import * as Dialog from '$lib/shared/components/shadcn/dialog/index.js';
	import { contextService } from '$lib/domains/contexts/services/contexts';
	import type { Context } from '$lib/domains/contexts/types/types';
	import LockIcon from '@lucide/svelte/icons/lock';
	import EyeIcon from '@lucide/svelte/icons/eye';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';

	// Generate breadcrumbs from current pathname
	let breadcrumbs = $derived(generateBreadcrumbs(page.url.pathname));
	let docsContent = $derived(docsStore.getContent(page.url.pathname));

	// Password gate
	let passwordGateOpen = $state(false);
	let passwordGateCtx = $state<Context | null>(null);
	let passwordGateValue = $state('');
	let passwordGateError = $state<string | null>(null);
	let passwordGateSubmitting = $state(false);
	let passwordGateVisible = $state(false);
	let previousContextName = $state<string | null>(null);

	async function submitPasswordGate() {
		if (!passwordGateCtx) return;
		passwordGateSubmitting = true;
		passwordGateError = null;
		try {
			const creds = await contextService.getCredentials(passwordGateCtx.name, passwordGateValue);
			const ctx = passwordGateCtx;
			// Clear sentinel BEFORE closing so onOpenChange doesn't treat this as an external dismiss
			passwordGateCtx = null;
			passwordGateOpen = false;
			passwordGateValue = '';
			passwordGateVisible = false;
			previousContextName = ctx.name;
			mirStore.connect(ctx, null, creds);
		} catch (e) {
			const status = (e as Error & { status?: number }).status;
			passwordGateError =
				status === 401 ? 'Invalid password' :
				status === 404 ? 'Context not found' :
				status === 500 ? 'Server error — check the credentials file path' :
				'Cannot reach server';
		} finally {
			passwordGateSubmitting = false;
		}
	}

	function cancelPasswordGate() {
		const prev = previousContextName;
		// Clear sentinel BEFORE closing so onOpenChange doesn't recurse
		passwordGateCtx = null;
		passwordGateOpen = false;
		passwordGateValue = '';
		passwordGateError = null;
		passwordGateVisible = false;
		if (prev) {
			contextStore.setActiveContext(prev);
		}
	}

	// VIM gate
	let vimGateOpen = $state(false);
	let vimGateAnswer = $state('');
	let vimGateWrong = $state(false);

	function handleVimToggle() {
		if (editorPrefs.vim) {
			editorPrefs.setVim(false);
		} else {
			vimGateAnswer = '';
			vimGateWrong = false;
			vimGateOpen = true;
		}
	}

	function submitVimGate() {
		if (vimGateAnswer.trim() === ':q') {
			editorPrefs.setVim(true);
			vimGateOpen = false;
		} else {
			vimGateWrong = true;
		}
	}

	onMount(async () => {
		await contextStore.initialize();
		themeStore.init();
	});

	$effect(() => {
		const ctx = contextStore.activeContext;
		if (!ctx) return;

		if (ctx.secured) {
			untrack(() => {
				passwordGateCtx = ctx;
				passwordGateOpen = true;
			});
		} else {
			untrack(() => {
				previousContextName = ctx.name;
				mirStore.connect(ctx);
			});
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
	<AppSidebar navMain={sidebarData.navMain} />
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
				<Button onclick={handleVimToggle} variant="ghost" class="h-7 px-2 font-mono text-[10px] {editorPrefs.vim ? 'bg-secondary text-secondary-foreground' : ''}">
					VIM
				</Button>
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
					<Empty.Content>
						{#if contextStore.activeContext?.secured}
							<Button
								size="sm"
								onclick={() => {
									if (contextStore.activeContext) {
										passwordGateCtx = contextStore.activeContext;
										passwordGateOpen = true;
									}
								}}
							>
								Sign in again
							</Button>
						{:else if contextStore.activeContext}
							<Button size="sm" onclick={() => mirStore.connect(contextStore.activeContext!)}>
								Retry connection
							</Button>
						{/if}
					</Empty.Content>
				</Empty.Root>
			{:else}
				{@render children()}
			{/if}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
<DocDrawer Content={docsContent} />
<Toaster position="bottom-left" duration={8000} />

<Dialog.Root
	bind:open={passwordGateOpen}
	onOpenChange={(open) => {
		// passwordGateCtx is cleared before programmatic closes (success/cancel button).
		// If it's still set when the dialog closes, the user dismissed it externally
		// (outside click or Escape) — treat that the same as cancel.
		if (!open && passwordGateCtx !== null) cancelPasswordGate();
	}}
>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<LockIcon class="size-4" />
				Sign in to {passwordGateCtx?.name}
			</Dialog.Title>
			<Dialog.Description>This context requires a password.</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-3 py-2">
			<div class="relative">
				<input
					type={passwordGateVisible ? 'text' : 'password'}
					placeholder="Password"
					class="w-full rounded-md border border-input bg-background px-3 py-2 pr-9 text-sm focus:outline-none focus:ring-1 focus:ring-ring {passwordGateError ? 'border-destructive focus:ring-destructive' : ''}"
					bind:value={passwordGateValue}
					disabled={passwordGateSubmitting}
					onkeydown={(e) => e.key === 'Enter' && submitPasswordGate()}
				/>
				<button
					type="button"
					class="absolute inset-y-0 right-0 flex items-center px-2.5 text-muted-foreground hover:text-foreground"
					onmousedown={(e) => { e.preventDefault(); passwordGateVisible = !passwordGateVisible; }}
					tabindex="-1"
				>
					{#if passwordGateVisible}
						<EyeOffIcon class="size-4" />
					{:else}
						<EyeIcon class="size-4" />
					{/if}
				</button>
			</div>
			{#if passwordGateError}
				<p class="text-xs text-destructive">{passwordGateError}</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="ghost" size="sm" onclick={cancelPasswordGate} disabled={passwordGateSubmitting}>Cancel</Button>
			<Button size="sm" onclick={submitPasswordGate} disabled={passwordGateSubmitting}>
				{passwordGateSubmitting ? 'Signing in…' : 'Sign in'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={vimGateOpen}>
	<Dialog.Content class="max-w-sm">
		<Dialog.Header>
			<Dialog.Title class="font-mono">VIM Proficiency Test</Dialog.Title>
			<Dialog.Description>Answer correctly to unlock VIM mode.</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-3 py-2">
			<p class="text-sm font-medium">How do I quit VIM?</p>
			<input
				class="w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm focus:outline-none focus:ring-1 focus:ring-ring {vimGateWrong ? 'border-destructive focus:ring-destructive' : ''}"
				placeholder="Type your answer..."
				bind:value={vimGateAnswer}
				onkeydown={(e) => e.key === 'Enter' && submitVimGate()}
			/>
			{#if vimGateWrong}
				<p class="text-xs text-destructive">Wrong. You are trapped forever.</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="ghost" size="sm" onclick={() => (vimGateOpen = false)}>Cancel</Button>
			<Button size="sm" onclick={submitVimGate}>Confirm</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
