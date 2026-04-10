<script lang="ts">
	import { tick } from 'svelte';
	import * as Tooltip from '$lib/shared/components/shadcn/tooltip';
	import type { DeviceTargetConfig } from '../api/dashboard-api';

	let {
		devices = [],
		target
	}: {
		devices: { id: string; name: string; color: string }[];
		target: DeviceTargetConfig;
	} = $props();

	type Pill =
		| { kind: 'device'; id: string; name: string; color: string }
		| { kind: 'ns'; value: string }
		| { kind: 'label'; key: string; value: string };

	const allPills = $derived<Pill[]>([
		...(!target.ids?.length && !(target.namespaces ?? []).length && !Object.keys(target.labels ?? {}).length
			? [{ kind: 'ns' as const, value: 'all' }]
			: []),
		...(target.namespaces ?? []).map((v): Pill => ({ kind: 'ns', value: v })),
		...Object.entries(target.labels ?? {}).map(([k, v]): Pill => ({ kind: 'label', key: k, value: v })),
		...devices.map((d): Pill => ({ kind: 'device', id: d.id, name: d.name, color: d.color }))
	]);

	let containerEl = $state<HTMLDivElement | undefined>(undefined);
	let measureEl = $state<HTMLDivElement | undefined>(undefined);
	let visibleCount = $state(0);

	const visiblePills = $derived(allPills.slice(0, visibleCount));
	const overflowPills = $derived(allPills.slice(visibleCount));

	async function measure() {
		await tick();
		if (!containerEl || !measureEl) return;
		const containerWidth = containerEl.offsetWidth;
		if (containerWidth === 0) return;

		const pills = Array.from(measureEl.children) as HTMLElement[];
		if (pills.length === 0) return;

		const gap = 4; // gap-1 = 4px
		const overflowBtnWidth = 34;

		let usedWidth = 0;
		let count = 0;

		for (let i = 0; i < pills.length; i++) {
			const pillWidth = pills[i].offsetWidth;
			const withGap = count === 0 ? pillWidth : usedWidth + gap + pillWidth;
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
		allPills.length;
		measure();
	});

	$effect(() => {
		if (!containerEl) return;
		const ro = new ResizeObserver(() => measure());
		ro.observe(containerEl);
		return () => ro.disconnect();
	});

	function pillLabel(pill: Pill): string {
		if (pill.kind === 'device') return pill.name;
		if (pill.kind === 'ns') return `ns:${pill.value}`;
		return `${pill.key}=${pill.value}`;
	}
</script>

{#if allPills.length > 0}
	<div bind:this={containerEl} class="flex w-full items-center gap-1 overflow-hidden">
		{#each visiblePills as pill (pill.kind === 'device' ? pill.id : pillLabel(pill))}
			{#if pill.kind === 'device'}
				<span
					class="flex shrink-0 items-center gap-1 rounded-full border px-1.5 py-0.5 font-mono text-[10px] leading-none"
					style="border-color: color-mix(in oklch, {pill.color} 50%, transparent); color: {pill.color}"
				>
					<span class="size-1.5 shrink-0 rounded-full" style="background: {pill.color}"></span>
					{pill.name}
				</span>
			{:else}
				<span
					class="shrink-0 rounded-full border border-border/60 bg-muted/20 px-1.5 py-0.5 font-mono text-[10px] leading-none text-muted-foreground"
				>
					{pillLabel(pill)}
				</span>
			{/if}
		{/each}

		{#if overflowPills.length > 0}
			<Tooltip.Root>
				<Tooltip.Trigger
					class="shrink-0 rounded-full border border-border/60 bg-muted/20 px-1.5 py-0.5 font-mono text-[10px] leading-none text-muted-foreground"
				>
					+{overflowPills.length}
				</Tooltip.Trigger>
				<Tooltip.Content class="p-1.5">
					<div class="flex flex-col gap-0.5">
						{#each overflowPills as pill (pill.kind === 'device' ? pill.id : pillLabel(pill))}
							{#if pill.kind === 'device'}
								<span class="flex items-center gap-1.5 px-1 py-0.5 font-mono text-xs">
									<span
										class="size-2 shrink-0 rounded-full"
										style="background: {pill.color}"
									></span>
									{pill.name}
								</span>
							{:else}
								<span class="px-1 py-0.5 font-mono text-xs text-muted-foreground">
									{pillLabel(pill)}
								</span>
							{/if}
						{/each}
					</div>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}
	</div>
{/if}

<!-- Off-screen measurement: renders all pills to get their natural widths -->
<div
	bind:this={measureEl}
	class="pointer-events-none invisible absolute flex gap-1"
	style="top:-9999px;left:-9999px"
	aria-hidden="true"
>
	{#each allPills as pill (pill.kind === 'device' ? pill.id : pillLabel(pill))}
		{#if pill.kind === 'device'}
			<span class="flex shrink-0 items-center gap-1 rounded-full border px-1.5 py-0.5 font-mono text-[10px] leading-none">
				<span class="size-1.5 shrink-0 rounded-full"></span>
				{pill.name}
			</span>
		{:else}
			<span class="shrink-0 rounded-full border px-1.5 py-0.5 font-mono text-[10px] leading-none">
				{pillLabel(pill)}
			</span>
		{/if}
	{/each}
</div>
