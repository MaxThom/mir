<script lang="ts">
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { Button } from '$lib/components/ui/button';

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
	<span class="tabular-nums">
		{pageIndex * pageSize + 1}–{Math.min((pageIndex + 1) * pageSize, totalRows)} of {totalRows}
	</span>
	<div class="flex items-center gap-2">
		<Button variant="outline" size="icon-sm" disabled={!canPreviousPage} onclick={onpreviouspage}>
			<ChevronLeftIcon class="h-3.5 w-3.5" />
		</Button>
		<span class="tabular-nums">{pageIndex + 1} / {pageCount}</span>
		<Button variant="outline" size="icon-sm" disabled={!canNextPage} onclick={onnextpage}>
			<ChevronRightIcon class="h-3.5 w-3.5" />
		</Button>
	</div>
	<div class="flex items-center gap-2">
		<span>Rows</span>
		<select
			class="h-7 rounded-md border bg-background px-2 text-xs"
			value={pageSize}
			onchange={(e) => onpagesizechange(Number((e.target as HTMLSelectElement).value))}
		>
			{#each [5, 10, 20, 50] as size, i (i)}
				<option value={size}>{size}</option>
			{/each}
		</select>
	</div>
</div>
