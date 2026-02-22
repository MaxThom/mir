<script lang="ts">
	import * as Tooltip from '$lib/shared/components/shadcn/tooltip';
	import { relativeTime, formatFullDate } from '$lib/shared/utils/time';
	import { cn } from '$lib/utils';

	let {
		timestamp,
		utc = false,
		prefix = '',
		class: cls = ''
	}: {
		timestamp: Date;
		utc?: boolean;
		prefix?: string;
		class?: string;
	} = $props();
</script>

<Tooltip.Root>
	<Tooltip.Trigger class={cn('cursor-default underline decoration-dotted underline-offset-2', cls)}>
		{prefix}{relativeTime(timestamp.getTime() / 1000)}
	</Tooltip.Trigger>
	<Tooltip.Content>
		{formatFullDate(timestamp.getTime() / 1000, utc)}
	</Tooltip.Content>
</Tooltip.Root>
