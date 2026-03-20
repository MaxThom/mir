<script lang="ts">
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { toast } from 'svelte-sonner';

	let { onAddWidget }: { onAddWidget: () => void } = $props();

	let isCreating = $state(false);
	let newName = $state('');
	let isRenaming = $state(false);
	let renameName = $state('');

	async function createDashboard() {
		if (!newName.trim()) return;
		try {
			await dashboardStore.create(newName.trim());
			newName = '';
			isCreating = false;
		} catch {
			toast.error('Failed to create dashboard');
		}
	}

	async function startRename() {
		renameName = dashboardStore.activeDashboard?.meta.name ?? '';
		isRenaming = true;
	}

	async function confirmRename() {
		if (!dashboardStore.activeDashboard || !renameName.trim()) return;
		try {
			await dashboardStore.rename(dashboardStore.activeDashboard, renameName.trim());
			isRenaming = false;
		} catch {
			toast.error('Failed to rename dashboard');
		}
	}

	async function removeDashboard() {
		if (!dashboardStore.activeDashboard) return;
		try {
			await dashboardStore.remove(dashboardStore.activeDashboard);
		} catch {
			toast.error('Failed to delete dashboard');
		}
	}
</script>

<div class="flex items-center gap-2 border-b px-4 py-2">
	<!-- Dashboard switcher -->
	<LayoutDashboardIcon class="text-muted-foreground h-4 w-4 shrink-0" />

	{#if dashboardStore.dashboards.length === 0}
		<span class="text-muted-foreground text-sm">No dashboards</span>
	{:else}
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				<Button variant="ghost" size="sm" class="gap-1">
					{#if isRenaming}
						<input
							class="border-input w-32 rounded border px-1 text-sm"
							bind:value={renameName}
							onkeydown={(e) => e.key === 'Enter' && confirmRename()}
							onclick={(e) => e.stopPropagation()}
						/>
						<Button
							variant="ghost"
							size="icon"
							class="h-5 w-5"
							onclick={(e) => { e.stopPropagation(); confirmRename(); }}
						>
							<CheckIcon class="h-3 w-3" />
						</Button>
					{:else}
						<span class="max-w-32 truncate">{dashboardStore.activeDashboard?.meta.name ?? '—'}</span>
						<ChevronDownIcon class="h-3 w-3" />
					{/if}
				</Button>
			</DropdownMenu.Trigger>
			<DropdownMenu.Content>
				{#each dashboardStore.dashboards as d (`${d.meta.namespace}/${d.meta.name}`)}
					<DropdownMenu.Item onclick={() => dashboardStore.setActive(d)}>
						{d.meta.name}
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	{/if}

	<!-- Create new dashboard -->
	{#if isCreating}
		<div class="flex items-center gap-1">
			<input
				class="border-input w-32 rounded border px-2 py-1 text-sm"
				placeholder="Dashboard name"
				bind:value={newName}
				onkeydown={(e) => e.key === 'Enter' && createDashboard()}
			/>
			<Button size="sm" onclick={createDashboard} disabled={!newName.trim()}>Create</Button>
			<Button variant="ghost" size="sm" onclick={() => { isCreating = false; newName = ''; }}>
				Cancel
			</Button>
		</div>
	{:else}
		<Button variant="ghost" size="icon" class="h-7 w-7" onclick={() => (isCreating = true)} aria-label="New dashboard">
			<PlusIcon class="h-4 w-4" />
		</Button>
	{/if}

	<div class="flex-1"></div>

	{#if dashboardStore.isSaving}
		<Spinner class="h-4 w-4" />
	{/if}

	{#if dashboardStore.activeDashboard}
		<!-- Rename -->
		<Button variant="ghost" size="icon" class="h-7 w-7" onclick={startRename} aria-label="Rename dashboard">
			<PencilIcon class="h-4 w-4" />
		</Button>

		<!-- Delete -->
		<Button variant="ghost" size="icon" class="text-destructive h-7 w-7" onclick={removeDashboard} aria-label="Delete dashboard">
			<Trash2Icon class="h-4 w-4" />
		</Button>

		<!-- Edit mode toggle -->
		<Button
			variant={dashboardStore.editMode ? 'default' : 'outline'}
			size="sm"
			onclick={() => dashboardStore.toggleEditMode()}
		>
			{dashboardStore.editMode ? 'Done' : 'Edit'}
		</Button>

		{#if dashboardStore.editMode}
			<Button size="sm" onclick={onAddWidget}>
				<PlusIcon class="mr-1 h-4 w-4" />
				Add Widget
			</Button>
		{/if}
	{/if}
</div>
