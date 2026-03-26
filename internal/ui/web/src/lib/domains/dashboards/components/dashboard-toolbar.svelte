<script lang="ts">
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
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import DeleteButton from '$lib/shared/components/ui/delete-button/delete-button.svelte';
	import RefreshButtonGroup from '$lib/shared/components/ui/refresh-button-group/refresh-button-group.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';

	let { onAddWidget, onRefresh }: { onAddWidget: () => void; onRefresh?: () => void } = $props();

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
		if (!d) return;
		if ((d.spec.refreshInterval ?? 10) === interval) return;
		dashboardStore.saveRefreshInterval(d, interval);
	});

	$effect(() => {
		const minutes = editorPrefs.timeMinutes;
		const d = dashboardStore.activeDashboard;
		if (!d) return;
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
		if (dashboardStore.activeDashboard) {
			try {
				await dashboardStore.update(dashboardStore.activeDashboard, {
					name: renameName.trim(),
					namespace: renameNamespace.trim() || 'default'
				});
			} catch {
				return; // error reported via activityStore
			}
		}
		dashboardStore.saveEditMode();
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
			<Button variant="ghost" size="sm" class="-ml-4 gap-1" aria-label="Select dashboards">
				<LayoutDashboardIcon class="h-4 w-4" />
				<ChevronDownIcon class="h-3 w-3" />
			</Button>
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
	{:else}
		{#each dashboardStore.pinnedDashboards as d (`${d.meta.namespace}/${d.meta.name}`)}
			{@const isActive =
				!!dashboardStore.activeDashboard &&
				dashboardKey(d) === dashboardKey(dashboardStore.activeDashboard)}
			<Button
				variant={isActive ? 'secondary' : 'ghost'}
				size="sm"
				onclick={() => dashboardStore.setActive(d)}
			>
				{d.meta.name}
			</Button>
		{/each}
	{/if}

	<!-- Create new dashboard -->
	{#if !dashboardStore.editMode && !dashboardStore.isCreatingNew}
		<button
			onclick={createDashboard}
			aria-label="New dashboard"
			class="flex items-center rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
		>
			<PlusIcon class="size-3.5" />
		</button>
	{/if}

	<div class="flex-1"></div>

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
				confirmHint="Type &quot;{dashboardStore.activeDashboard.meta.name}/{dashboardStore
					.activeDashboard.meta.namespace}&quot; to confirm."
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
		<!-- Edit button -->
		{#if !dashboardStore.editMode && !dashboardStore.isCreatingNew}
			<button
				onclick={() => {
					const { name, namespace } = dashboardStore.enterEditMode();
					renameName = name;
					renameNamespace = namespace;
				}}
				aria-label="Edit dashboard"
				class="flex items-center rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
			>
				<PencilIcon class="size-3.5" />
			</button>
		{/if}
		{#if dashboardStore.editMode || dashboardStore.isCreatingNew}
			<Separator orientation="vertical" class="data-[orientation=vertical]:h-2" />
		{/if}
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
</div>
