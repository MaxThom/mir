<script lang="ts">
	import ArrowUpIcon from '@lucide/svelte/icons/arrow-up';
	import { Button } from '$lib/components/ui/button/index.js';
</script>

<h1>Welcome to SvelteKit</h1>
<p>Visit <a href="https://svelte.dev/docs/kit">svelte.dev/docs/kit</a> to read the documentation</p>

<br />
<Button>Click me</Button>

<div class="flex flex-wrap items-center gap-2 md:flex-row">
	<Button variant="outline">Button</Button>
	<Button variant="outline" size="icon" aria-label="Submit">
		<ArrowUpIcon />
	</Button>
</div>
