<script lang="ts">
	import { Separator } from "$lib/shared/components/shadcn/separator/index.js";
	import { cn } from "$lib/shared/utils/utils";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComponentProps<typeof Separator> = $props();
</script>

<Separator
	bind:ref
	data-slot="sidebar-separator"
	data-sidebar="separator"
	class={cn("bg-sidebar-border", className)}
	{...restProps}
/>
