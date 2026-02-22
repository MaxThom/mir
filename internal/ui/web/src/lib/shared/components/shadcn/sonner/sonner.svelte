<script lang="ts">
	import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
	import InfoIcon from "@lucide/svelte/icons/info";
	import Loader2Icon from "@lucide/svelte/icons/loader-2";
	import OctagonXIcon from "@lucide/svelte/icons/octagon-x";
	import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";

	import { Toaster as Sonner, type ToasterProps as SonnerProps } from "svelte-sonner";
	import { mode } from "mode-watcher";

	let { ...restProps }: SonnerProps = $props();
</script>

<Sonner
	theme={mode.current}
	class="toaster group"
	style="
		--width: 420px;
		--font-size: 15px;
		--normal-bg: var(--color-popover);
		--normal-text: var(--color-popover-foreground);
		--normal-border: var(--color-border);
	"
	{...restProps}
	>{#snippet loadingIcon()}
		<Loader2Icon class="size-5 animate-spin" />
	{/snippet}
	{#snippet successIcon()}
		<CircleCheckIcon class="size-5 text-[oklch(0.723_0.191_149.6)]" />
	{/snippet}
	{#snippet errorIcon()}
		<OctagonXIcon class="size-5 text-destructive" />
	{/snippet}
	{#snippet infoIcon()}
		<InfoIcon class="size-5" />
	{/snippet}
	{#snippet warningIcon()}
		<TriangleAlertIcon class="size-5" />
	{/snippet}
</Sonner>
