<script lang="ts">
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { EventTarget, DateFilter, MirEvent } from '@mir/sdk';
	import type { EventsWidgetConfig } from '../api/dashboard-api';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import ChevronRightIcon from '@lucide/svelte/icons/chevron-right';
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';

	let {
		config,
		refreshTick = 0,
		onDevicesReady
	}: {
		config: EventsWidgetConfig;
		refreshTick?: number;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	let events = $state<MirEvent[]>([]);
	let isLoading = $state(false);
	let hasLoaded = $state(false);
	let error = $state<string | null>(null);
	let isInRefresh = false;

	async function loadEvents(from?: Date, to?: Date) {
		const mir = mirStore.mir;
		if (!mir) return;
		isLoading = true;
		error = null;
		try {
			const target = new EventTarget({
				names: config.target.names ?? [],
				namespaces: config.target.namespaces ?? [],
				limit: config.limit ?? 50,
				dateFilter: new DateFilter({ from, to })
			});
			events = await mir.client().listEvents().request(target);
			const seen = new Map<string, string>();
			for (const ev of events) {
				const name = ev.spec.relatedObject.meta.name;
				if (name && !seen.has(name)) seen.set(name, ev.spec.relatedObject.meta.namespace ?? '');
			}
			const deviceInfos = [...seen.entries()].map(([name], i) => ({
				id: name,
				name,
				color: CHART_COLORS[i % CHART_COLORS.length]
			}));
			onDevicesReady?.(deviceInfos);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load events';
		} finally {
			isLoading = false;
			hasLoaded = true;
			if (isInRefresh) {
				isInRefresh = false;
				dashboardStore.refreshDone();
			}
		}
	}

	$effect(() => {
		if (refreshTick > 0) {
			if (!isInRefresh) {
				isInRefresh = true;
				dashboardStore.refreshStart();
			}
			loadEvents();
		}
	});

	$effect(() => {
		if (mirStore.mir) {
			events = [];
			hasLoaded = false;
			loadEvents();
		} else {
			events = [];
			hasLoaded = false;
		}
	});

	const sorted = $derived(
		[...events].sort((a, b) => {
			const ta = a.status.lastAt?.getTime() ?? 0;
			const tb = b.status.lastAt?.getTime() ?? 0;
			return tb - ta;
		})
	);

	function relativeTime(date: Date | undefined): string {
		if (!date) return '—';
		const diffSec = Math.floor((Date.now() - date.getTime()) / 1000);
		if (diffSec < 60) return `${diffSec}s`;
		if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m`;
		if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h`;
		return `${Math.floor(diffSec / 86400)}d`;
	}

	let containerWidth = $state(0);
	const narrow = $derived(containerWidth < 420);

	let expandedRows = $state(new Set<string>());
	let highlightedPayloads = $state<Record<string, string>>({});

	function toggleRow(name: string) {
		const nowExpanded = !expandedRows.has(name);
		expandedRows = new Set(
			nowExpanded ? [...expandedRows, name] : [...expandedRows].filter((k) => k !== name)
		);
		if (nowExpanded) {
			const ev = sorted.find((e) => e.meta.name === name);
			if (ev?.spec.payload) highlightPayload(name, formatPayload(ev.spec.payload));
		}
	}

	function formatPayload(payload: unknown): string {
		if (payload == null) return '';
		try {
			return JSON.stringify(payload, null, 2);
		} catch {
			return String(payload);
		}
	}

	function escapeHtml(s: string): string {
		return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
	}

	function payloadHtml(key: string, plain: string): string {
		return highlightedPayloads[key]
			?? `<pre class="px-4 py-3 break-all whitespace-pre-wrap">${escapeHtml(plain)}</pre>`;
	}

	async function highlightPayload(key: string, code: string) {
		if (highlightedPayloads[key] !== undefined) return;
		const hl = await getHighlighter();
		highlightedPayloads[key] = hl.codeToHtml(code, {
			lang: 'json',
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
	}

	// colspan = caret + device + [type] + reason + time
	const colspan = $derived(narrow ? 4 : 5);
</script>

<div class="flex h-full flex-col" bind:clientWidth={containerWidth}>
	<div class="mt-2.5 shrink-0 border-b"></div>

	{#if isLoading && !hasLoaded}
		<div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
			Loading…
		</div>
	{:else if error}
		<div class="flex flex-1 items-center justify-center text-sm text-destructive">{error}</div>
	{:else if sorted.length === 0}
		<div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
			No events
		</div>
	{:else}
		<div class="min-h-0 flex-1 overflow-y-auto">
			<table class="w-full table-fixed text-xs">
				<colgroup>
					<col class="w-7" />
					<col class="w-[33%]" />
					{#if !narrow}<col class="w-[22%]" />{/if}
					<col />
					<col class="w-20" />
				</colgroup>
				<thead class="sticky top-0 bg-card">
					<tr class="border-b text-left text-muted-foreground">
						<th></th>
						<th class="px-2 py-1.5 font-medium">Device</th>
						{#if !narrow}<th class="px-2 py-1.5 font-medium">Type</th>{/if}
						<th class="px-2 py-1.5 font-medium">Reason</th>
						<th class="px-2 py-1.5 font-medium">Seen</th>
					</tr>
				</thead>
				<tbody>
					{#each sorted as ev (ev.meta.name)}
						{@const key = ev.meta.name}
						{@const isExpanded = expandedRows.has(key)}
						{@const payload = formatPayload(ev.spec.payload)}
						<tr
							class="cursor-pointer border-b border-border/50 hover:bg-muted/40"
							onclick={() => toggleRow(key)}
						>
							<td class="py-1.5 pl-2">
								<ChevronRightIcon
									class="h-3.5 w-3.5 text-muted-foreground transition-transform {isExpanded
										? 'rotate-90'
										: ''}"
								/>
							</td>
							<td class="overflow-hidden px-2 py-1.5">
								<span
									class="block truncate font-mono"
									title={ev.spec.relatedObject.meta.name || '—'}
								>
									{ev.spec.relatedObject.meta.name || '—'}
								</span>
							</td>
							{#if !narrow}
								<td class="py-1.5 pr-4 pl-2">
									<Badge
										variant={ev.spec.type === 'warning' ? 'destructive' : 'secondary'}
										class="font-mono text-[10px]"
									>
										{ev.spec.type}
									</Badge>
								</td>
							{/if}
							<td class="overflow-hidden px-2 py-1.5 text-muted-foreground">
								<span class="block truncate" title={ev.spec.reason || '—'}>
									{ev.spec.reason || '—'}
								</span>
							</td>
							<td class="px-2 py-1.5 text-muted-foreground">{#if ev.status.lastAt}<TimeTooltip timestamp={ev.status.lastAt} class="text-xs text-muted-foreground" />{:else}—{/if}</td>
						</tr>
						{#if isExpanded}
							<tr class="border-b border-border/50">
								<td {colspan} class="p-0">
									{#if payload}
										<div
											class="text-[11px] leading-relaxed [&_.shiki]:bg-transparent [&>pre]:px-4 [&>pre]:py-3 [&>pre]:break-all [&>pre]:whitespace-pre-wrap"
										>
											{@html payloadHtml(key, payload)}
										</div>
									{:else}
										<p class="px-4 py-3 text-xs text-muted-foreground">No payload.</p>
									{/if}
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
