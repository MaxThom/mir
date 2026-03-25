<script lang="ts">
	import { onMount } from 'svelte';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import DashboardToolbar from '$lib/domains/dashboards/components/dashboard-toolbar.svelte';
	import DashboardGrid from '$lib/domains/dashboards/components/dashboard-grid.svelte';
	import AddWidgetDialog from '$lib/domains/dashboards/components/add-widget-dialog.svelte';
	import * as Empty from '$lib/shared/components/shadcn/empty';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';

	let addWidgetOpen = $state(false);
	let refreshTick = $state(0);

	onMount(() => {
		dashboardStore.load();
	});
</script>

<div class="flex h-full flex-col">
	<DashboardToolbar onAddWidget={() => (addWidgetOpen = true)} onRefresh={() => refreshTick++} />

	<div class="flex-1 overflow-auto p-2">
		{#if dashboardStore.isLoading}
			<div class="flex h-full items-center justify-center">
				<Spinner />
			</div>
		{:else if dashboardStore.error}
			<div class="flex h-full items-center justify-center">
				<p class="text-destructive text-sm">{dashboardStore.error}</p>
			</div>
		{:else if !dashboardStore.activeDashboard}
			<div class="flex h-full items-center justify-center">
				<Empty.Root class="border-none">
					<Empty.Header>
						<Empty.Media variant="icon">
							<LayoutDashboardIcon />
						</Empty.Media>
						<Empty.Title>No Dashboard</Empty.Title>
						<Empty.Description>Create a dashboard using the + button above.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			</div>
		{:else if (dashboardStore.activeDashboard.spec.widgets?.length ?? 0) === 0}
			<div class="flex h-full items-center justify-center">
				<Empty.Root class="border-none">
					<Empty.Header>
						<Empty.Media variant="icon">
							<LayoutDashboardIcon />
						</Empty.Media>
						<Empty.Title>{dashboardStore.activeDashboard.meta.name}</Empty.Title>
						<Empty.Description>
							Click <strong>Edit</strong> then <strong>Add Widget</strong> to get started.
						</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			</div>
		{:else}
			<DashboardGrid widgets={dashboardStore.activeDashboard.spec.widgets ?? []} {refreshTick} />
		{/if}
	</div>
</div>

<AddWidgetDialog bind:open={addWidgetOpen} />
