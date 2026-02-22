<script lang="ts">
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import BellIcon from '@lucide/svelte/icons/bell';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XCircleIcon from '@lucide/svelte/icons/x-circle';
	import InfoIcon from '@lucide/svelte/icons/info';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ChevronUpIcon from '@lucide/svelte/icons/chevron-up';
	import { cn } from '$lib/utils';
	import { SvelteSet } from 'svelte/reactivity';
	import JsonBlock from '../json-block/json-block.svelte';

	let expandedIds = $state(new Set<string>());

	function toggleExpand(id: string) {
		const next = new SvelteSet(expandedIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		expandedIds = next;
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button variant="ghost" size="icon" class="size-7" {...props}>
				<BellIcon class="size-4" />
				<span class="sr-only">Activity log</span>
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-80 p-0" sideOffset={8}>
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-3 py-2">
			<span class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Activity</span>
			{#if activityStore.entries.length > 0}
				<Button
					variant="ghost"
					size="sm"
					class="h-6 px-2 text-xs text-muted-foreground"
					onclick={() => activityStore.clear()}
				>
					Clear
				</Button>
			{/if}
		</div>

		<!-- Entry list -->
		{#if activityStore.entries.length === 0}
			<div class="px-3 py-6 text-center text-xs text-muted-foreground">No activity yet</div>
		{:else}
			<div class="max-h-[60vh] overflow-y-auto">
				{#each activityStore.entries as entry (entry.id)}
					{@const isExpanded = expandedIds.has(entry.id)}
					{@const hasPayload =
						entry.request !== undefined ||
						entry.response !== undefined ||
						entry.error !== undefined}
					<div class="border-b px-3 py-2 last:border-b-0">
						<!-- Row 1: icon + category · title + time -->
						<div class="flex items-center gap-1.5">
							{#if entry.kind === 'success'}
								<CheckIcon class="size-3.5 shrink-0 text-emerald-500" />
							{:else if entry.kind === 'error'}
								<XCircleIcon class="size-3.5 shrink-0 text-destructive" />
							{:else}
								<InfoIcon class="size-3.5 shrink-0 text-muted-foreground" />
							{/if}
							<span class="flex-1 truncate text-xs">
								<span class="font-medium">{entry.category}</span>
								<span class="text-muted-foreground"> · {entry.title}</span>
							</span>
							<TimeTooltip
								timestamp={entry.timestamp}
								utc={editorPrefs.utc}
								class="shrink-0 text-[10px] text-muted-foreground"
							/>
							{#if hasPayload}
								<button
									class={cn(
										'shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground',
										'transition-colors'
									)}
									onclick={() => toggleExpand(entry.id)}
									aria-label={isExpanded ? 'Collapse details' : 'Expand details'}
								>
									{#if isExpanded}
										<ChevronUpIcon class="size-3" />
									{:else}
										<ChevronDownIcon class="size-3" />
									{/if}
								</button>
							{/if}
						</div>

						<!-- expandable details -->
						{#if isExpanded && hasPayload}
							<div class="mt-2 space-y-1.5">
								{#if entry.error}
									<p class="text-xs text-destructive">{entry.error}</p>
								{/if}
								{#if entry.request !== undefined}
									<div>
										<p class="mb-0.5 text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
											Request
										</p>
										<JsonBlock value={entry.request} />
									</div>
								{/if}
								{#if entry.response !== undefined}
									<div>
										<p class="mb-0.5 text-[10px] font-medium tracking-wide text-muted-foreground uppercase">
											Response
										</p>
										<JsonBlock value={entry.response} />
									</div>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
