<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import type { Snippet } from 'svelte';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import TimeRangePicker from './time-range-picker.svelte';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	type TimeFilter =
		| { mode: 'relative'; minutes: number }
		| { mode: 'absolute'; start: Date; end: Date };

	let {
		measurementName = null,
		measurementError = null,
		grafanaUrl = null,
		timeFilter = $bindable<TimeFilter>({ mode: 'relative', minutes: 5 }),
		fullscreen = $bindable(false),
		queryData = null,
		compact = false,
		showZoom = true,
		presets,
		onQuery,
		toolbarEnd = undefined,
		compactDropdownExtra = undefined
	}: {
		measurementName?: string | null;
		measurementError?: string | null;
		grafanaUrl?: string | null;
		timeFilter?: TimeFilter;
		fullscreen?: boolean;
		queryData?: QueryData | null;
		compact?: boolean;
		showZoom?: boolean;
		presets?: readonly { label: string; minutes: number }[];
		onQuery: () => void;
		// Optional extra buttons rendered just before the fullscreen button
		toolbarEnd?: Snippet;
		// Optional extra content rendered inside the compact overflow dropdown (below action buttons)
		compactDropdownExtra?: Snippet;
	} = $props();

	let overflowOpen = $state(false);
	let copied = $state(false);

	function getTimeRange(): { start: Date; end: Date } {
		const f = timeFilter;
		if (f.mode === 'absolute') {
			const start = f.start;
			const end = f.end.getTime() <= start.getTime() ? new Date(start.getTime() + 1000) : f.end;
			return { start, end };
		}
		const end = new Date();
		const start = new Date(end.getTime() - f.minutes * 60 * 1000);
		return { start, end };
	}

	function zoom(factor: number) {
		const { start, end } = getTimeRange();
		const delta = (end.getTime() - start.getTime()) * 0.25 * factor;
		const newStart = new Date(start.getTime() + delta);
		const newEnd = new Date(end.getTime() - delta);
		if (newEnd.getTime() <= newStart.getTime() + 1000) return;
		timeFilter = { mode: 'absolute', start: newStart, end: newEnd };
		onQuery();
	}

	function dateToRFC3339(date: Date): string {
		if (editorPrefs.utc) return date.toISOString();
		const offset = -date.getTimezoneOffset();
		const sign = offset >= 0 ? '+' : '-';
		const pad = (n: number, w = 2) => String(n).padStart(w, '0');
		const tz = `${sign}${pad(Math.floor(Math.abs(offset) / 60))}:${pad(Math.abs(offset) % 60)}`;
		return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}.${pad(date.getMilliseconds(), 3)}${tz}`;
	}

	async function copyAsCsv() {
		if (!queryData) return;
		const { headers, rows } = queryData;
		const lines = [
			headers.join(','),
			...rows.map((row) =>
				headers
					.map((h) => {
						const v = row.values[h] ?? null;
						if (v === null || v === undefined) return '';
						if (v instanceof Date) return dateToRFC3339(v);
						if (typeof v === 'boolean') return v ? 'true' : 'false';
						return String(v);
					})
					.join(',')
			)
		];
		await navigator.clipboard.writeText(lines.join('\n'));
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="flex shrink-0 items-center gap-2 border-b px-4 py-1.5">
	{#if measurementName}
		<span class="min-w-0 truncate font-mono text-sm font-medium">{measurementName}</span>
		{#if grafanaUrl}
			<a
				href={grafanaUrl}
				target="_blank"
				rel="noreferrer"
				title="Open in Grafana"
				class="inline-flex shrink-0 -translate-y-0.5 items-center text-muted-foreground transition-colors hover:text-foreground"
			>
				<ExternalLinkIcon class="size-3.5" />
			</a>
		{/if}
		{#if measurementError}
			<span class="text-xs text-destructive">{measurementError}</span>
		{/if}
	{/if}

	{#if !compact}
		<div class="ml-auto flex shrink-0 items-center gap-1">
			{#if showZoom}
				<button
					onclick={() => zoom(-1)}
					title="Zoom out"
					class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
				>
					<ZoomOutIcon class="size-3.5" />
				</button>
			{/if}
			{#if presets}
				<TimeRangePicker
					{timeFilter}
					{presets}
					ontimechange={(f) => { timeFilter = f; onQuery(); }}
				/>
			{/if}
			{@render toolbarEnd?.()}
			<button
				onclick={() => (fullscreen = !fullscreen)}
				title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
				class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
			>
				{#if fullscreen}
					<MinimizeIcon class="size-3.5" />
				{:else}
					<MaximizeIcon class="size-3.5" />
				{/if}
			</button>
			<button
				onclick={copyAsCsv}
				disabled={!queryData}
				title="Copy data as CSV"
				class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-40"
			>
				{#if copied}
					<CheckIcon class="size-3.5 text-green-500" />
				{:else}
					<CopyIcon class="size-3.5" />
				{/if}
			</button>
		</div>
	{:else}
		<div class="ml-auto flex shrink-0 items-center gap-1">
			<Popover.Root bind:open={overflowOpen}>
				<Popover.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							title="More options"
							class="flex items-center rounded-md border border-border bg-background px-1.5 py-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground
								{overflowOpen ? 'border-ring ring-1 ring-ring' : ''}"
						>
							<ChevronDownIcon class="size-3.5" />
						</button>
					{/snippet}
				</Popover.Trigger>
				<Popover.Content class="w-auto p-1.5 shadow-lg" align="end">
					{#if presets}
						<TimeRangePicker
							{timeFilter}
							{presets}
							ontimechange={(f) => { timeFilter = f; onQuery(); }}
							fullWidth
						/>
					{/if}
					<div class="mt-1 flex items-center gap-1">
						{#if showZoom}
							<button
								onclick={() => { zoom(-1); overflowOpen = false; }}
								title="Zoom out"
								class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
							>
								<ZoomOutIcon class="size-3.5" />
							</button>
						{/if}
						{@render toolbarEnd?.()}
						<button
							onclick={() => { fullscreen = !fullscreen; overflowOpen = false; }}
							title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
							class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
						>
							{#if fullscreen}
								<MinimizeIcon class="size-3.5" />
							{:else}
								<MaximizeIcon class="size-3.5" />
							{/if}
						</button>
						<button
							onclick={() => { copyAsCsv(); overflowOpen = false; }}
							disabled={!queryData}
							title="Copy data as CSV"
							class="flex items-center rounded-md border border-border bg-background p-1 text-muted-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-40"
						>
							{#if copied}
								<CheckIcon class="size-3.5 text-green-500" />
							{:else}
								<CopyIcon class="size-3.5" />
							{/if}
						</button>
					</div>
					{@render compactDropdownExtra?.()}
				</Popover.Content>
			</Popover.Root>
		</div>
	{/if}
</div>
