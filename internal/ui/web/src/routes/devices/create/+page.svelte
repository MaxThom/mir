<script lang="ts">
	import { goto } from '$app/navigation';
	import { Device, Meta, DeviceSpec } from '@mir/sdk';
	import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
	import { ROUTES } from '$lib/shared/constants/routes';
	import * as Card from '$lib/shared/components/shadcn/card';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Separator } from '$lib/shared/components/shadcn/separator';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { KeyValueEditor } from '$lib/shared/components/ui/key-value-editor';
	import { CodeEditor } from '$lib/shared/components/ui/code-editor';
	import CheckIcon from '@lucide/svelte/icons/check';
	import FileCode2Icon from '@lucide/svelte/icons/file-code-2';

	// ── Form state ────────────────────────────────────────────────────────────
	let name = $state('');
	let namespace = $state('default');
	let deviceId = $state('');
	let disabled = $state(false);
	let labels = $state<{ key: string; value: string }[]>([]);
	let annotations = $state<{ key: string; value: string }[]>([]);

	let nameError = $state('');
	let deviceIdError = $state('');

	// ── Code editor state ─────────────────────────────────────────────────────
	let isCodeEditing = $state(false);
	let codeContent = $state('');
	let isCodeSaving = $state(false);
	let codeError = $state<string | null>(null);

	function buildTemplate() {
		const obj: Record<string, unknown> = {
			apiVersion: 'v1',
			kind: 'Device',
			metadata: {
				name: name || 'my-device',
				namespace: namespace || 'default',
				...(labels.length ? { labels: Object.fromEntries(labels.map(({ key, value }) => [key, value])) } : {}),
				...(annotations.length ? { annotations: Object.fromEntries(annotations.map(({ key, value }) => [key, value])) } : {})
			},
			spec: {
				deviceId: deviceId || 'device-001',
				...(disabled ? { disabled: true } : {})
			}
		};
		return editorPrefs.json ? JSON.stringify(obj, null, 2) : stringifyYaml(obj, { lineWidth: 0 });
	}

	function openCodeEditor() {
		codeContent = buildTemplate();
		codeError = null;
		isCodeEditing = true;
	}

	function cancelCode() {
		isCodeEditing = false;
		codeError = null;
	}

	async function parseAndApplyCode(text: string) {
		const parsed = (editorPrefs.json ? JSON.parse(text) : parseYaml(text)) as Record<string, unknown>;
		const meta = (parsed.metadata ?? parsed.meta ?? {}) as Record<string, unknown>;
		const spec = (parsed.spec ?? {}) as Record<string, unknown>;

		const resolvedName = String(meta.name ?? '').trim();
		const resolvedDeviceId = String(spec.deviceId ?? '').trim();
		if (!resolvedName) throw new Error('metadata.name is required');
		if (!resolvedDeviceId) throw new Error('spec.deviceId is required');

		const toKV = (raw: unknown) =>
			Object.entries((raw ?? {}) as Record<string, string>).map(([key, value]) => ({ key, value: String(value) }));

		name = resolvedName;
		namespace = String(meta.namespace ?? 'default').trim();
		deviceId = resolvedDeviceId;
		disabled = Boolean(spec.disabled ?? false);
		labels = toKV(meta.labels);
		annotations = toKV(meta.annotations);
	}

	async function saveCode(text: string) {
		if (!mirStore.mir) return;
		isCodeSaving = true;
		codeError = null;
		try {
			await parseAndApplyCode(text);
			await submit(false);
		} catch (err) {
			codeError = err instanceof Error ? err.message : 'Failed to parse';
		} finally {
			isCodeSaving = false;
		}
	}

	async function saveCodeMany(text: string) {
		if (!mirStore.mir) return;
		isCodeSaving = true;
		codeError = null;
		try {
			await parseAndApplyCode(text);
			await submit(true);
		} catch (err) {
			codeError = err instanceof Error ? err.message : 'Failed to parse';
		} finally {
			isCodeSaving = false;
		}
	}

	// ── Form submit ───────────────────────────────────────────────────────────
	function validate() {
		nameError = '';
		deviceIdError = '';
		let ok = true;
		if (!name.trim()) { nameError = 'Required'; ok = false; }
		if (!deviceId.trim()) { deviceIdError = 'Required'; ok = false; }
		return ok;
	}

async function submit(stayOnPage = false) {
		if (!validate() || !mirStore.mir) return;
		const labelMap = Object.fromEntries(labels.map(({ key, value }) => [key, value]));
		const annotationMap = Object.fromEntries(annotations.map(({ key, value }) => [key, value]));
		const created = await deviceStore.createDevice(
			mirStore.mir,
			name.trim(),
			namespace.trim(),
			deviceId.trim(),
			{ disabled, labels: labelMap, annotations: annotationMap }
		);
		if (!stayOnPage) {
			goto(ROUTES.DEVICES.DETAIL(created.spec.deviceId));
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		try { await submit(); } catch { /* error shown via deviceStore.createError */ }
	}

	async function handleCreateMany() {
		try { await submit(true); } catch { /* error shown via deviceStore.createError */ }
	}
</script>

<div class="flex flex-col gap-4">
	<form onsubmit={handleSubmit} class="w-1/2">
		<Card.Root class="gap-0 py-4">
			{#if isCodeEditing}
				<Card.Content class="flex flex-col gap-2 px-6">
					<CodeEditor
						content={codeContent}
						onSave={saveCode}
						onSaveMany={saveCodeMany}
						onCancel={cancelCode}
						isSaving={isCodeSaving}
						error={codeError}
						saveLabel="Create"
						saveManyLabel="Create Many"
						cancelIconOnly={true}
					/>
				</Card.Content>
			{:else}
				<Card.Content class="space-y-2 px-6">

					<!-- ── Meta ── -->
					<div class="flex items-center justify-between">
						<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Meta</p>
						<div class="flex items-center gap-1">
							<Button variant="ghost" size="sm" type="submit" disabled={deviceStore.isCreating || !mirStore.mir} class="h-7 gap-1 text-xs">
								{#if deviceStore.isCreating}
									<Spinner class="size-3" />
								{:else}
									<CheckIcon class="size-3" />
								{/if}
								Create
							</Button>
							<Button variant="ghost" size="sm" type="button" onclick={handleCreateMany} disabled={deviceStore.isCreating || !mirStore.mir} class="h-7 gap-1 text-xs">
								{#if deviceStore.isCreating}
									<Spinner class="size-3" />
								{:else}
									<CheckIcon class="size-3" />
								{/if}
								Create Many
							</Button>
							<Button variant="ghost" size="icon-sm" onclick={openCodeEditor} class="size-7" type="button">
								<FileCode2Icon class="size-3.5" />
								<span class="sr-only">Edit as YAML/JSON</span>
							</Button>
						</div>
					</div>

					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Name</span>
						<div class="flex flex-1 flex-col gap-0.5">
							<Input bind:value={name} class="h-7 flex-1 text-sm {nameError ? 'border-destructive' : ''}" placeholder="my-device" />
							{#if nameError}<p class="text-xs text-destructive">{nameError}</p>{/if}
						</div>
					</div>

					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Namespace</span>
						<Input bind:value={namespace} class="h-7 flex-1 font-mono text-sm" placeholder="default" />
					</div>

					<div class="flex items-start justify-between gap-4">
						<span class="w-28 shrink-0 pt-0.5 text-sm text-muted-foreground">Labels</span>
						<div class="flex-1">
							<KeyValueEditor bind:items={labels} isEditing variant="badge" addLabel="Add label" />
						</div>
					</div>

					<div class="flex items-start justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Annotations</span>
						<div class="flex-1">
							<KeyValueEditor bind:items={annotations} isEditing variant="list" addLabel="Add annotation" />
						</div>
					</div>

					<Separator />

					<!-- ── Spec ── -->
					<p class="text-xs font-medium tracking-wide text-muted-foreground uppercase">Spec</p>

					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Device ID</span>
						<div class="flex flex-1 flex-col gap-0.5">
							<Input bind:value={deviceId} class="h-7 flex-1 font-mono text-sm {deviceIdError ? 'border-destructive' : ''}" placeholder="device-001" />
							{#if deviceIdError}<p class="text-xs text-destructive">{deviceIdError}</p>{/if}
						</div>
					</div>

					<div class="flex items-center justify-between gap-4">
						<span class="w-28 shrink-0 text-sm text-muted-foreground">Disabled</span>
						<div class="flex-1">
							<label class="flex cursor-pointer items-center gap-2">
								<input type="checkbox" bind:checked={disabled} class="h-4 w-4 rounded border border-input accent-primary" />
								<span class="text-sm">{disabled ? 'Disabled' : 'Enabled'}</span>
							</label>
						</div>
					</div>

					{#if deviceStore.createError}
						<p class="text-sm text-destructive">{deviceStore.createError}</p>
					{/if}

				</Card.Content>
			{/if}
		</Card.Root>
	</form>
</div>
