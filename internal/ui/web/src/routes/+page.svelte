<script lang="ts">
	import { onMount } from 'svelte';
	import * as Empty from '$lib/components/ui/empty';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';

	onMount(() => {
		// goto('/dashboard');
	});
</script>

<div class="flex min-h-screen items-center justify-center">
	<p class="text-muted-foreground">
		<Empty.Root class="border-none">
			<Empty.Header>
				<Empty.Media variant="icon">
					<SatelliteDishIcon />
				</Empty.Media>
				<Empty.Title>Dashboard</Empty.Title>
				<Empty.Description>Coming soon!</Empty.Description>
			</Empty.Header>
		</Empty.Root>
	</p>
</div>
