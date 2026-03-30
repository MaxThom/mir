<script lang="ts">
	import { tick } from 'svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import * as Popover from '$lib/shared/components/shadcn/popover';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';

	let {
		fields,
		selectedFields,
		ontoggle,
		class: wrapperClass = 'shrink-0 px-4 py-1.5'
	}: {
		fields: string[];
		selectedFields: string[];
		ontoggle: (field: string, shift: boolean) => void;
		class?: string;
	} = $props();

	let containerEl: HTMLDivElement | undefined;
	let measureEl: HTMLDivElement | undefined;
	let visibleCount = $state(0);
	let overflowOpen = $state(false);

	const visibleFields = $derived(fields.slice(0, visibleCount));
	const overflowFields = $derived(fields.slice(visibleCount));

	async function measure() {
		await tick();
		if (!containerEl || !measureEl) return;
		const containerWidth = containerEl.offsetWidth;
		if (containerWidth === 0) return;

		const pills = Array.from(measureEl.children) as HTMLElement[];
		if (pills.length === 0) return;

		const gap = 4; // gap-1 = 4px
		const overflowBtnWidth = 34; // approx +N button width

		let usedWidth = 0;
		let count = 0;

		for (let i = 0; i < pills.length; i++) {
			const pillWidth = pills[i].offsetWidth;
			const withGap = count === 0 ? pillWidth : usedWidth + gap + pillWidth;
			// If this isn't the last pill, reserve space for overflow button
			const needsOverflowBtn = i < pills.length - 1;
			const available = needsOverflowBtn ? containerWidth - overflowBtnWidth - gap : containerWidth;

			if (withGap <= available) {
				usedWidth = withGap;
				count++;
			} else {
				break;
			}
		}

		visibleCount = Math.max(1, count);
	}

	$effect(() => {
		// Re-measure when fields list changes
		fields.length;
		measure();
	});

	$effect(() => {
		if (!containerEl) return;
		const ro = new ResizeObserver(() => measure());
		ro.observe(containerEl);
		return () => ro.disconnect();
	});
</script>

<div class="flex {wrapperClass}">
	<div bind:this={containerEl} class="flex w-full items-center gap-1 overflow-hidden">
		{#each visibleFields as field (field)}
			{@const idx = fields.indexOf(field)}
			<button
				onclick={(e) => ontoggle(field, e.shiftKey || e.ctrlKey)}
				class="shrink-0 flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
					{selectedFields.includes(field)
						? 'border-transparent text-white'
						: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
				style={selectedFields.includes(field)
					? `background: ${CHART_COLORS[idx % CHART_COLORS.length]};`
					: ''}
			>
				{field}
			</button>
		{/each}

		{#if overflowFields.length > 0}
			<Popover.Root bind:open={overflowOpen}>
				<Popover.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							class="shrink-0 flex items-center rounded-sm border border-border/60 bg-muted/40 px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground transition-colors hover:bg-accent
								{overflowOpen ? 'border-ring ring-1 ring-ring' : ''}"
						>
							+{overflowFields.length}
						</button>
					{/snippet}
				</Popover.Trigger>
				<Popover.Content class="w-auto p-1.5 shadow-lg" align="start">
					<div class="flex flex-col gap-0.5">
						{#each overflowFields as field (field)}
							{@const idx = fields.indexOf(field)}
							{@const selected = selectedFields.includes(field)}
							<button
								onclick={(e) => ontoggle(field, e.shiftKey || e.ctrlKey)}
								class="flex items-center gap-2 rounded-sm px-2 py-1 text-xs transition-colors
									{selected
										? 'bg-accent text-accent-foreground font-medium'
										: 'text-foreground hover:bg-accent/60'}"
							>
								<span
									class="inline-block size-2 shrink-0 rounded-full"
									style="background: {CHART_COLORS[idx % CHART_COLORS.length]}"
								></span>
								{field}
							</button>
						{/each}
					</div>
				</Popover.Content>
			</Popover.Root>
		{/if}
	</div>
</div>

<!-- Off-screen measurement: renders all pills to get their natural widths -->
<div
	bind:this={measureEl}
	class="pointer-events-none invisible absolute flex gap-1"
	style="top:-9999px;left:-9999px"
	aria-hidden="true"
>
	{#each fields as field (field)}
		<button class="shrink-0 flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px]">
			{field}
		</button>
	{/each}
</div>
