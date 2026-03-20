<script lang="ts">
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { dashboardKey } from '../api/dashboard-api';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XIcon from '@lucide/svelte/icons/x';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { toast } from 'svelte-sonner';
	import DeleteButton from '$lib/shared/components/ui/delete-button/delete-button.svelte';

	let { onAddWidget }: { onAddWidget: () => void } = $props();

	let isCreating = $state(false);
	let newName = $state('');
	let newNamespace = $state('');
	let renameName = $state('');
	let renameNamespace = $state('');

	async function createDashboard() {
		if (!newName.trim()) return;
		try {
			await dashboardStore.create(newName.trim(), newNamespace.trim() || 'default');
			newName = '';
			newNamespace = '';
			isCreating = false;
		} catch {
			toast.error('Failed to create dashboard');
		}
	}

	function cancelCreating() {
		isCreating = false;
		newName = '';
		newNamespace = '';
	}

	async function saveEdits() {
		if (dashboardStore.activeDashboard && renameName.trim()) {
			try {
				await dashboardStore.update(dashboardStore.activeDashboard, {
					name: renameName.trim(),
					namespace: renameNamespace.trim() || 'default'
				});
			} catch {
				toast.error('Failed to rename dashboard');
				return;
			}
		}
		dashboardStore.saveEditMode();
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
	{#if dashboardStore.editMode}
		<input
			class="w-36 rounded border border-input px-2 py-1 text-sm"
			placeholder="namespace"
			bind:value={renameNamespace}
		/>
		<input
			class="w-44 rounded border border-input px-2 py-1 text-sm"
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

	<div class="flex-1"></div>

	{#if dashboardStore.isSaving}
		<Spinner class="h-4 w-4" />
	{/if}

	<!-- Create new dashboard -->
	{#if !dashboardStore.editMode}
		{#if isCreating}
			<div class="flex items-center gap-1">
				<input
					class="w-36 rounded border border-input px-2 py-1 text-sm"
					placeholder="namespace"
					bind:value={newNamespace}
					onkeydown={(e) => e.key === 'Enter' && createDashboard()}
				/>
				<input
					class="w-44 rounded border border-input px-2 py-1 text-sm"
					placeholder="name"
					bind:value={newName}
					onkeydown={(e) => e.key === 'Enter' && createDashboard()}
				/>
				<Button
					variant="ghost"
					size="icon"
					class="h-7 w-7 text-green-500"
					onclick={createDashboard}
					disabled={!newName.trim()}
					aria-label="Create dashboard"
				>
					<CheckIcon class="h-4 w-4" />
				</Button>
				<Button
					variant="ghost"
					size="icon"
					class="h-7 w-7 text-destructive"
					onclick={cancelCreating}
					aria-label="Cancel"
				>
					<XIcon class="h-4 w-4" />
				</Button>
			</div>
		{:else}
			<Button
				variant="ghost"
				size="icon"
				class="h-7 w-7"
				onclick={() => (isCreating = true)}
				aria-label="New dashboard"
			>
				<PlusIcon class="h-4 w-4" />
			</Button>
		{/if}
	{/if}

	{#if dashboardStore.activeDashboard}
		{#if dashboardStore.editMode}
			<!-- Save -->
			<Button
				variant="ghost"
				size="icon"
				class="h-7 w-7 text-green-500"
				onclick={saveEdits}
				aria-label="Save"
			>
				<CheckIcon class="h-4 w-4" />
			</Button>
			<!-- Cancel -->
			<Button
				variant="ghost"
				size="icon"
				class="h-7 w-7 text-destructive"
				onclick={() => dashboardStore.cancelEditMode()}
				aria-label="Cancel editing"
			>
				<XIcon class="h-4 w-4" />
			</Button>
			<Button size="sm" onclick={onAddWidget}>
				<PlusIcon class="mr-1 h-4 w-4" />
				Add Widget
			</Button>
			<!-- Delete -->
			<DeleteButton
				confirmValue="{dashboardStore.activeDashboard.meta.name}/{dashboardStore.activeDashboard
					.meta.namespace}"
				confirmHint="Type &quot;{dashboardStore.activeDashboard.meta.name}/{dashboardStore
					.activeDashboard.meta.namespace}&quot; to confirm."
				error={deleteError}
				{isDeleting}
				onconfirm={removeDashboard}
			/>
		{:else}
			<!-- Edit mode -->
			<Button
				variant="ghost"
				size="icon"
				class="h-7 w-7"
				onclick={() => {
					const { name, namespace } = dashboardStore.enterEditMode();
					renameName = name;
					renameNamespace = namespace;
				}}
				aria-label="Edit dashboard"
			>
				<PencilIcon class="h-4 w-4" />
			</Button>
		{/if}
	{/if}
</div>
