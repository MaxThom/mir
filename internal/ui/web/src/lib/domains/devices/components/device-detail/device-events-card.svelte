<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { eventStore } from '$lib/domains/events/stores/event.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';

	let expandedEvents = new SvelteSet<number>();
	let highlightedPayloads = $state<Record<number, string>>({});
	let copiedIndex = $state<number | null>(null);

	function toggleEvent(i: number) {
		if (expandedEvents.has(i)) {
			expandedEvents.delete(i);
		} else {
			expandedEvents.add(i);
			const payload = formatPayload(eventStore.events[i]?.spec?.payload);
			if (payload) highlightPayload(i, payload);
		}
	}

	function formatPayload(payload: unknown): string {
		if (payload === undefined || payload === null) return '';
		try {
			return JSON.stringify(payload, null, 2);
		} catch {
			return String(payload);
		}
	}

	async function highlightPayload(i: number, payload: string) {
		if (highlightedPayloads[i] !== undefined) return;
		const hl = await getHighlighter();
		highlightedPayloads[i] = hl.codeToHtml(payload, {
			lang: 'json',
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
	}

	function copyPayload(i: number, payload: string) {
		navigator.clipboard.writeText(payload).then(() => {
			copiedIndex = i;
			setTimeout(() => (copiedIndex = null), 2000);
		});
	}
</script>

<Card.Root class="min-w-0 gap-0 py-4">
	<Card.Content class="px-6 py-2">
		<!-- Toolbar -->
		<div class="mb-3 flex items-center gap-2">
			<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Events</p>
			<Badge variant="secondary" class="tabular-nums">{eventStore.events.length}</Badge>
		</div>

		<!-- Body -->
		{#if eventStore.isLoading && eventStore.events.length === 0}
			<div class="flex items-center gap-2 text-sm text-muted-foreground">
				<Spinner class="size-3.5" /> Loading events…
			</div>
		{:else if eventStore.error}
			<p class="text-xs text-destructive">{eventStore.error}</p>
		{:else if eventStore.events.length === 0}
			<p class="text-sm text-muted-foreground">No events.</p>
		{:else}
			<div class="max-h-72 w-full overflow-y-auto">
				{#each eventStore.events as event, i (i)}
					{@const expanded = expandedEvents.has(i)}
					{@const payload = formatPayload(event.spec?.payload)}
					<div class="border-b border-border/40 last:border-0">
						<!-- Summary row -->
						<button
							onclick={() => toggleEvent(i)}
							class="flex w-full items-center gap-2 py-1.5 text-left"
						>
							<ChevronDownIcon
								class="size-3 shrink-0 text-muted-foreground transition-transform {expanded
									? ''
									: '-rotate-90'}"
							/>
							<Badge variant="outline" class="shrink-0 font-mono text-[10px]">
								{event.spec?.type ?? '—'}
							</Badge>
							<span class="min-w-0 flex-1 truncate text-xs">
								{event.spec?.message || event.spec?.reason || '—'}
							</span>
							{#if (event.status?.count ?? 0) > 1}
								<Badge variant="secondary" class="shrink-0 text-[10px] tabular-nums">
									×{event.status!.count}
								</Badge>
							{/if}
							{#if event.status?.lastAt}
								<!-- Stop click propagation so the tooltip trigger doesn't toggle the row -->
								<span
									role="none"
									onclick={(e) => e.stopPropagation()}
									onkeydown={(e) => e.stopPropagation()}
									class="shrink-0"
								>
									<TimeTooltip
										timestamp={event.status.lastAt}
										utc={editorPrefs.utc}
										class="text-[10px] text-muted-foreground"
									/>
								</span>
							{/if}
						</button>

						<!-- Expanded payload -->
						{#if expanded}
							<div class="min-w-0 pb-2 pl-5">
								{#if event.spec?.reason}
									<p class="mb-1 text-[10px] text-muted-foreground">
										<span class="font-medium">Reason:</span>
										{event.spec.reason}
									</p>
								{/if}
								{#if payload}
									<div
										class="group relative overflow-hidden rounded border border-border text-[10px] leading-relaxed [&>pre]:px-3 [&>pre]:py-2 [&>pre]:break-all [&>pre]:whitespace-pre-wrap"
									>
										<button
											onclick={() => copyPayload(i, payload)}
											aria-label="Copy"
											class="absolute top-1.5 right-1.5 z-10 rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-foreground"
										>
											{#if copiedIndex === i}
												<CheckIcon class="size-3 text-emerald-500" />
											{:else}
												<CopyIcon class="size-3" />
											{/if}
										</button>
										{#if highlightedPayloads[i]}
											<!-- eslint-disable-next-line svelte/no-at-html-tags -->
											{@html highlightedPayloads[i]}
										{:else}
											<pre
												class="bg-muted px-3 py-2 font-mono text-[10px] break-all whitespace-pre-wrap">{payload}</pre>
										{/if}
									</div>
								{:else}
									<p class="text-[10px] text-muted-foreground">No payload.</p>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</Card.Content>
</Card.Root>
