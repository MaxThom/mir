<script lang="ts">
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';

	let {
		fields,
		selectedFields,
		ontoggle
	}: {
		fields: string[];
		selectedFields: string[];
		ontoggle: (field: string, shift: boolean) => void;
	} = $props();
</script>

<div class="flex shrink-0 flex-wrap items-center gap-2 border-b px-4 py-1.75">
	<div class="flex flex-wrap gap-1">
		{#each fields as field, i (field)}
			<button
				onclick={(e) => ontoggle(field, e.shiftKey || e.ctrlKey)}
				class="flex items-center gap-1 rounded-sm border px-1.5 py-0.5 font-mono text-[11px] transition-colors
					{selectedFields.includes(field)
						? 'border-transparent text-white'
						: 'border-border/60 bg-muted/40 text-muted-foreground hover:bg-accent'}"
				style={selectedFields.includes(field)
					? `background: ${CHART_COLORS[i % CHART_COLORS.length]};`
					: ''}
			>
				{field}
			</button>
		{/each}
	</div>
</div>
