<script lang="ts">
	import { Device } from '@mir/sdk';
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { JsonValue } from '$lib/shared/components/ui/json-value';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import CircleCheckBigIcon from '@lucide/svelte/icons/circle-check-big';

	let { device }: { device: Device } = $props();

	let desiredProps = $derived(
		Object.entries(device?.properties?.desired ?? {}).sort(([a], [b]) => a.localeCompare(b))
	);
	let reportedProps = $derived(
		Object.entries(device?.properties?.reported ?? {}).sort(([a], [b]) => a.localeCompare(b))
	);

	function isMatchingDesired(key: string, reportedVal: unknown): boolean {
		const desired = (device?.properties?.desired ?? {}) as Record<string, unknown>;
		if (!(key in desired)) return false;
		return JSON.stringify(desired[key]) === JSON.stringify(reportedVal);
	}
</script>

<Card.Root class="gap-0 py-4">
	<Card.Content class="px-6 py-2">
		<div class="max-h-96 overflow-auto">
			{#if desiredProps.length === 0 && reportedProps.length === 0}
				<p class="text-sm text-muted-foreground">No properties configured.</p>
			{:else}
				<div class="space-y-3">
					<!-- Desired -->
					<div>
						<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Desired
						</p>
						{#if desiredProps.length === 0}
							<p class="text-xs text-muted-foreground">—</p>
						{:else}
							<div class="space-y-1.5">
								{#each desiredProps as [k, v] (k)}
									<div class="flex flex-col">
										<div class="flex items-center gap-1.5">
											<span class="font-mono text-xs text-muted-foreground">{k}</span>
											{#if device?.status?.properties?.desired?.[k]}
												<TimeTooltip
													timestamp={device.status.properties.desired[k]}
													utc={editorPrefs.utc}
													class="text-[10px] text-muted-foreground/60"
												/>
											{/if}
										</div>
										<JsonValue value={v} />
									</div>
								{/each}
							</div>
						{/if}
					</div>

					<Separator />

					<!-- Reported -->
					<div>
						<p class="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
							Reported
						</p>
						{#if reportedProps.length === 0}
							<p class="text-xs text-muted-foreground">—</p>
						{:else}
							<div class="space-y-1.5">
								{#each reportedProps as [k, v] (k)}
									<div class="flex flex-col">
										<div class="flex items-center gap-1.5">
											<span class="font-mono text-xs text-muted-foreground">{k}</span>
											{#if isMatchingDesired(k, v)}
												<CircleCheckBigIcon class="size-3 text-emerald-500" />
											{/if}
											{#if device?.status?.properties?.reported?.[k]}
												<TimeTooltip
													timestamp={device.status.properties.reported[k]}
													utc={editorPrefs.utc}
													class="text-[10px] text-muted-foreground/60"
												/>
											{/if}
										</div>
										<JsonValue value={v} />
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</Card.Content>
</Card.Root>
