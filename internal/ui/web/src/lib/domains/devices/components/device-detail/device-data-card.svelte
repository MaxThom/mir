<script lang="ts">
	import { Device, Meta, DeviceSpec, DeviceProperties } from '@mir/sdk';
	import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { cn } from '$lib/utils';
	import { TimeTooltip } from '$lib/shared/components/ui/time-tooltip';
	import { KeyValueEditor } from '$lib/shared/components/ui/key-value-editor';
	import { CodeEditor } from '$lib/shared/components/ui/code-editor';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { setDeviceLabels } from '../../utils/device';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import FileCode2Icon from '@lucide/svelte/icons/file-code-2';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XIcon from '@lucide/svelte/icons/x';

	let { device }: { device: Device } = $props();

	// ── Edit state ──────────────────────────────────────────────────────────────
	let isEditing = $state(false);
	let editName = $state('');
	let editNamespace = $state('');
	let editDisabled = $state(false);
	let editLabels = $state<{ key: string; value: string }[]>([]);
	let editAnnotations = $state<{ key: string; value: string }[]>([]);

	// View-mode derived items for KeyValueEditor
	let viewLabels = $derived(
		Object.entries(device.meta?.labels ?? {}).map(([key, value]) => ({ key, value }))
	);
	let viewAnnotations = $derived(
		Object.entries(device.meta?.annotations ?? {}).map(([key, value]) => ({ key, value }))
	);

	function startEdit() {
		editName = device.meta?.name ?? '';
		editNamespace = device.meta?.namespace ?? '';
		editDisabled = device.spec?.disabled ?? false;
		editLabels = Object.entries(device.meta?.labels ?? {}).map(([key, value]) => ({ key, value }));
		editAnnotations = Object.entries(device.meta?.annotations ?? {}).map(([key, value]) => ({
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
		if (!mirStore.mir) return;

		const newLabels = setDeviceLabels(device.meta?.labels ?? {}, editLabels);
		const newAnnotations = setDeviceLabels(device.meta?.annotations ?? {}, editAnnotations);
		const updated = new Device({
			...device,
			meta: new Meta({
				name: editName.trim() || device.meta.name,
				namespace: editNamespace.trim() || device.meta.namespace,
				labels: newLabels,
				annotations: newAnnotations
			}),
			spec: new DeviceSpec({ deviceId: device.spec.deviceId, disabled: editDisabled })
		});

		try {
			await deviceStore.updateDevice(mirStore.mir, updated);
			isEditing = false;
		} catch {
			// error stored in deviceStore.updateError
		}
	}

	// ── Code editor state ────────────────────────────────────────────────────────
	let isCodeEditing = $state(false);
	let codeContent = $state('');
	let isCodeSaving = $state(false);
	let codeError = $state<string | null>(null);

	function openCodeEditor() {
		const isJsonMode = editorPrefs.json;
		const obj: Record<string, unknown> = {
			apiVersion: device.apiVersion || 'v1',
			kind: device.kind || 'Device',
			metadata: {
				name: device.meta?.name ?? '',
				namespace: device.meta?.namespace ?? '',
				...(Object.keys(device.meta?.labels ?? {}).length ? { labels: device.meta!.labels } : {}),
				...(Object.keys(device.meta?.annotations ?? {}).length
					? { annotations: device.meta!.annotations }
					: {})
			},
			spec: {
				deviceId: device.spec?.deviceId ?? '',
				disabled: device.spec?.disabled ?? false
			}
		};

		const desired = device.properties?.desired ?? {};
		const reported = device.properties?.reported ?? {};
		if (Object.keys(desired).length || Object.keys(reported).length) {
			obj.properties = {
				...(Object.keys(desired).length ? { desired } : {}),
				...(Object.keys(reported).length ? { reported } : {})
			};
		}

		const s = device.status;
		if (s) {
			obj.status = {
				online: s.online,
				...(s.lastHearthbeat ? { lastHeartbeat: s.lastHearthbeat } : {}),
				...(s.schema?.packageNames?.length ? { schema: { packages: s.schema.packageNames } } : {})
			};
		}

		codeContent = isJsonMode ? JSON.stringify(obj, null, 2) : stringifyYaml(obj, { lineWidth: 0 });
		codeError = null;
		isCodeEditing = true;
	}

	function cancelCode() {
		isCodeEditing = false;
		codeError = null;
	}

	async function saveCode(text: string) {
		if (!mirStore.mir) return;
		isCodeSaving = true;
		codeError = null;
		try {
			const parsed = (editorPrefs.json ? JSON.parse(text) : parseYaml(text)) as Record<
				string,
				unknown
			>;
			const meta = (parsed.metadata ?? parsed.meta ?? {}) as Record<string, unknown>;
			const spec = (parsed.spec ?? {}) as Record<string, unknown>;
			const parsedProps = (parsed.properties ?? {}) as Record<string, unknown>;

			const toEditLabels = (raw: unknown) =>
				Object.entries((raw ?? {}) as Record<string, string>).map(([key, value]) => ({
					key,
					value: String(value)
				}));

			const updated = new Device({
				...device,
				meta: new Meta({
					name: String(meta.name ?? '').trim() || device.meta.name,
					namespace: String(meta.namespace ?? '').trim() || device.meta.namespace,
					labels: setDeviceLabels(device.meta?.labels ?? {}, toEditLabels(meta.labels)),
					annotations: setDeviceLabels(device.meta?.annotations ?? {}, toEditLabels(meta.annotations))
				}),
				spec: new DeviceSpec({
					deviceId: device.spec.deviceId,
					disabled: Boolean(spec.disabled ?? false)
				}),
				properties: new DeviceProperties({
					desired: ('desired' in parsedProps
						? parsedProps.desired
						: device.properties?.desired ?? {}) as Record<string, unknown>
				})
			});

			await deviceStore.updateDevice(mirStore.mir, updated);
			cancelCode();
		} catch (err) {
			codeError = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			isCodeSaving = false;
		}
	}
</script>

<Card.Root class="gap-0 py-4">
	{#if isCodeEditing}
		<!-- ── Code editor mode ── -->
		<Card.Content class="flex flex-col gap-2 px-6">
			<CodeEditor
				content={codeContent}
				onSave={saveCode}
				onCancel={cancelCode}
				isSaving={isCodeSaving}
				error={codeError}
			/>
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
						<Button variant="ghost" size="icon-sm" onclick={openCodeEditor} class="size-7">
							<FileCode2Icon class="size-3.5" />
							<span class="sr-only">Edit as YAML/JSON</span>
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
						<KeyValueEditor
							bind:items={editLabels}
							isEditing
							variant="badge"
							addLabel="Add label"
						/>
					{:else}
						<KeyValueEditor items={viewLabels} variant="badge" />
					{/if}
				</div>
			</div>

			<!-- Annotations -->
			<div class="flex items-start justify-between gap-4">
				<span class="w-28 shrink-0 text-sm text-muted-foreground">Annotations</span>
				<div class="flex-1">
					{#if isEditing}
						<KeyValueEditor
							bind:items={editAnnotations}
							isEditing
							variant="list"
							addLabel="Add annotation"
						/>
					{:else}
						<div class="mt-1"><KeyValueEditor items={viewAnnotations} variant="list" /></div>
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
						<TimeTooltip
							timestamp={device.status.lastHearthbeat}
							utc={editorPrefs.utc}
							class="text-sm hover:text-foreground"
						/>
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
							<TimeTooltip
								timestamp={device.status.schema.lastSchemaFetch}
								utc={editorPrefs.utc}
								prefix="fetched "
								class="mt-0.5 text-xs text-muted-foreground hover:text-foreground"
							/>
						{/if}
					{:else}
						<span class="text-sm text-muted-foreground">Not loaded</span>
					{/if}
				</div>
			</div>
		</Card.Content>
	{/if}
</Card.Root>
