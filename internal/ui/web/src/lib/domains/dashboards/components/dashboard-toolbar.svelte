<script lang="ts">
	import { tick } from 'svelte';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { dashboardKey } from '../api/dashboard-api';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu';
	import TimeRangePicker from '$lib/domains/devices/components/telemetry/time-range-picker.svelte';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XIcon from '@lucide/svelte/icons/x';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import EllipsisIcon from '@lucide/svelte/icons/ellipsis';
	import DeleteButton from '$lib/shared/components/ui/delete-button/delete-button.svelte';
	import RefreshButtonGroup from '$lib/shared/components/ui/refresh-button-group/refresh-button-group.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	let { onAddWidget, onRefresh }: { onAddWidget: () => void; onRefresh?: () => void } = $props();

	// ─── Tab scroll state ─────────────────────────────────────────────────────

	let scrollEl = $state<HTMLDivElement | null>(null);
	let canScrollLeft = $state(false);
	let canScrollRight = $state(false);

	function updateScrollState() {
		if (!scrollEl) return;
		canScrollLeft = scrollEl.scrollLeft > 0;
		canScrollRight = scrollEl.scrollLeft + scrollEl.clientWidth < scrollEl.scrollWidth - 1;
	}

	function scrollTabs(dir: number) {
		scrollEl?.scrollBy({ left: dir * 120, behavior: 'smooth' });
	}

	$effect(() => {
		// Re-check whenever pinned dashboards change (setTimeout lets DOM settle first)
		void dashboardStore.pinnedDashboards.length;
		setTimeout(updateScrollState, 0);
	});

	$effect(() => {
		if (!dashboardStore.editMode && !dashboardStore.isCreatingNew) return;
		function onKeydown(e: KeyboardEvent) {
			if (e.key === 'Escape') cancelEdits();
			else if (e.key === 'Enter') saveEdits();
		}
		window.addEventListener('keydown', onKeydown);
		return () => window.removeEventListener('keydown', onKeydown);
	});

	$effect(() => {
		const interval = editorPrefs.refreshInterval;
		const d = dashboardStore.activeDashboard;
		if (!d || !dashboardStore.editMode) return;
		if ((d.spec.refreshInterval ?? 10) === interval) return;
		dashboardStore.saveRefreshInterval(d, interval);
	});

	$effect(() => {
		const minutes = editorPrefs.timeMinutes;
		const d = dashboardStore.activeDashboard;
		if (!d || !dashboardStore.editMode) return;
		if ((d.spec.timeMinutes ?? 60) === minutes) return;
		dashboardStore.saveTimeMinutes(d, minutes);
	});

	// ─── Time picker state ────────────────────────────────────────────────────

	const TIME_PRESETS = [
		{ label: '1m', minutes: 1 },
		{ label: '5m', minutes: 5 },
		{ label: '10m', minutes: 10 },
		{ label: '15m', minutes: 15 },
		{ label: '30m', minutes: 30 },
		{ label: '1h', minutes: 60 },
		{ label: '3h', minutes: 180 },
		{ label: '6h', minutes: 360 },
		{ label: '12h', minutes: 720 },
		{ label: '24h', minutes: 1440 },
		{ label: '2d', minutes: 2880 },
		{ label: '7d', minutes: 10080 },
		{ label: '30d', minutes: 43200 },
		{ label: '90d', minutes: 129600 }
	] as const;

	function zoom(factor: number) {
		const f = editorPrefs.timeFilter;
		let start: Date, end: Date;
		if (f.mode === 'absolute') {
			start = f.start;
			end = f.end;
		} else {
			end = new Date();
			start = new Date(end.getTime() - f.minutes * 60 * 1000);
		}
		const delta = (end.getTime() - start.getTime()) * 0.25 * factor;
		const newStart = new Date(start.getTime() + delta);
		const newEnd = new Date(end.getTime() - delta);
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		editorPrefs.setTimeFilter({ mode: 'absolute', start: newStart, end: newEnd });
	}

	// ─── Dashboard management ─────────────────────────────────────────────────

	let renameName = $state('');
	let renameNamespace = $state('');

	function createDashboard() {
		renameName = '';
		renameNamespace = 'default';
		dashboardStore.beginCreate();
	}

	async function saveEdits() {
		if (!renameName.trim()) return;
		if (dashboardStore.isCreatingNew) {
			try {
				await dashboardStore.create(renameName.trim(), renameNamespace.trim() || 'default');
			} catch {
				// error reported via activityStore
			}
			return;
		}
		const d = dashboardStore.activeDashboard;
		if (!d) return;

		const nameChanged = renameName.trim() !== d.meta.name;
		const nsChanged = (renameNamespace.trim() || 'default') !== d.meta.namespace;

		// Exit edit mode FIRST so widget dirty effects (e.g. command payload save) run
		// before any name change triggers a {#key} remount and destroys the widgets.
		dashboardStore.saveEditMode();
		await tick();

		// After tick, activeDashboard has the latest widget state (dirty effects have flushed).
		// Send everything in a single PUT — meta rename + widgets + time settings.
		const snap = dashboardStore.activeDashboard?.spec;
		try {
			await dashboardStore.update(d, {
				...(nameChanged ? { name: renameName.trim() } : {}),
				...(nsChanged ? { namespace: renameNamespace.trim() || 'default' } : {}),
				widgets: snap?.widgets ?? d.spec.widgets,
				refreshInterval: snap?.refreshInterval,
				timeMinutes: snap?.timeMinutes
			});
		} catch {
			// error reported via activityStore
		}
	}

	async function cancelEdits() {
		if (dashboardStore.isCreatingNew) {
			dashboardStore.cancelCreate();
			return;
		}
		dashboardStore.cancelEditMode();
	}

	let deleteError = $state<string | null>(null);
	let isDeleting = $state(false);

	async function removeDashboard() {
		if (!dashboardStore.activeDashboard) return;
		isDeleting = true;
		deleteError = null;
		try {
			await dashboardStore.remove(dashboardStore.activeDashboard);
			dashboardStore.saveEditMode();
		} catch {
			deleteError = 'Failed to delete dashboard';
		} finally {
			isDeleting = false;
		}
	}
</script>

<div class="flex items-center gap-2 border-b px-4 py-2">
	<!-- Icon as dropdown trigger -->
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<button
					{...props}
					aria-label="Select dashboards"
					class="-ml-1 flex items-center gap-1 rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					<LayoutDashboardIcon class="size-4" />
					<ChevronDownIcon class="size-3" />
				</button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content>
			{#if dashboardStore.dashboards.length === 0}
				<DropdownMenu.Item disabled>No dashboards</DropdownMenu.Item>
			{:else}
				{#each dashboardStore.dashboardsByNamespace as [namespace, group], i (namespace)}
					{#if i > 0}<DropdownMenu.Separator />{/if}
					<DropdownMenu.CheckboxItem
						checked={dashboardStore.isNamespaceFullyPinned(namespace)}
						indeterminate={dashboardStore.isNamespacePartiallyPinned(namespace)}
						onCheckedChange={() => dashboardStore.toggleNamespace(namespace)}
						closeOnSelect={false}
						class="font-semibold"
					>
						{namespace}
					</DropdownMenu.CheckboxItem>
					{#each group as d (`${d.meta.namespace}/${d.meta.name}`)}
						<DropdownMenu.CheckboxItem
							checked={dashboardStore.isPinned(d)}
							onCheckedChange={() => dashboardStore.togglePinned(d)}
							closeOnSelect={false}
							class="pl-6"
						>
							{d.meta.name}
						</DropdownMenu.CheckboxItem>
					{/each}
				{/each}
			{/if}
		</DropdownMenu.Content>
	</DropdownMenu.Root>

	<!-- Tab bar / edit input -->
	{#if dashboardStore.isCreatingNew || dashboardStore.editMode}
		<input
			class="w-36 rounded border border-input px-2 py-1 text-sm"
			placeholder="namespace"
			bind:value={renameNamespace}
		/>
		<input
			class="w-44 rounded border px-2 py-1 text-sm {renameName.trim()
				? 'border-input'
				: 'border-destructive'}"
			placeholder="name"
			bind:value={renameName}
		/>
		<div class="flex-1"></div>
	{:else}
		<div class="flex min-w-0 flex-1 items-center gap-1">
			{#if canScrollLeft}
				<button
					onclick={() => scrollTabs(-1)}
					aria-label="Scroll tabs left"
					class="flex shrink-0 items-center rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					<ChevronLeftIcon class="size-3" />
				</button>
			{/if}
			<div class="min-w-0 flex-1 overflow-hidden">
				<div
					bind:this={scrollEl}
					onscroll={updateScrollState}
					class="tab-scroll flex items-center gap-1 overflow-x-auto"
				>
					{#each dashboardStore.pinnedDashboards as d (`${d.meta.namespace}/${d.meta.name}`)}
						{@const isActive =
							!!dashboardStore.activeDashboard &&
							dashboardKey(d) === dashboardKey(dashboardStore.activeDashboard)}
						<Button
							variant={isActive ? 'secondary' : 'ghost'}
							size="sm"
							class="shrink-0"
							onclick={() => dashboardStore.setActive(d)}
						>
							{d.meta.name}
						</Button>
					{/each}
				</div>
			</div>
			{#if canScrollRight}
				<button
					onclick={() => scrollTabs(1)}
					aria-label="Scroll tabs right"
					class="flex shrink-0 items-center rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					<ChevronRightIcon class="size-3" />
				</button>
			{/if}
		</div>
	{/if}

	{#if dashboardStore.isCreatingNew || dashboardStore.editMode}
		<!-- Save -->
		<Button
			variant="ghost"
			size="icon"
			class="h-7 w-7 text-green-500"
			onclick={saveEdits}
			disabled={!renameName.trim()}
			aria-label="Save"
		>
			<CheckIcon class="h-4 w-4" />
		</Button>
		<!-- Cancel -->
		<Button
			variant="ghost"
			size="icon"
			class="h-7 w-7 text-destructive"
			onclick={cancelEdits}
			aria-label="Cancel editing"
		>
			<XIcon class="h-4 w-4" />
		</Button>
		<!-- Delete (only for existing dashboards) -->
		{#if !dashboardStore.isCreatingNew && dashboardStore.activeDashboard}
			<DeleteButton
				confirmValue="{dashboardStore.activeDashboard.meta.namespace}/{dashboardStore
					.activeDashboard.meta.name}"
				confirmHint="Type &quot;{dashboardStore.activeDashboard.meta.namespace}/{dashboardStore
					.activeDashboard.meta.name}&quot; to confirm."
				error={deleteError}
				{isDeleting}
				onconfirm={removeDashboard}
			/>
		{/if}
		<Button size="sm" onclick={onAddWidget}>
			<PlusIcon class="mr-1 h-4 w-4" />
			Add Widget
		</Button>
	{/if}

	{#if dashboardStore.activeDashboard}
		<!-- Global time range picker -->
		<button
			onclick={() => zoom(-1)}
			title="Zoom out"
			class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
		>
			<ZoomOutIcon class="size-3.5" />
		</button>
		<TimeRangePicker
			timeFilter={editorPrefs.timeFilter}
			presets={TIME_PRESETS}
			ontimechange={(f) => editorPrefs.setTimeFilter(f)}
		/>
		<RefreshButtonGroup {onRefresh} isLoading={dashboardStore.isRefreshing} />
	{/if}

	<!-- Dashboard management dropdown -->
	{#if !dashboardStore.editMode && !dashboardStore.isCreatingNew}
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<button
						{...props}
						aria-label="Dashboard actions"
						class="flex items-center rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
					>
						<EllipsisIcon class="size-3.5" />
					</button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end">
				{#if dashboardStore.activeDashboard}
					<DropdownMenu.Item
						onclick={async () => {
							const { name, namespace } = await dashboardStore.enterEditMode();
							renameName = name;
							renameNamespace = namespace;
						}}
					>
						<PencilIcon class="mr-2 size-3.5" />
						Edit Dashboard
					</DropdownMenu.Item>
				{/if}
				<DropdownMenu.Item onclick={createDashboard}>
					<PlusIcon class="mr-2 size-3.5" />
					New Dashboard
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}
</div>

<style>
	/*
	 * Scrollbar trick: padding-bottom pushes the track into the overflow area,
	 * margin-bottom cancels the extra height, and the parent overflow:hidden clips it.
	 * Net result: zero height impact, scrollbar appears overlaid on hover.
	 */
	.tab-scroll {
		padding-bottom: 6px;
		margin-bottom: -6px;
		scrollbar-width: none; /* Firefox: no native scrollbar, still scrollable */
	}
	.tab-scroll::-webkit-scrollbar {
		height: 2px;
	}
	.tab-scroll::-webkit-scrollbar-track {
		background: transparent;
	}
	.tab-scroll::-webkit-scrollbar-thumb {
		border-radius: 9999px;
		background: transparent;
	}
	.tab-scroll:hover::-webkit-scrollbar-thumb {
		background: hsl(var(--border) / 0.6);
	}
</style>
