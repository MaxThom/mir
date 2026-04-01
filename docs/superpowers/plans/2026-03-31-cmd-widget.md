# Command Widget Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `widget-command.svelte` to support multi-device command dispatch with grouped descriptor panel, broadcast + per-device send modes, inline response log, view state persistence, and fullscreen — mirroring the quality of `widget-telemetry.svelte`.

**Architecture:** `DescriptorPanel` in grouped mode on the left; `JsonPayloadEditor` + response log on the right. On load, all targeted devices are resolved via one `listCommands()` call. Send fans out per device with `Promise.allSettled`. Three files change: the widget (full rewrite), the config type (add `selectedCommand?`), and the grid (wire `onDevicesReady`).

**Tech Stack:** Svelte 5, TypeScript, `@mir/sdk` (`CommandGroup`, `CommandDescriptor`, `CommandResponseStatus`, `DeviceTarget`), CodeMirror (via `JsonPayloadEditor`), Lucide icons, TailwindCSS.

---

### Key Types (reference for all tasks)

From `@mir/sdk`:
```ts
type DeviceId = { id: string; name: string; namespace: string };

type CommandGroup = {
    ids: DeviceId[];
    descriptors: CommandDescriptor[];
    error: string;
};

type CommandDescriptor = {
    name: string;
    labels: Record<string, string>;
    template: string;
    error: string;
};

type SendCommandResult = Map<string, CommandResponse>;

type CommandResponse = {
    deviceId: string;
    name: string;
    payload: Uint8Array;
    status: CommandResponseStatus;
    error: string;
};

enum CommandResponseStatus {
    UNSPECIFIED = 0,
    PENDING = 1,
    VALIDATED = 2,
    ERROR = 3,
    SUCCESS = 4
}
```

From `$lib/domains/devices/types/types`:
```ts
type Descriptor = {
    name: string;
    labels: Record<string, string>;
    template: string;
    error: string;
};
// CommandDescriptor is structurally identical to Descriptor — they are compatible.
```

---

### Task 1: Update `CommandWidgetConfig` type

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts`

- [ ] **Step 1: Add `selectedCommand` view-state field**

Open `dashboard-api.ts`. Find the `CommandWidgetConfig` interface (currently lines 22–24) and add `selectedCommand?`:

```ts
export interface CommandWidgetConfig {
	target: DeviceTargetConfig;
	selectedCommand?: string;
}
```

- [ ] **Step 2: Verify type check passes**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -E "error|warning" | head -20
```

Expected: no new errors (3 pre-existing errors in `nav-section.svelte` and `multi/telemetry` are OK).

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts
git commit -m "feat(dashboard): add selectedCommand view-state field to CommandWidgetConfig"
```

---

### Task 2: Wire `onDevicesReady` in `dashboard-grid.svelte`

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte`

- [ ] **Step 1: Add `widgetId` and `onDevicesReady` to the `<WidgetCommand>` invocation**

Find this block in `dashboard-grid.svelte` (currently around line 108):
```svelte
{:else if widget.type === 'command'}
    <WidgetCommand config={widget.config as CommandWidgetConfig} />
```

Replace it with:
```svelte
{:else if widget.type === 'command'}
    <WidgetCommand
        widgetId={widget.id}
        config={widget.config as CommandWidgetConfig}
        onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
    />
```

- [ ] **Step 2: Verify type check passes**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -E "error|warning" | head -20
```

Expected: TypeScript will complain that `WidgetCommand` doesn't have `widgetId`/`onDevicesReady` props yet — that's fine, it will be resolved in Task 3.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/dashboard-grid.svelte
git commit -m "feat(dashboard): wire widgetId and onDevicesReady to WidgetCommand in grid"
```

---

### Task 3: Rewrite `widget-command.svelte`

**Files:**
- Modify (full rewrite): `internal/ui/web/src/lib/domains/dashboards/components/widget-command.svelte`

- [ ] **Step 1: Replace the entire file with the new implementation**

```svelte
<script lang="ts">
	import { untrack } from 'svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import { DescriptorPanel, JsonPayloadEditor } from '$lib/domains/devices/components/commands';
	import { DeviceTarget } from '@mir/sdk';
	import type { CommandDescriptor, CommandGroup } from '@mir/sdk';
	import { CommandResponseStatus } from '@mir/sdk';
	import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
	import { dashboardStore } from '$lib/domains/dashboards/stores/dashboard.svelte';
	import { CHART_COLORS } from '$lib/domains/devices/utils/tlm-time';
	import type { CommandWidgetConfig } from '../api/dashboard-api';
	import MaximizeIcon from '@lucide/svelte/icons/maximize';
	import MinimizeIcon from '@lucide/svelte/icons/minimize';
	import CheckIcon from '@lucide/svelte/icons/check';
	import XCircleIcon from '@lucide/svelte/icons/x-circle';

	let {
		config,
		widgetId,
		onDevicesReady
	}: {
		config: CommandWidgetConfig;
		widgetId: string;
		onDevicesReady?: (infos: { id: string; name: string; color: string }[]) => void;
	} = $props();

	// ─── Types ────────────────────────────────────────────────────────────────

	type CmdResponseEntry = {
		deviceId: string;
		deviceName: string;
		status: 'success' | 'error';
		message: string;
		durationMs: number;
		expanded: boolean;
	};

	// ─── State ────────────────────────────────────────────────────────────────

	let groups = $state<CommandGroup[]>([]);
	let isLoading = $state(false);
	let loadError = $state<string | null>(null);
	let hasLoaded = $state(false);

	let selectedGroupIdx = $state<number | null>(null);
	let selectedCommand = $state<CommandDescriptor | null>(null);
	let editorContent = $state('{}');

	let isSending = $state(false);
	let sendError = $state<string | null>(null);
	let responses = $state<CmdResponseEntry[]>([]);
	let hasResponses = $state(false);

	let fullscreen = $state(false);

	// ─── Derived ──────────────────────────────────────────────────────────────

	const selectedKey = $derived(
		selectedGroupIdx !== null && selectedCommand
			? `${selectedGroupIdx}:${selectedCommand.name}`
			: null
	);

	const descriptorGroups = $derived(
		groups.map((g) => ({
			label: g.ids.map((d) => `${d.name}/${d.namespace}`).join(', '),
			items: g.descriptors,
			errors: g.error ? [g.error] : []
		}))
	);

	const groupErrors = $derived(groups.map((g) => g.error).filter(Boolean) as string[]);

	const selectedGroupDevices = $derived(
		selectedGroupIdx !== null ? (groups[selectedGroupIdx]?.ids ?? []) : []
	);

	const deviceValues = $derived.by(() => {
		if (selectedGroupDevices.length <= 1) return undefined;
		return selectedGroupDevices.map((d) => ({
			label: `${d.name}/${d.namespace}`,
			deviceId: d.id,
			values: editorContent
		}));
	});

	// ─── Startup ──────────────────────────────────────────────────────────────

	$effect(() => {
		if (mirStore.mir) {
			untrack(loadCommands);
		} else {
			groups = [];
			loadError = null;
		}
	});

	async function loadCommands() {
		const mir = mirStore.mir;
		if (!mir) return;

		isLoading = true;
		loadError = null;
		groups = [];
		selectedCommand = null;
		selectedGroupIdx = null;
		responses = [];
		hasResponses = false;

		try {
			const target = new DeviceTarget({
				ids: config.target.ids,
				namespaces: config.target.namespaces,
				labels: config.target.labels
			});
			groups = await mir.client().listCommands().request(target);

			const allDevices = groups.flatMap((g) => g.ids);
			onDevicesReady?.(
				allDevices.map((d, i) => ({
					id: d.id,
					name: d.name,
					color: CHART_COLORS[i % CHART_COLORS.length]
				}))
			);

			// Restore selected command from view state
			if (config.selectedCommand) {
				for (let gi = 0; gi < groups.length; gi++) {
					const desc = groups[gi].descriptors.find((d) => d.name === config.selectedCommand);
					if (desc) {
						selectCommand(gi, desc);
						break;
					}
				}
			}
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load commands';
		} finally {
			isLoading = false;
			hasLoaded = true;
		}
	}

	// ─── Selection ────────────────────────────────────────────────────────────

	function selectCommand(groupIdx: number, desc: CommandDescriptor) {
		selectedGroupIdx = groupIdx;
		selectedCommand = desc;
		try {
			editorContent = JSON.stringify(JSON.parse(desc.template || '{}'), null, 2);
		} catch {
			editorContent = desc.template || '{}';
		}
		isSending = false;
		sendError = null;
		responses = [];
		hasResponses = false;
	}

	// ─── View state ───────────────────────────────────────────────────────────

	$effect(() => {
		if (!hasLoaded) return;
		const name = selectedCommand?.name;
		untrack(() => {
			dashboardStore.saveWidgetViewState(widgetId, {
				...config,
				selectedCommand: name
			});
		});
	});

	// ─── Send helpers ─────────────────────────────────────────────────────────

	function isResponseSuccess(status: CommandResponseStatus): boolean {
		return status === CommandResponseStatus.SUCCESS || status === CommandResponseStatus.VALIDATED;
	}

	function responseMessage(status: CommandResponseStatus, error: string): string {
		if (error) return error;
		return CommandResponseStatus[status] ?? 'OK';
	}

	async function sendToDevice(
		mir: NonNullable<typeof mirStore.mir>,
		deviceId: string,
		deviceName: string,
		text: string,
		dryRun: boolean
	): Promise<CmdResponseEntry> {
		const start = performance.now();
		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.sendCommand()
				.request(target, selectedCommand!.name, text, dryRun);
			const durationMs = Math.round(performance.now() - start);
			const resp = result.get(deviceId) ?? [...result.values()][0];
			const success = resp ? isResponseSuccess(resp.status) : true;
			const message = resp ? responseMessage(resp.status, resp.error) : 'OK';
			activityStore.add({
				kind: success ? 'success' : 'error',
				category: 'Command',
				title: selectedCommand!.name,
				request: { deviceId, name: selectedCommand!.name, text, dryRun },
				...(success ? { response: Object.fromEntries(result) } : { error: message })
			});
			return { deviceId, deviceName, status: success ? 'success' : 'error', message, durationMs, expanded: false };
		} catch (err) {
			const durationMs = Math.round(performance.now() - start);
			const message = err instanceof Error ? err.message : 'Failed';
			activityStore.add({
				kind: 'error',
				category: 'Command',
				title: selectedCommand!.name,
				request: { deviceId, name: selectedCommand!.name, text, dryRun },
				error: message
			});
			return { deviceId, deviceName, status: 'error', message, durationMs, expanded: false };
		}
	}

	// ─── Send (broadcast) ─────────────────────────────────────────────────────

	async function handleSend(dryRun: boolean, text: string) {
		const mir = mirStore.mir;
		if (!mir || !selectedCommand || selectedGroupDevices.length === 0) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		const results = await Promise.allSettled(
			selectedGroupDevices.map((dev) => sendToDevice(mir, dev.id, dev.name, text, dryRun))
		);

		responses = results.map((r) =>
			r.status === 'fulfilled'
				? r.value
				: {
						deviceId: '',
						deviceName: 'unknown',
						status: 'error' as const,
						message: String(r.reason),
						durationMs: 0,
						expanded: false
					}
		);

		isSending = false;
	}

	// ─── Send (per-device) ────────────────────────────────────────────────────

	async function handleSendMulti(dryRun: boolean, payloads: Map<string, string>) {
		const mir = mirStore.mir;
		if (!mir || !selectedCommand) return;

		isSending = true;
		sendError = null;
		responses = [];
		hasResponses = true;

		const results = await Promise.allSettled(
			[...payloads.entries()].map(([deviceId, text]) => {
				const dev =
					selectedGroupDevices.find((d) => d.id === deviceId) ??
					({ id: deviceId, name: deviceId, namespace: 'default' } as const);
				return sendToDevice(mir, dev.id, dev.name, text, dryRun);
			})
		);

		responses = results.map((r) =>
			r.status === 'fulfilled'
				? r.value
				: {
						deviceId: '',
						deviceName: 'unknown',
						status: 'error' as const,
						message: String(r.reason),
						durationMs: 0,
						expanded: false
					}
		);

		isSending = false;
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape' && fullscreen) fullscreen = false;
	}}
/>

<div
	class="{fullscreen
		? 'fixed inset-0 z-50 bg-background'
		: 'h-full'} flex flex-col overflow-hidden"
>
	{#if loadError}
		<p class="px-4 py-2 text-xs text-destructive">{loadError}</p>
	{:else}
		<div class="flex min-h-0 flex-1 overflow-hidden">
			<!-- Left: command list -->
			<DescriptorPanel
				title="Commands"
				items={[]}
				{isLoading}
				error={null}
				{groupErrors}
				groups={descriptorGroups}
				onSelect={() => {}}
				onSelectGrouped={(gi, desc) => selectCommand(gi, desc)}
				selectedKey={selectedKey}
				emptyText="No commands."
			/>

			<!-- Right: editor + responses -->
			<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
				{#if selectedCommand}
					<!-- Fullscreen toggle -->
					<div class="flex shrink-0 items-center justify-end px-2 py-0.5">
						<button
							onclick={() => (fullscreen = !fullscreen)}
							title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
							class="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
						>
							{#if fullscreen}
								<MinimizeIcon class="size-3.5" />
							{:else}
								<MaximizeIcon class="size-3.5" />
							{/if}
						</button>
					</div>

					<!-- JSON editor -->
					<div class="flex min-h-0 flex-1 overflow-hidden">
						<JsonPayloadEditor
							name={selectedCommand.name}
							value={editorContent}
							hasResponse={false}
							{isSending}
							{sendError}
							{deviceValues}
							onSend={handleSend}
							onSendMulti={handleSendMulti}
						/>
					</div>

					<!-- Response log (appears after first send) -->
					{#if hasResponses}
						<div class="max-h-48 shrink-0 overflow-y-auto border-t">
							<div class="border-b px-3 py-1">
								<span
									class="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground"
									>Responses</span
								>
							</div>
							{#each responses as entry (entry.deviceId + entry.deviceName)}
								<div class="border-b last:border-0">
									<button
										class="flex w-full items-center gap-2 px-3 py-1.5 text-left hover:bg-accent/50"
										onclick={() => (entry.expanded = !entry.expanded)}
									>
										{#if entry.status === 'success'}
											<CheckIcon class="size-3 shrink-0 text-emerald-500" />
										{:else}
											<XCircleIcon class="size-3 shrink-0 text-destructive" />
										{/if}
										<span class="min-w-0 flex-1 truncate font-mono text-xs"
											>{entry.deviceName}</span
										>
										<span class="shrink-0 font-mono text-[10px] text-muted-foreground"
											>{entry.durationMs}ms</span
										>
										<span
											class="max-w-32 truncate font-mono text-[10px] text-muted-foreground"
											>{entry.message}</span
										>
									</button>
									{#if entry.expanded}
										<pre
											class="overflow-x-auto bg-muted/40 px-3 py-2 font-mono text-[11px] break-all whitespace-pre-wrap">{entry.message}</pre>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				{:else}
					<p class="p-4 text-xs text-muted-foreground">Select a command</p>
				{/if}
			</div>
		</div>
	{/if}
</div>
```

- [ ] **Step 2: Run type check**

```bash
cd internal/ui/web && npm run check 2>&1 | grep -E "Error|error TS" | grep -v "nav-section\|multi/telemetry"
```

Expected: no new errors. If `CHART_COLORS` import fails (path not found), use the fallback:

```ts
// Replace the CHART_COLORS import + usage with:
const FALLBACK_COLORS = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)', 'var(--chart-4)', 'var(--chart-5)'];
// and in onDevicesReady:
color: FALLBACK_COLORS[i % FALLBACK_COLORS.length]
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/widget-command.svelte
git commit -m "feat(dashboard): rewrite cmd widget with multi-device, per-device send, and response log"
```

---

### Task 4: Smoke test and final type check

**Files:** none (verification only)

- [ ] **Step 1: Full type check**

```bash
cd internal/ui/web && npm run check 2>&1 | tail -5
```

Expected output ends with something like:
```
svelte-check found 0 errors, 3 warnings
```
(The 3 warnings are pre-existing in `nav-section.svelte` and `multi/telemetry`.)

- [ ] **Step 2: Start dev server and manually verify**

```bash
cd internal/ui/web && npm run dev
```

Open the dashboard, add a Command widget targeting ≥1 device that has commands registered. Verify:

1. Command list loads in the left panel (grouped by schema if multiple devices)
2. Device pills appear in the widget header
3. Selecting a command populates the editor with the JSON template
4. Clicking **Send** dispatches to all devices in the group and shows the response log below
5. With multiple devices, clicking **PER DEVICE** in the editor header switches to per-device mode; each device gets its own editable JSON block; clicking **Send** dispatches individual payloads
6. Clicking **Dry Run** sends with `dryRun: true`; responses appear in the log
7. Click a response entry — it expands to show the full message
8. The fullscreen button (top-right of editor area) toggles fullscreen; Escape exits it
9. Reload the page — the previously selected command is restored

- [ ] **Step 3: Commit if any fixes were needed from manual testing**

```bash
git add internal/ui/web/src/lib/domains/dashboards/components/widget-command.svelte
git commit -m "fix(cmd-widget): address issues found during manual smoke test"
```

---

## Self-Review Notes

**Spec coverage:**
- ✅ Multi-device support — `CommandGroup.ids[]` drives all device operations
- ✅ Broadcast mode — `handleSend` fans out same text to all group devices
- ✅ Per-device mode — `handleSendMulti` receives per-device payloads from `JsonPayloadEditor`
- ✅ Response log — `responses: CmdResponseEntry[]`, `hasResponses` flag, expand on click
- ✅ Device pills — `onDevicesReady` prop wired in `dashboard-grid.svelte`
- ✅ View state — `selectedCommand` saved via `saveWidgetViewState`, restored on load
- ✅ Fullscreen — `fixed inset-0 z-50 bg-background`, Escape exits
- ✅ `CommandWidgetConfig.selectedCommand?` added in Task 1
- ✅ `dashboard-grid.svelte` wired in Task 2

**Type consistency:**
- `sendToDevice()` defined in Task 3 script, called in `handleSend` and `handleSendMulti` — same name throughout
- `CmdResponseEntry` defined locally in the widget — no external dependency
- `CommandResponseStatus` imported from `@mir/sdk` — matches enum values used in `isResponseSuccess()`
- `descriptorGroups` items have type `{ label, items: CommandDescriptor[], errors? }` — `CommandDescriptor` is structurally identical to `Descriptor` so `DescriptorPanel` accepts it without cast
