<script lang="ts">
	import { page } from '$app/stores';
	import * as Empty from '$lib/components/ui/empty';
	import { Button } from '$lib/components/ui/button';
	import { CircleAlert, House, RefreshCw } from '@lucide/svelte/icons';

	function handleRefresh() {
		window.location.reload();
	}
</script>

<div class="flex h-full flex-1 items-center justify-center p-4">
	<Empty.Root>
		<Empty.Header>
			<Empty.Media variant="icon">
				<CircleAlert class="text-destructive" />
			</Empty.Media>
			<Empty.Title>
				{#if $page.status === 404}
					Page Not Found
				{:else if $page.status >= 500}
					Server Error
				{:else}
					Error {$page.status}
				{/if}
			</Empty.Title>
			<Empty.Description>
				{#if $page.status === 404}
					The page you're looking for doesn't exist or has been moved.
				{:else if $page.status >= 500}
					Something went wrong on our end. We're working to fix it.
				{:else}
					{$page.error?.message || 'An unexpected error occurred'}
				{/if}
			</Empty.Description>
		</Empty.Header>
		<Empty.Content>
			{#if $page.error?.message && $page.status !== 404}
				<div class="rounded-lg bg-muted p-3 w-full">
					<p class="font-mono text-sm">{$page.error.message}</p>
				</div>
			{/if}
			<div class="flex gap-2 w-full">
				<Button href="/" variant="default" class="flex-1">
					<House class="mr-2 h-4 w-4" />
					Go Home
				</Button>
				<Button onclick={handleRefresh} variant="outline" class="flex-1">
					<RefreshCw class="mr-2 h-4 w-4" />
					Refresh
				</Button>
			</div>
		</Empty.Content>
	</Empty.Root>
</div>
