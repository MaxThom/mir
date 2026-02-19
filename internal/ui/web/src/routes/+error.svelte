<script lang="ts">
	import { page } from '$app/stores';
	import * as Card from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { CircleAlert, House, RefreshCw } from '@lucide/svelte/icons';

	function handleRefresh() {
		window.location.reload();
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-background p-4">
	<Card.Root class="w-full max-w-md">
		<Card.Header>
			<div class="flex items-center gap-2">
				<CircleAlert class="h-6 w-6 text-destructive" />
				<Card.Title class="text-2xl">
					{#if $page.status === 404}
						Page Not Found
					{:else if $page.status >= 500}
						Server Error
					{:else}
						Error {$page.status}
					{/if}
				</Card.Title>
			</div>
		</Card.Header>
		<Card.Content class="space-y-4">
			<p class="text-muted-foreground">
				{#if $page.status === 404}
					The page you're looking for doesn't exist or has been moved.
				{:else if $page.status >= 500}
					Something went wrong on our end. We're working to fix it.
				{:else}
					{$page.error?.message || 'An unexpected error occurred'}
				{/if}
			</p>

			{#if $page.error?.message && $page.status !== 404}
				<div class="rounded-lg bg-muted p-3">
					<p class="font-mono text-sm">{$page.error.message}</p>
				</div>
			{/if}
		</Card.Content>
		<Card.Footer class="flex gap-2">
			<Button href="/" variant="default" class="flex-1">
				<House class="mr-2 h-4 w-4" />
				Go Home
			</Button>
			<Button onclick={handleRefresh} variant="outline" class="flex-1">
				<RefreshCw class="mr-2 h-4 w-4" />
				Refresh
			</Button>
		</Card.Footer>
	</Card.Root>
</div>

<style>
	:global(body) {
		overflow: hidden;
	}
</style>
