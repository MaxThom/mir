<script lang="ts">
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { Button } from '$lib/shared/components/shadcn/button';
	import * as DropdownMenu from '$lib/shared/components/shadcn/dropdown-menu/index.js';

	let {
		pageIndex,
		pageSize,
		totalRows,
		pageCount,
		canPreviousPage,
		canNextPage,
		onpreviouspage,
		onnextpage,
		onpagesizechange
	}: {
		pageIndex: number;
		pageSize: number;
		totalRows: number;
		pageCount: number;
		canPreviousPage: boolean;
		canNextPage: boolean;
		onpreviouspage: () => void;
		onnextpage: () => void;
		onpagesizechange: (size: number) => void;
	} = $props();
</script>

<div class="flex items-center justify-between border-t px-6 py-3 text-sm text-muted-foreground">
	<span class="tabular-nums text-xs">
		{pageIndex * pageSize + 1}–{Math.min((pageIndex + 1) * pageSize, totalRows)} of {totalRows}
	</span>
	<div class="flex items-center gap-2">
		<Button variant="outline" size="icon-sm" disabled={!canPreviousPage} onclick={onpreviouspage}>
			<ChevronLeftIcon class="h-3.5 w-3.5" />
		</Button>
		<span class="tabular-nums text-xs">{pageIndex + 1} / {pageCount}</span>
		<Button variant="outline" size="icon-sm" disabled={!canNextPage} onclick={onnextpage}>
			<ChevronRightIcon class="h-3.5 w-3.5" />
		</Button>
	</div>
	<div class="flex items-center gap-2">
		<span class="text-xs">Rows</span>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button variant="outline" size="sm" class="h-7 gap-1 px-2.5 text-xs tabular-nums" {...props}>
						{pageSize}
						<ChevronDownIcon class="h-3 w-3 opacity-50" />
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align="end">
				{#each [5, 10, 20, 50] as size (size)}
					<DropdownMenu.Item
						class="tabular-nums {size === pageSize ? 'font-medium text-foreground' : ''}"
						onclick={() => onpagesizechange(size)}
					>
						{size}
					</DropdownMenu.Item>
				{/each}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
</div>
