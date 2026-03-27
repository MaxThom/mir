<script lang="ts">
	import { untrack } from 'svelte';
	import type { Device } from '@mir/sdk';
	import type { DeviceTargetConfig } from '../api/dashboard-api';
	import DevicePicker from './device-picker.svelte';
	import * as Table from '$lib/shared/components/shadcn/table';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { ButtonGroup } from '$lib/shared/components/shadcn/button-group';
	import { SuggestionInput } from '$lib/shared/components/ui/suggestion-input';
	import { cn } from '$lib/utils';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';

	let {
		devices,
		isLoading = false,
		target = $bindable<DeviceTargetConfig>({}),
		initialTarget
	}: {
		devices: Device[];
		isLoading?: boolean;
		target: DeviceTargetConfig;
		initialTarget?: DeviceTargetConfig;
	} = $props();

	let mode = $state<'dynamic' | 'specific'>(
		untrack(() =>
			initialTarget?.namespaces?.length || initialTarget?.labels
				? 'dynamic'
				: 'specific'
		)
	);

	// Dynamic state
	let selectedNamespaces = $state<string[]>(untrack(() => initialTarget?.namespaces ?? []));
	let labelConditions = $state<{ key: string; value: string }[]>(
		untrack(() => Object.entries(initialTarget?.labels ?? {}).map(([key, value]) => ({ key, value })))
	);

	// Specific state
	let selectedIds = $state<string[]>(untrack(() => initialTarget?.ids ?? []));

	// ─── Derived suggestions ──────────────────────────────────────────────────

	const allNamespaces = $derived(
		[...new Set(devices.map((d) => d.meta?.namespace ?? 'default'))].sort()
	);
	const allLabelKeys = $derived(
		[...new Set(devices.flatMap((d) => Object.keys(d.meta?.labels ?? {})))].sort()
	);

	function valuesForKey(key: string): string[] {
		return [
			...new Set(
				devices.flatMap((d) => {
					const v = d.meta?.labels?.[key];
					return v !== undefined ? [v] : [];
				})
			)
		].sort();
	}

	// Live preview
	const previewDevices = $derived(
		devices.filter((d) => {
			const activeNs = selectedNamespaces.filter((ns) => ns);
			const nsMatch = activeNs.length === 0 || activeNs.includes(d.meta?.namespace ?? 'default');
			const valid = labelConditions.filter((c) => c.key && c.value);
			const labelMatch = valid.every(({ key, value }) => d.meta?.labels?.[key] === value);
			return nsMatch && labelMatch;
		})
	);

	const previewOnline = $derived(previewDevices.filter((d) => d.status?.online).length);

	const hasActiveFilters = $derived(
		selectedNamespaces.some((ns) => ns) || labelConditions.some((c) => c.key && c.value)
	);

	// ─── Sync to bindable target ──────────────────────────────────────────────

	$effect(() => {
		if (mode === 'dynamic') {
			const activeNs = selectedNamespaces.filter((ns) => ns);
			const valid = labelConditions.filter((c) => c.key && c.value);
			target = {
				...(activeNs.length ? { namespaces: activeNs } : {}),
				...(valid.length ? { labels: Object.fromEntries(valid.map((c) => [c.key, c.value])) } : {})
			};
		} else {
			target = { ids: selectedIds };
		}
	});

	// ─── Namespace helpers ────────────────────────────────────────────────────

	function addNamespace() {
		selectedNamespaces = [...selectedNamespaces, ''];
	}
	function removeNamespace(i: number) {
		selectedNamespaces = selectedNamespaces.filter((_, idx) => idx !== i);
	}

	// ─── Label helpers ────────────────────────────────────────────────────────

	function addCondition() {
		labelConditions = [...labelConditions, { key: '', value: '' }];
	}
	function removeCondition(i: number) {
		labelConditions = labelConditions.filter((_, idx) => idx !== i);
	}
	function setKey(i: number, key: string) {
		labelConditions = labelConditions.map((c, idx) => (idx === i ? { key, value: '' } : c));
	}
	function setValue(i: number, value: string) {
		labelConditions = labelConditions.map((c, idx) => (idx === i ? { ...c, value } : c));
	}
</script>

<!-- ─── Mode toggle ─────────────────────────────────────────────────────── -->
<ButtonGroup>
	{#each [['specific', 'Specific'], ['dynamic', 'Dynamic']] as [val, label] (val)}
		<Button
			variant={mode === val ? 'secondary' : 'outline'}
			size="sm"
			onclick={() => (mode = val as 'dynamic' | 'specific')}
		>
			{label}
		</Button>
	{/each}
</ButtonGroup>

<!-- ─── Dynamic mode ──────────────────────────────────────────────────────── -->
{#if mode === 'dynamic'}
	<div class="rounded-md border border-border">
		<!-- Namespaces -->
		<div class="space-y-2 border-b p-3">
			<div class="flex items-center justify-between">
				<span class="text-xs font-semibold tracking-wide text-muted-foreground uppercase"
					>Namespaces</span
				>
				<span class="text-xs text-muted-foreground">match any (OR)</span>
			</div>

			{#if selectedNamespaces.length > 0}
				<div class="space-y-1.5">
					{#each selectedNamespaces as ns, i (i)}
						<div class="flex items-center gap-1.5">
							<SuggestionInput
								bind:value={selectedNamespaces[i]}
								suggestions={allNamespaces}
								placeholder="namespace"
								class="h-7 flex-1 font-mono text-xs"
							/>
							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => removeNamespace(i)}
								aria-label="Remove namespace"
								class="size-7 text-muted-foreground hover:text-destructive"
							>
								<XIcon class="size-3.5" />
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<Button
				variant="ghost"
				size="sm"
				onclick={addNamespace}
				class="h-7 gap-1 px-2 text-xs text-muted-foreground"
			>
				<PlusIcon class="size-3" />
				Add namespace
			</Button>
		</div>

		<!-- Labels -->
		<div class="space-y-2 p-3">
			<div class="flex items-center justify-between">
				<span class="text-xs font-semibold tracking-wide text-muted-foreground uppercase"
					>Labels</span
				>
				<span class="text-xs text-muted-foreground">match all (AND)</span>
			</div>

			{#if labelConditions.length > 0}
				<div class="space-y-1.5">
					{#each labelConditions as condition, i (i)}
						<div class="flex items-center gap-1.5">
							<!-- Key input -->
							<SuggestionInput
								value={condition.key}
								suggestions={allLabelKeys}
								placeholder="key"
								class="h-7 w-28 font-mono text-xs"
								onchange={(v) => setKey(i, v)}
							/>

							<span class="text-xs text-muted-foreground">=</span>

							<!-- Value input -->
							<SuggestionInput
								value={condition.value}
								suggestions={valuesForKey(condition.key)}
								placeholder="value"
								disabled={!condition.key}
								class="h-7 w-28 font-mono text-xs"
								onchange={(v) => setValue(i, v)}
							/>

							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => removeCondition(i)}
								aria-label="Remove condition"
								class="size-7 text-muted-foreground hover:text-destructive"
							>
								<XIcon class="size-3.5" />
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<Button
				variant="ghost"
				size="sm"
				onclick={addCondition}
				class="h-7 gap-1 px-2 text-xs text-muted-foreground"
			>
				<PlusIcon class="size-3" />
				Add label condition
			</Button>
		</div>
	</div>

	<!-- Preview table -->
	<div class="rounded-md border border-border">
		<!-- Header bar -->
		<div class="flex items-center justify-between border-b px-4 py-2">
			<div class="flex items-center gap-2">
				<span class="text-sm font-semibold"> Device Preview </span>
				<Badge variant="secondary" class="tabular-nums">{previewDevices.length}</Badge>
			</div>
			<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
				<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
				{previewOnline} online
			</div>
		</div>

		<div class="max-h-44 overflow-y-auto">
			<Table.Root>
				<Table.Header>
					<Table.Row class="hover:bg-transparent">
						<Table.Head class="w-5"></Table.Head>
						<Table.Head
							class="h-8 text-xs font-medium tracking-wide text-muted-foreground uppercase"
							>Name</Table.Head
						>
						<Table.Head
							class="h-8 text-xs font-medium tracking-wide text-muted-foreground uppercase"
							>Namespace</Table.Head
						>
						<Table.Head
							class="h-8 text-xs font-medium tracking-wide text-muted-foreground uppercase"
							>Device ID</Table.Head
						>
						<Table.Head
							class="h-8 text-xs font-medium tracking-wide text-muted-foreground uppercase"
							>Labels</Table.Head
						>
						<Table.Head
							class="h-8 text-xs font-medium tracking-wide text-muted-foreground uppercase"
							>Schema</Table.Head
						>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#if previewDevices.length === 0}
						<Table.Row class="hover:bg-transparent">
							<Table.Cell colspan={6} class="py-8 text-center">
								<div class="flex flex-col items-center gap-2 text-muted-foreground">
									<SatelliteDishIcon class="h-6 w-6 opacity-40" />
									<span class="text-sm">No devices match</span>
								</div>
							</Table.Cell>
						</Table.Row>
					{:else}
						{#each previewDevices as d (d.spec?.deviceId ?? d.meta?.name)}
							<Table.Row class="hover:bg-transparent">
								<Table.Cell class="w-5 pr-0">
									<span
										class={cn(
											'block h-2 w-2 rounded-full',
											d.status?.online
												? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
												: 'bg-muted-foreground/30'
										)}
									></span>
								</Table.Cell>
								<Table.Cell class="text-sm font-medium">{d.meta?.name ?? '—'}</Table.Cell>
								<Table.Cell>
									<Badge variant="outline" class="font-mono text-xs font-normal">
										{d.meta?.namespace ?? 'default'}
									</Badge>
								</Table.Cell>
								<Table.Cell class="font-mono text-xs text-muted-foreground">
									{d.spec?.deviceId ?? '—'}
								</Table.Cell>
								<Table.Cell>
									<div class="flex flex-wrap gap-1">
										{#each Object.entries(d.meta?.labels ?? {}) as [k, v] (k)}
											<Badge variant="secondary" class="font-mono text-xs font-normal"
												>{k}={v}</Badge
											>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/each}
									</div>
								</Table.Cell>
								<Table.Cell>
									<div class="flex flex-wrap gap-1">
										{#each d.status?.schema?.packageNames ?? [] as pkg (pkg)}
											<Badge variant="outline" class="font-mono text-xs font-normal">{pkg}</Badge>
										{:else}
											<span class="text-muted-foreground">—</span>
										{/each}
									</div>
								</Table.Cell>
							</Table.Row>
						{/each}
					{/if}
				</Table.Body>
			</Table.Root>
		</div>
	</div>

	<!-- ─── Specific mode ──────────────────────────────────────────────────────── -->
{:else}
	<DevicePicker {devices} bind:selectedIds {isLoading} />
{/if}
