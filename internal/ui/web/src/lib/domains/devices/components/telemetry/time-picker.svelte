<style>
	.hide-scrollbar {
		scrollbar-width: none;
	}
	.hide-scrollbar::-webkit-scrollbar {
		display: none;
	}
</style>

<script lang="ts">
	import ClockIcon from '@lucide/svelte/icons/clock';

	let {
		value = $bindable('00:00'),
		onchange
	}: {
		value?: string;
		onchange?: () => void;
	} = $props();

	let open = $state(false);
	let hourListEl = $state<HTMLDivElement | null>(null);
	let minuteListEl = $state<HTMLDivElement | null>(null);

	const HOURS = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'));
	const MINUTES = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'));

	let selectedHour = $derived(value?.split(':')[0] ?? '00');
	let selectedMinute = $derived(value?.split(':')[1] ?? '00');

	function handleInput() {
		if (/^\d{2}:\d{2}$/.test(value ?? '')) {
			onchange?.();
		}
	}

	function setHour(h: string) {
		value = `${h}:${selectedMinute}`;
		onchange?.();
	}

	function setMinute(m: string) {
		value = `${selectedHour}:${m}`;
		onchange?.();
	}

	$effect(() => {
		if (open) {
			setTimeout(() => {
				hourListEl?.querySelector('[data-selected="true"]')?.scrollIntoView({ block: 'center', behavior: 'instant' });
				minuteListEl?.querySelector('[data-selected="true"]')?.scrollIntoView({ block: 'center', behavior: 'instant' });
			}, 0);
		}
	});
</script>

{#if open}
	<div
		class="fixed inset-0 z-10"
		role="presentation"
		onclick={() => (open = false)}
		onkeydown={() => {}}
	></div>
{/if}

<div class="relative w-24">
	<!-- Trigger: writable input + clock icon toggle -->
	<div
		class="flex h-8 items-center rounded-md border border-input bg-background shadow-sm transition-colors focus-within:border-ring focus-within:ring-1 focus-within:ring-ring"
	>
		<input
			type="text"
			bind:value
			oninput={handleInput}
			placeholder="00:00"
			maxlength={5}
			class="h-full min-w-0 flex-1 bg-transparent pl-2.5 font-mono text-xs text-foreground focus:outline-none"
		/>
		<button
			onclick={() => (open = !open)}
			tabindex="-1"
			class="flex h-full shrink-0 items-center px-2 text-muted-foreground transition-colors hover:text-foreground"
		>
			<ClockIcon class="size-3" />
		</button>
	</div>

	{#if open}
		<div
			class="absolute right-0 top-full z-20 mt-1 flex overflow-hidden rounded-md border border-border bg-popover shadow-lg"
		>
			<!-- Hours column -->
			<div bind:this={hourListEl} class="hide-scrollbar flex h-44 w-14 flex-col overflow-y-auto py-1">
				{#each HOURS as h (h)}
					<button
						data-selected={h === selectedHour}
						onclick={() => setHour(h)}
						class="flex-none py-1 text-center font-mono text-xs transition-colors {h === selectedHour
							? 'bg-primary font-semibold text-primary-foreground'
							: 'text-foreground hover:bg-accent hover:text-accent-foreground'}"
					>
						{h}
					</button>
				{/each}
			</div>

			<!-- Minutes column -->
			<div bind:this={minuteListEl} class="hide-scrollbar flex h-44 w-14 flex-col overflow-y-auto py-1">
				{#each MINUTES as m (m)}
					<button
						data-selected={m === selectedMinute}
						onclick={() => setMinute(m)}
						class="flex-none py-1 text-center font-mono text-xs transition-colors {m === selectedMinute
							? 'bg-primary font-semibold text-primary-foreground'
							: 'text-foreground hover:bg-accent hover:text-accent-foreground'}"
					>
						{m}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
