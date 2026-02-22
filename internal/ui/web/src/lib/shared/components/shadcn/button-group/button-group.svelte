<script lang="ts">
	import { cn } from '$lib/utils';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		class: className,
		orientation = 'horizontal',
		children,
		...restProps
	}: HTMLAttributes<HTMLDivElement> & { orientation?: 'horizontal' | 'vertical' } = $props();
</script>

<div
	data-slot="button-group"
	data-orientation={orientation}
	class={cn(
		'inline-flex items-center',
		orientation === 'horizontal'
			? '[&>*:not(:first-child)]:-ml-px [&>*:not(:first-child)]:rounded-l-none [&>*:not(:last-child)]:rounded-r-none'
			: 'flex-col [&>*:not(:first-child)]:-mt-px [&>*:not(:first-child)]:rounded-t-none [&>*:not(:last-child)]:rounded-b-none',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</div>
