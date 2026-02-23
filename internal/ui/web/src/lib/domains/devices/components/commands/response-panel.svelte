<script lang="ts">
	import type { ResponseEntry } from '$lib/domains/devices/types/types';
	import { SvelteMap } from 'svelte/reactivity';
	import { getHighlighter } from '$lib/shared/utils/highlighter';
	import XIcon from '@lucide/svelte/icons/x';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CheckIcon from '@lucide/svelte/icons/check';

	let {
		response,
		statusLabel,
		statusClass,
		onClear
	}: {
		response: Map<string, ResponseEntry>;
		statusLabel: (status: number) => string;
		statusClass: (status: number) => string;
		onClear: () => void;
	} = $props();

	let copiedId = $state<string | null>(null);
	let responseHtml = $state(new Map<string, string>());

	function decodePayload(payload: Uint8Array): string {
		if (!payload || payload.length === 0) return '';
		try {
			return JSON.stringify(JSON.parse(new TextDecoder().decode(payload)), null, 2);
		} catch {
			return new TextDecoder().decode(payload);
		}
	}

	async function handleCopy(devId: string, text: string) {
		try {
			await navigator.clipboard.writeText(text);
			copiedId = devId;
			setTimeout(() => (copiedId = null), 1500);
		} catch {
			// clipboard unavailable
		}
	}

	$effect(() => {
		const entries = [...response.entries()];
		getHighlighter().then((hl) => {
			const next = new SvelteMap<string, string>();
			for (const [devId, r] of entries) {
				const decoded = decodePayload(r.payload);
				if (decoded) {
					next.set(
						devId,
						hl.codeToHtml(decoded, {
							lang: 'json',
							themes: { light: 'github-light', dark: 'github-dark' },
							defaultColor: false
						})
					);
				}
			}
			responseHtml = next;
		});
	});
</script>

<div class="flex flex-1 flex-col overflow-hidden">
	<!-- Header with clear button -->
	<div class="flex items-center gap-2 border-b px-4 py-2">
		<span class="text-sm text-muted-foreground">Response</span>
		<button
			onclick={onClear}
			class="ml-auto rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
			title="Clear response"
		>
			<XIcon class="size-3.5" />
		</button>
	</div>

	<!-- Response entries -->
	<div class="flex-1 overflow-y-auto p-4">
		<div class="flex flex-col gap-3">
			{#each [...response.entries()] as [devId, resp] (devId)}
				<div class="rounded-lg border p-3">
					<div class="mb-2 flex items-center gap-2">
						<span class="font-mono text-xs text-muted-foreground">{devId}</span>
						<span
							class="rounded px-1.5 py-0.5 text-[10px] font-medium uppercase {statusClass(
								resp.status
							)}"
						>
							{statusLabel(resp.status)}
						</span>
					</div>
					{#if resp.error}
						<p class="rounded bg-destructive/10 px-2 py-1.5 text-xs text-destructive">
							{resp.error}
						</p>
					{/if}
					{#if resp.payload && resp.payload.length > 0}
						{@const decoded = decodePayload(resp.payload)}
						{#if decoded}
							<div class="relative mt-2 overflow-hidden rounded-md">
								{#if responseHtml.has(devId)}
									<!-- eslint-disable-next-line svelte/no-at-html-tags -->
									{@html responseHtml.get(devId)}
								{:else}
									<pre
										class="overflow-x-auto bg-muted px-3 py-2 font-mono text-xs leading-relaxed">{decoded}</pre>
								{/if}
								<button
									onclick={() => handleCopy(devId, decoded)}
									class="absolute top-1.5 right-1.5 rounded p-1 text-muted-foreground/60 transition-colors hover:text-foreground"
									title="Copy"
								>
									{#if copiedId === devId}
										<CheckIcon class="size-3 text-emerald-500" />
									{:else}
										<CopyIcon class="size-3" />
									{/if}
								</button>
							</div>
						{/if}
					{/if}
				</div>
			{/each}
		</div>
	</div>
</div>
