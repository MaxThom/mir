<script lang="ts">
	import { getContext } from 'svelte';
	import type { Device, OptString, UpdateDeviceRequest } from '@mir/sdk';
	import {
		UpdateDeviceRequestSchema,
		UpdateDeviceRequest_MetaSchema,
		UpdateDeviceRequest_SpecSchema,
		DeviceTargetSchema,
		OptStringSchema
	} from '@mir/sdk';
	import { create } from '@bufbuild/protobuf';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Separator } from '$lib/components/ui/separator';
	import { Spinner } from '$lib/components/ui/spinner';
	import { cn } from '$lib/utils';
	import { relativeTime, formatFullDate } from '$lib/shared/utils/time';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
	import { EditorView, basicSetup } from 'codemirror';
	import { Compartment } from '@codemirror/state';
	import { yaml as yamlLang } from '@codemirror/lang-yaml';
	import { json as jsonLang } from '@codemirror/lang-json';
	import { oneDark } from '@codemirror/theme-one-dark';
	import { vim, Vim } from '@replit/codemirror-vim';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import FileCode2Icon from '@lucide/svelte/icons/file-code-2';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XIcon from '@lucide/svelte/icons/x';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import CircleCheckBigIcon from '@lucide/svelte/icons/circle-check-big';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { getHighlighter } from '$lib/utils/highlighter';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { eventStore } from '$lib/domains/events/stores/event.svelte';
	import { JsonValue } from '$lib/components/ui/json-value';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	const ctx = getContext<{ device: Device | null }>('device');
	let device = $derived(ctx.device);

	// ── Edit state ──────────────────────────────────────────────────────────────
	let isEditing = $state(false);
	let editName = $state('');
	let editNamespace = $state('');
	let editDisabled = $state(false);
	let editLabels = $state<{ key: string; value: string }[]>([]);
	let editAnnotations = $state<{ key: string; value: string }[]>([]);

	function startEdit() {
		editName = device?.meta?.name ?? '';
		editNamespace = device?.meta?.namespace ?? '';
		editDisabled = device?.spec?.disabled ?? false;
		editLabels = Object.entries(device?.meta?.labels ?? {}).map(([key, value]) => ({ key, value }));
		editAnnotations = Object.entries(device?.meta?.annotations ?? {}).map(([key, value]) => ({
			key,
			value
		}));
		isEditing = true;
	}

	function cancelEdit() {
		isEditing = false;
		deviceStore.updateError = null;
	}

	async function saveEdit() {
		if (!mirStore.mir || !device) return;

		const origLabels = device.meta?.labels ?? {};
		const newLabelsMap: Record<string, OptString> = {};
		for (const { key, value } of editLabels.filter((l) => l.key.trim())) {
			newLabelsMap[key.trim()] = create(OptStringSchema, { value });
		}
		for (const key of Object.keys(origLabels)) {
			if (!(key in newLabelsMap)) newLabelsMap[key] = create(OptStringSchema, {});
		}

		const origAnnotations = device.meta?.annotations ?? {};
		const newAnnotationsMap: Record<string, OptString> = {};
		for (const { key, value } of editAnnotations.filter((a) => a.key.trim())) {
			newAnnotationsMap[key.trim()] = create(OptStringSchema, { value });
		}
		for (const key of Object.keys(origAnnotations)) {
			if (!(key in newAnnotationsMap)) newAnnotationsMap[key] = create(OptStringSchema, {});
		}

		const request: UpdateDeviceRequest = create(UpdateDeviceRequestSchema, {
			targets: create(DeviceTargetSchema, { ids: [device.spec?.deviceId ?? ''] }),
			meta: create(UpdateDeviceRequest_MetaSchema, {
				name: editName.trim() || undefined,
				namespace: editNamespace.trim() || undefined,
				labels: newLabelsMap,
				annotations: newAnnotationsMap
			}),
			spec: create(UpdateDeviceRequest_SpecSchema, { disabled: editDisabled })
		});

		try {
			await deviceStore.updateDevice(mirStore.mir, request);
			isEditing = false;
		} catch {
			// error stored in deviceStore.updateError
		}
	}

	// ── YAML editor state ────────────────────────────────────────────────────────
	let isYamlEditing = $state(false);
	let yamlContent = $state('');
	let yamlError = $state<string | null>(null);
	let isYamlSaving = $state(false);
	let isVimMode = $derived(editorPrefs.vim);
	let isJsonMode = $derived(editorPrefs.json);
	let cmEditorEl = $state<HTMLElement | null>(null);
	let cmView: EditorView | null = null;
	const vimCompartment = new Compartment();
	const langCompartment = new Compartment();

	function tsToIso(ts?: { seconds: bigint }): string | undefined {
		if (!ts) return undefined;
		return new Date(Number(ts.seconds) * 1000).toISOString();
	}

	function openYamlEditor() {
		if (!device) return;

		const obj: Record<string, unknown> = {
			apiVersion: device.apiVersion || 'v1',
			kind: device.kind || 'Device',
			metadata: {
				name: device.meta?.name ?? '',
				namespace: device.meta?.namespace ?? '',
				...(Object.keys(device.meta?.labels ?? {}).length
					? { labels: device.meta!.labels }
					: {}),
				...(Object.keys(device.meta?.annotations ?? {}).length
					? { annotations: device.meta!.annotations }
					: {})
			},
			spec: {
				deviceId: device.spec?.deviceId ?? '',
				disabled: device.spec?.disabled ?? false
			}
		};

		// properties (desired / reported) — read-only on save
		const desired = device.properties?.desired ?? {};
		const reported = device.properties?.reported ?? {};
		if (Object.keys(desired).length || Object.keys(reported).length) {
			obj.properties = {
				...(Object.keys(desired).length ? { desired } : {}),
				...(Object.keys(reported).length ? { reported } : {})
			};
		}

		// status — read-only on save
		const s = device.status;
		if (s) {
			obj.status = {
				online: s.online,
				...(s.lastHearthbeat ? { lastHeartbeat: tsToIso(s.lastHearthbeat) } : {}),
				...(s.schema?.packageNames?.length
					? { schema: { packages: s.schema.packageNames } }
					: {})
			};
		}

		yamlContent = isJsonMode
			? JSON.stringify(obj, null, 2)
			: stringifyYaml(obj, { lineWidth: 0 });
		yamlError = null;
		isYamlEditing = true;
	}

	function cancelYaml() {
		isYamlEditing = false;
		yamlError = null;
		cmView?.destroy();
		cmView = null;
	}

	async function saveYaml() {
		if (!mirStore.mir || !device) return;
		isYamlSaving = true;
		yamlError = null;
		try {
			const text = cmView ? cmView.state.doc.toString() : yamlContent;
			const parsed = (isJsonMode ? JSON.parse(text) : parseYaml(text)) as Record<string, unknown>;
			const meta = (parsed.metadata ?? parsed.meta ?? {}) as Record<string, unknown>;
			const spec = (parsed.spec ?? {}) as Record<string, unknown>;

			const origLabels = device.meta?.labels ?? {};
			const newLabelsRaw = (meta.labels ?? {}) as Record<string, string>;
			const newLabelsMap: Record<string, OptString> = {};
			for (const [k, v] of Object.entries(newLabelsRaw))
				newLabelsMap[k] = create(OptStringSchema, { value: String(v) });
			for (const k of Object.keys(origLabels))
				if (!(k in newLabelsMap)) newLabelsMap[k] = create(OptStringSchema, {});

			const origAnnotations = device.meta?.annotations ?? {};
			const newAnnotationsRaw = (meta.annotations ?? {}) as Record<string, string>;
			const newAnnotationsMap: Record<string, OptString> = {};
			for (const [k, v] of Object.entries(newAnnotationsRaw))
				newAnnotationsMap[k] = create(OptStringSchema, { value: String(v) });
			for (const k of Object.keys(origAnnotations))
				if (!(k in newAnnotationsMap)) newAnnotationsMap[k] = create(OptStringSchema, {});

			const request: UpdateDeviceRequest = create(UpdateDeviceRequestSchema, {
				targets: create(DeviceTargetSchema, { ids: [device.spec?.deviceId ?? ''] }),
				meta: create(UpdateDeviceRequest_MetaSchema, {
					name: String(meta.name ?? '').trim() || undefined,
					namespace: String(meta.namespace ?? '').trim() || undefined,
					labels: newLabelsMap,
					annotations: newAnnotationsMap
				}),
				spec: create(UpdateDeviceRequest_SpecSchema, {
					disabled: Boolean(spec.disabled ?? false)
				})
			});

			await deviceStore.updateDevice(mirStore.mir, request);
			cancelYaml();
		} catch (err) {
			yamlError = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			isYamlSaving = false;
		}
	}

	function toggleVim() {
		editorPrefs.setVim(!isVimMode);
		cmView?.dispatch({
			effects: vimCompartment.reconfigure(!isVimMode ? vim() : [])
		});
	}

	function toggleFormat() {
		if (!cmView) return;
		const text = cmView.state.doc.toString();
		try {
			yamlContent = isJsonMode
				? stringifyYaml(JSON.parse(text), { lineWidth: 0 })
				: JSON.stringify(parseYaml(text), null, 2);
		} catch {
			yamlContent = text;
		}
		editorPrefs.setJson(!isJsonMode);
		// $effect batches both state changes and recreates the editor with new content + language
	}

	$effect(() => {
		if (isYamlEditing && cmEditorEl) {
			Vim.defineEx('write', 'w', () => saveYaml());
			Vim.defineEx('quit', 'q', () => cancelYaml());
			cmView?.destroy();
			cmView = new EditorView({
				doc: yamlContent,
				extensions: [
					vimCompartment.of(isVimMode ? vim() : []),
					langCompartment.of(isJsonMode ? jsonLang() : yamlLang()),
					basicSetup,
					oneDark
				],
				parent: cmEditorEl
			});
		}
	});

	// ── Events ───────────────────────────────────────────────────────────────────
	$effect(() => {
		if (mirStore.mir && device?.meta?.name) {
			eventStore.loadEvents(mirStore.mir, device.meta.name);
		} else {
			eventStore.reset();
		}
	});

	let expandedEvents = $state(new Set<number>());

	function toggleEvent(i: number) {
		if (expandedEvents.has(i)) {
			expandedEvents.delete(i);
		} else {
			expandedEvents.add(i);
		}
		expandedEvents = new Set(expandedEvents);
	}

	function decodePayload(bytes: Uint8Array): string {
		if (!bytes || bytes.length === 0) return '';
		try {
			const text = new TextDecoder().decode(bytes);
			return JSON.stringify(JSON.parse(text), null, 2);
		} catch {
			return new TextDecoder().decode(bytes);
		}
	}

	let highlightedPayloads = $state<Record<number, string>>({});

	async function highlightPayload(i: number, payload: string) {
		if (highlightedPayloads[i] !== undefined) return;
		const hl = await getHighlighter();
		highlightedPayloads[i] = hl.codeToHtml(payload, {
			lang: 'json',
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
	}

	// ── Properties derived state ─────────────────────────────────────────────────
	let desiredProps = $derived(Object.entries(device?.properties?.desired ?? {}));
	let reportedProps = $derived(Object.entries(device?.properties?.reported ?? {}));

	function isMatchingDesired(key: string, reportedVal: unknown): boolean {
		const desired = (device?.properties?.desired ?? {}) as Record<string, unknown>;
		if (!(key in desired)) return false;
		return JSON.stringify(desired[key]) === JSON.stringify(reportedVal);
	}

	function displayValue(val: unknown): string {
		if (val === null || val === undefined) return '—';
		if (typeof val === 'object') return JSON.stringify(val);
		return String(val);
	}
</script>

{#if device}
	<div class="flex flex-col gap-4">
		<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
			<!-- ── Device card ──────────────────────────────────────────────────────── -->
			<Card.Root class="gap-0 py-4">
				{#if isYamlEditing}
					<!-- ── YAML editor mode ── -->
					<Card.Content class="flex flex-col gap-2 px-6">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<div class="flex overflow-hidden rounded border border-input font-mono text-[10px]">
									<button
										onclick={() => isJsonMode && toggleFormat()}
										class="px-2 py-0.5 transition-colors {!isJsonMode
											? 'bg-secondary text-secondary-foreground'
											: 'text-muted-foreground hover:text-foreground'}"
									>YAML</button>
									<button
										onclick={() => !isJsonMode && toggleFormat()}
										class="px-2 py-0.5 transition-colors {isJsonMode
											? 'bg-secondary text-secondary-foreground'
											: 'text-muted-foreground hover:text-foreground'}"
									>JSON</button>
								</div>
								<button
									onclick={toggleVim}
									class="rounded px-2 py-0.5 font-mono text-[10px] transition-colors {isVimMode
										? 'bg-secondary text-secondary-foreground'
										: 'text-muted-foreground hover:text-foreground'}"
								>
									VIM
								</button>
							</div>
							<div class="flex items-center gap-1">
								<Button
									variant="ghost"
									size="sm"
									onclick={saveYaml}
									disabled={isYamlSaving}
									class="h-7 gap-1 text-xs"
								>
									{#if isYamlSaving}<Spinner class="size-3" />{:else}<CheckIcon class="size-3" />{/if}
									Save
								</Button>
								<Button
									variant="ghost"
									size="sm"
									onclick={cancelYaml}
									disabled={isYamlSaving}
									class="h-7 gap-1 text-xs"
								>
									<XIcon class="size-3" />
									Cancel
								</Button>
							</div>
						</div>
						{#if yamlError}
							<p class="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
								{yamlError}
							</p>
						{/if}
						<div
							bind:this={cmEditorEl}
							class="overflow-hidden rounded-md border border-input [&_.cm-editor]:min-h-64 [&_.cm-editor]:outline-none [&_.cm-scroller]:font-mono [&_.cm-scroller]:text-xs"
						></div>
					</Card.Content>
				{:else}
					<!-- ── Normal meta/spec/status view ── -->
					<Card.Content class="space-y-2 px-6">
					<!-- ── Meta (+ controls) ── -->
					<div class="flex items-center justify-between">
						<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Meta</p>
						{#if isEditing}
							<div class="flex items-center gap-1">
								<Button
									variant="ghost"
									size="sm"
									onclick={saveEdit}
									disabled={deviceStore.isUpdating}
									class="h-7 gap-1 text-xs"
								>
									{#if deviceStore.isUpdating}
										<Spinner class="size-3" />
									{:else}
										<CheckIcon class="size-3" />
									{/if}
									Save
								</Button>
								<Button
									variant="ghost"
									size="sm"
									onclick={cancelEdit}
									disabled={deviceStore.isUpdating}
									class="h-7 gap-1 text-xs"
								>
									<XIcon class="size-3" />
									Cancel
								</Button>
							</div>
						{:else}
							<div class="flex items-center gap-1">
								<Button variant="ghost" size="icon-sm" onclick={startEdit} class="size-7">
									<PencilIcon class="size-3.5" />
									<span class="sr-only">Edit device</span>
								</Button>
								<Button variant="ghost" size="icon-sm" onclick={openYamlEditor} class="size-7">
									<FileCode2Icon class="size-3.5" />
									<span class="sr-only">Edit as YAML</span>
								</Button>
							</div>
						{/if}
					</div>

					{#if deviceStore.updateError}
						<p class="rounded-md bg-destructive/10 px-3 py-2 text-xs text-destructive">
							{deviceStore.updateError}
						</p>
					{/if}

					<!-- Name -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Name</span>
						{#if isEditing}
							<Input bind:value={editName} class="h-7 flex-1 text-sm" placeholder="Device name" />
						{:else}
							<span class="flex-1 text-sm font-medium">{device.meta?.name ?? '—'}</span>
						{/if}
					</div>

					<!-- Namespace -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Namespace</span>
						{#if isEditing}
							<Input
								bind:value={editNamespace}
								class="h-7 flex-1 font-mono text-sm"
								placeholder="default"
							/>
						{:else}
							<span class="flex-1 font-mono text-sm">{device.meta?.namespace ?? '—'}</span>
						{/if}
					</div>

					<!-- Labels -->
					<div class="flex items-start justify-between gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Labels</span>
						<div class="flex-1">
							{#if isEditing}
								<div class="space-y-1.5">
									{#each editLabels as label, i (i)}
										<div class="flex items-center gap-1">
											<Input
												bind:value={label.key}
												placeholder="key"
												class="h-7 w-24 font-mono text-xs"
											/>
											<span class="text-muted-foreground">=</span>
											<Input
												bind:value={label.value}
												placeholder="value"
												class="h-7 flex-1 font-mono text-xs"
											/>
											<Button
												variant="ghost"
												size="icon-sm"
												onclick={() => editLabels.splice(i, 1)}
												class="size-7"
											>
												<XIcon class="size-3" />
											</Button>
										</div>
									{/each}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => editLabels.push({ key: '', value: '' })}
										class="h-7 gap-1 px-2 text-xs"
									>
										<PlusIcon class="size-3" />
										Add label
									</Button>
								</div>
							{:else}
								{@const labels = Object.entries(device.meta?.labels ?? {})}
								{#if labels.length > 0}
									<div class="flex flex-wrap gap-1">
										{#each labels as [k, v] (k)}
											<Badge variant="secondary" class="font-mono text-xs font-normal"
												>{k}={v}</Badge
											>
										{/each}
									</div>
								{:else}
									<span class="text-sm text-muted-foreground">—</span>
								{/if}
							{/if}
						</div>
					</div>

					<!-- Annotations -->
					<div class="flex items-start justify-between gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Annotations</span>
						<div class="flex-1">
							{#if isEditing}
								<div class="space-y-1.5">
									{#each editAnnotations as annotation, i (i)}
										<div class="flex items-center gap-1">
											<Input
												bind:value={annotation.key}
												placeholder="key"
												class="h-7 w-24 font-mono text-xs"
											/>
											<span class="text-muted-foreground">=</span>
											<Input
												bind:value={annotation.value}
												placeholder="value"
												class="h-7 flex-1 font-mono text-xs"
											/>
											<Button
												variant="ghost"
												size="icon-sm"
												onclick={() => editAnnotations.splice(i, 1)}
												class="size-7"
											>
												<XIcon class="size-3" />
											</Button>
										</div>
									{/each}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => editAnnotations.push({ key: '', value: '' })}
										class="h-7 gap-1 px-2 text-xs"
									>
										<PlusIcon class="size-3" />
										Add annotation
									</Button>
								</div>
							{:else}
								{@const annotations = Object.entries(device.meta?.annotations ?? {})}
								{#if annotations.length > 0}
									<div class="space-y-1">
										{#each annotations as [k, v] (k)}
											<div class="flex gap-2">
												<span class="font-mono text-xs text-muted-foreground">{k}:</span>
												<span class="font-mono text-xs">{v}</span>
											</div>
										{/each}
									</div>
								{:else}
									<span class="text-sm text-muted-foreground">—</span>
								{/if}
							{/if}
						</div>
					</div>

					<Separator />

					<!-- ── Spec ── -->
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Spec</p>

					<!-- Device ID (always read-only) -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Device ID</span>
						<span class="flex-1 font-mono text-xs text-muted-foreground">
							{device.spec?.deviceId ?? '—'}
						</span>
					</div>

					<!-- Disabled -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Disabled</span>
						<div class="flex-1">
							{#if isEditing}
								<label class="flex cursor-pointer items-center gap-2">
									<input
										type="checkbox"
										bind:checked={editDisabled}
										class="h-4 w-4 rounded border border-input accent-primary"
									/>
									<span class="text-sm">{editDisabled ? 'Disabled' : 'Enabled'}</span>
								</label>
							{:else if device.spec?.disabled}
								<Badge variant="destructive" class="text-xs">Yes</Badge>
							{:else}
								<span class="text-sm text-muted-foreground">No</span>
							{/if}
						</div>
					</div>

					<Separator />

					<!-- ── Status ── -->
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Status</p>

					<!-- Connectivity (read-only) -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Connectivity</span>
						<div class="flex flex-1 items-center gap-2">
							<span
								class={cn(
									'h-2 w-2 shrink-0 rounded-full',
									device.status?.online
										? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
										: 'bg-muted-foreground/30'
								)}
							></span>
							<span
								class={cn(
									'text-sm font-medium',
									device.status?.online
										? 'text-emerald-600 dark:text-emerald-400'
										: 'text-muted-foreground'
								)}
							>
								{device.status?.online ? 'Online' : 'Offline'}
							</span>
						</div>
					</div>

					<!-- Last Heartbeat (read-only) -->
					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Last Heartbeat</span>
						<div class="flex-1">
							{#if device.status?.lastHearthbeat}
								<Tooltip.Root>
									<Tooltip.Trigger
										class="cursor-default text-sm underline decoration-dotted underline-offset-2 hover:text-foreground"
									>
										{relativeTime(device.status.lastHearthbeat.seconds)}
									</Tooltip.Trigger>
									<Tooltip.Content>
										{formatFullDate(device.status.lastHearthbeat.seconds)}
									</Tooltip.Content>
								</Tooltip.Root>
							{:else}
								<span class="text-sm text-muted-foreground">—</span>
							{/if}
						</div>
					</div>

					<!-- Schema (read-only) -->
					<div class="flex items-start justify-between gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Schema</span>
						<div class="flex-1">
							{#if device.status?.schema?.packageNames?.length}
								<div class="flex flex-wrap gap-1">
									{#each device.status.schema.packageNames as pkg (pkg)}
										<Badge variant="outline" class="font-mono text-xs font-normal">{pkg}</Badge>
									{/each}
								</div>
								{#if device.status.schema.lastSchemaFetch}
									<Tooltip.Root>
										<Tooltip.Trigger
											class="mt-0.5 cursor-default text-xs text-muted-foreground underline decoration-dotted underline-offset-2 hover:text-foreground"
										>
											fetched {relativeTime(device.status.schema.lastSchemaFetch.seconds)}
										</Tooltip.Trigger>
										<Tooltip.Content>
											{formatFullDate(device.status.schema.lastSchemaFetch.seconds)}
										</Tooltip.Content>
									</Tooltip.Root>
								{/if}
							{:else}
								<span class="text-sm text-muted-foreground">Not loaded</span>
							{/if}
						</div>
					</div>
				</Card.Content>
				{/if}
			</Card.Root>

			<!-- ── Properties card ────────────────────────────────────────────────── -->
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
															<span class="text-[10px] text-muted-foreground/60">
																{relativeTime(device.status.properties.desired[k].seconds)}
															</span>
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
															<span class="text-[10px] text-muted-foreground/60">
																{relativeTime(device.status.properties.reported[k].seconds)}
															</span>
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
		</div>

		<!-- Events card — full width -->
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
					<div class="w-full max-h-72 overflow-y-auto">
						{#each eventStore.events as event, i (i)}
							{@const expanded = expandedEvents.has(i)}
							{@const payload = decodePayload(event.spec?.jsonPayload ?? new Uint8Array())}
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
										<Tooltip.Root>
											<Tooltip.Trigger
												onclick={(e) => e.stopPropagation()}
												class="shrink-0 cursor-default text-[10px] text-muted-foreground underline decoration-dotted underline-offset-2"
											>
												{relativeTime(event.status.lastAt.seconds)}
											</Tooltip.Trigger>
											<Tooltip.Content
												>{formatFullDate(event.status.lastAt.seconds)}</Tooltip.Content
											>
										</Tooltip.Root>
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
											{@const _ = highlightPayload(i, payload)}
											<div
												class="overflow-hidden rounded border border-border text-[10px] leading-relaxed [&>pre]:whitespace-pre-wrap [&>pre]:break-all [&>pre]:px-3 [&>pre]:py-2"
											>
												{#if highlightedPayloads[i]}
													{@html highlightedPayloads[i]}
												{:else}
													<pre
														class="bg-muted px-3 py-2 font-mono text-[10px] whitespace-pre-wrap break-all">{payload}</pre>
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
	</div>
{/if}
