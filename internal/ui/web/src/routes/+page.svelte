<script lang="ts">
	import { onMount } from 'svelte';
	import { welcomeStore } from '$lib/domains/welcome/stores/welcome.svelte';
	import DashboardGrid from '$lib/domains/dashboards/components/dashboard-grid.svelte';
	import * as Empty from '$lib/shared/components/shadcn/empty';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';

	let refreshTick = $state(0);

	onMount(() => {
		welcomeStore.load();
		const id = setInterval(() => refreshTick++, 10_000);
		return () => clearInterval(id);
	});
</script>

<div class="flex h-full flex-col">

	<div class="flex-1 overflow-auto p-2">
		{#if welcomeStore.isLoading}
			<div class="flex h-full items-center justify-center">
				<Spinner />
			</div>
		{:else if welcomeStore.error}
			<div class="flex h-full items-center justify-center">
				<p class="text-sm text-destructive">{welcomeStore.error}</p>
			</div>
		{:else if !welcomeStore.dashboard}
			<div class="flex h-full items-center justify-center">
				<Empty.Root class="border-none">
					<Empty.Header>
						<Empty.Media variant="icon">
							<LayoutDashboardIcon />
						</Empty.Media>
						<Empty.Title>Welcome to Mir</Empty.Title>
						<Empty.Description>Welcome dashboard not found.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			</div>
		{:else}
			{#key `${welcomeStore.dashboard.meta.namespace}/${welcomeStore.dashboard.meta.name}`}
				<DashboardGrid
					widgets={welcomeStore.dashboard.spec.widgets ?? []}
					{refreshTick}
					readonly={true}
				/>
			{/key}
		{/if}
	</div>
</div>
