<script lang="ts">
	import type { Device } from '@mir/sdk';
	import type { DeviceTargetConfig } from '../api/dashboard-api';
	import DevicePicker from './device-picker.svelte';
	import * as Table from '$lib/shared/components/shadcn/table';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { cn } from '$lib/utils';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import SatelliteDishIcon from '@lucide/svelte/icons/satellite-dish';

	let {
		devices,
		isLoading = false,
		target = $bindable<DeviceTargetConfig>({})
	}: {
		devices: Device[];
		isLoading?: boolean;
		target: DeviceTargetConfig;
	} = $props();

	let mode = $state<'dynamic' | 'specific'>('dynamic');

	// Dynamic state
	let selectedNamespaces = $state<string[]>([]);
	let labelConditions = $state<{ key: string; value: string }[]>([]);
	let nsInput = $state('');

	// Specific state
	let selectedIds = $state<string[]>([]);

	// ─── Derived suggestions ──────────────────────────────────────────────────

	const allNamespaces = $derived(
		[...new Set(devices.map((d) => d.meta?.namespace ?? 'default'))].sort()
	);
	const allLabelKeys = $derived(
		[...new Set(devices.flatMap((d) => Object.keys(d.meta?.labels ?? {})))].sort()
	);

	function valuesForKey(key: string): string[] {
		return [...new Set(
			devices.flatMap((d) => {
				const v = d.meta?.labels?.[key];
				return v !== undefined ? [v] : [];
			})
		)].sort();
	}

	// Live preview
	const previewDevices = $derived(
		devices.filter((d) => {
			const nsMatch =
				selectedNamespaces.length === 0 ||
				selectedNamespaces.includes(d.meta?.namespace ?? 'default');
			const valid = labelConditions.filter((c) => c.key && c.value);
			const labelMatch = valid.every(({ key, value }) => d.meta?.labels?.[key] === value);
			return nsMatch && labelMatch;
		})
	);

	const previewOnline = $derived(previewDevices.filter((d) => d.status?.online).length);

	const hasActiveFilters = $derived(
		selectedNamespaces.length > 0 ||
		labelConditions.some((c) => c.key && c.value)
	);

	// ─── Sync to bindable target ──────────────────────────────────────────────

	$effect(() => {
		if (mode === 'dynamic') {
			const valid = labelConditions.filter((c) => c.key && c.value);
			target = {
				...(selectedNamespaces.length ? { namespaces: selectedNamespaces } : {}),
				...(valid.length
					? { labels: Object.fromEntries(valid.map((c) => [c.key, c.value])) }
					: {})
			};
		} else {
			target = { ids: selectedIds };
		}
	});

	// ─── Namespace helpers ────────────────────────────────────────────────────

	function commitNamespace() {
		const ns = nsInput.trim();
		if (ns && !selectedNamespaces.includes(ns)) {
			selectedNamespaces = [...selectedNamespaces, ns];
		}
		nsInput = '';
	}

	function removeNamespace(ns: string) {
		selectedNamespaces = selectedNamespaces.filter((n) => n !== ns);
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
<div class="flex w-fit rounded-md border border-input bg-muted/50 p-0.5">
	{#each [['dynamic', 'Dynamic'], ['specific', 'Specific']] as [val, label] (val)}
		<button
			onclick={() => (mode = val as 'dynamic' | 'specific')}
			class={cn(
				'rounded px-3 py-1 text-sm transition-colors',
				mode === val
					? 'bg-background text-foreground shadow-sm'
					: 'text-muted-foreground hover:text-foreground'
			)}
		>
			{label}
		</button>
	{/each}
</div>

<!-- ─── Dynamic mode ──────────────────────────────────────────────────────── -->
{#if mode === 'dynamic'}
	<div class="rounded-md border border-border">

		<!-- Namespaces -->
		<div class="space-y-2 border-b p-3">
			<div class="flex items-center justify-between">
				<span class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Namespaces</span>
				<span class="text-xs text-muted-foreground">match any</span>
			</div>

			{#if selectedNamespaces.length > 0}
				<div class="flex flex-wrap gap-1.5">
					{#each selectedNamespaces as ns (ns)}
						<Badge variant="secondary" class="gap-1 pr-1 font-mono text-xs font-normal">
							{ns}
							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => removeNamespace(ns)}
								aria-label="Remove {ns}"
								class="ml-0.5 size-4 rounded opacity-60 hover:opacity-100 hover:text-destructive hover:bg-transparent"
							>
								<XIcon class="size-3" />
							</Button>
						</Badge>
					{/each}
				</div>
			{/if}

			<!-- Input to add namespace -->
			<div class="flex gap-1.5">
				<!-- svelte-ignore a11y_autocomplete_valid -->
				<Input
					list="ns-suggestions"
					bind:value={nsInput}
					placeholder={selectedNamespaces.length === 0 ? 'All namespaces — type to filter' : 'Add namespace…'}
					class="h-7 flex-1 font-mono text-xs"
					onkeydown={(e) => e.key === 'Enter' && commitNamespace()}
				/>
				<datalist id="ns-suggestions">
					{#each allNamespaces.filter((n) => !selectedNamespaces.includes(n)) as ns (ns)}
						<option value={ns}></option>
					{/each}
				</datalist>
				<Button
					variant="outline"
					size="icon-sm"
					onclick={commitNamespace}
					disabled={!nsInput.trim()}
					aria-label="Add namespace"
					class="size-7"
				>
					<PlusIcon class="size-3.5" />
				</Button>
			</div>
		</div>

		<!-- Labels -->
		<div class="space-y-2 border-b p-3">
			<div class="flex items-center justify-between">
				<span class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Labels</span>
				<span class="text-xs text-muted-foreground">match all</span>
			</div>

			{#if labelConditions.length > 0}
				<div class="space-y-1.5">
					{#each labelConditions as condition, i (i)}
						<div class="flex items-center gap-1.5">
							<!-- Key input -->
							<!-- svelte-ignore a11y_autocomplete_valid -->
							<Input
								list="lk-suggestions-{i}"
								value={condition.key}
								placeholder="key"
								class="h-7 w-28 font-mono text-xs"
								oninput={(e) => setKey(i, (e.target as HTMLInputElement).value)}
							/>
							<datalist id="lk-suggestions-{i}">
								{#each allLabelKeys as k (k)}
									<option value={k}></option>
								{/each}
							</datalist>

							<span class="text-xs text-muted-foreground">=</span>

							<!-- Value input -->
							<!-- svelte-ignore a11y_autocomplete_valid -->
							<Input
								list="lv-suggestions-{i}"
								value={condition.value}
								placeholder="value"
								disabled={!condition.key}
								class="h-7 w-28 font-mono text-xs"
								oninput={(e) => setValue(i, (e.target as HTMLInputElement).value)}
							/>
							<datalist id="lv-suggestions-{i}">
								{#each valuesForKey(condition.key) as v (v)}
									<option value={v}></option>
								{/each}
							</datalist>

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

		<!-- Preview table -->
		<div>
			<!-- Header bar -->
			<div class="flex items-center justify-between border-b px-4 py-2">
				<div class="flex items-center gap-2">
					<span class="text-sm font-semibold">
						{#if hasActiveFilters}Matching{:else}All{/if} devices
					</span>
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
							<Table.Head class="h-8 text-xs font-medium uppercase tracking-wide text-muted-foreground">Name</Table.Head>
							<Table.Head class="h-8 text-xs font-medium uppercase tracking-wide text-muted-foreground">Namespace</Table.Head>
							<Table.Head class="h-8 text-xs font-medium uppercase tracking-wide text-muted-foreground">Labels</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#if previewDevices.length === 0}
							<Table.Row class="hover:bg-transparent">
								<Table.Cell colspan={4} class="py-8 text-center">
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
										<span class={cn(
											'block h-2 w-2 rounded-full',
											d.status?.online
												? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
												: 'bg-muted-foreground/30'
										)}></span>
									</Table.Cell>
									<Table.Cell class="text-sm font-medium">{d.meta?.name ?? '—'}</Table.Cell>
									<Table.Cell>
										<Badge variant="outline" class="font-mono text-xs font-normal">
											{d.meta?.namespace ?? 'default'}
										</Badge>
									</Table.Cell>
									<Table.Cell>
										<div class="flex flex-wrap gap-1">
											{#each Object.entries(d.meta?.labels ?? {}) as [k, v] (k)}
												<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
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
	</div>

<!-- ─── Specific mode ──────────────────────────────────────────────────────── -->
{:else}
	<DevicePicker {devices} bind:selectedIds {isLoading} />
{/if}
