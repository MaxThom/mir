# Multi-Device Bulk Operations Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add checkbox selection to the device list table and dedicated multi-device pages for Telemetry, Commands, and Configuration — letting users operate on multiple devices at once.

**Architecture:** A singleton Svelte 5 `$state` selection store holds selected `Device[]` across navigation. The device table gets a checkbox column + bulk action bar that navigates to new `/devices/multi/*` routes. Those routes group selected devices by their schema (using the `CommandGroup.ids` / `ConfigGroup.ids` / `TelemetryGroup.ids` structure already returned by the SDK) and render per-group operation panels using existing components.

**Tech Stack:** SvelteKit 5, Svelte 5 runes, TanStack Table (already configured for row selection), `@mir/sdk` DeviceTarget (multi-ID already supported by backend), shadcn-svelte, TailwindCSS.

---

## Key Context

All paths relative to `internal/ui/web/src/`.

**How schema grouping works:** The SDK's `ListCommands`, `ListConfigs`, `ListTelemetry` responses already return groups: each `CommandGroup` / `ConfigGroup` / `TelemetryGroup` has `.ids: DeviceId[]` — the set of devices sharing that schema, and `.descriptors[]` — the available operations. Sending a command just needs `DeviceTarget({ ids: group.ids.map(d => d.id) })`.

**How multi-device telemetry overlay works:** Query each device separately in parallel. Merge results by renaming field keys to `{device_name}_{field}` so TlmChart (unchanged) renders them as separate series with different colors from `chartConfig`.

**Existing row selection:** `device-data-table.svelte` already has `rowSelection: {}` in `INITIAL_STATE` but never wires it up. TanStack Table's `getSelectedRowModel()` just needs to be added to the `createTable` call.

---

## Files to Create

| File | Purpose |
|------|---------|
| `lib/domains/devices/stores/selection.svelte.ts` | Singleton store: selected Device[] |
| `routes/devices/multi/+layout.svelte` | Multi-device layout: header chips + tabs |
| `routes/devices/multi/commands/+page.svelte` | Bulk commands page |
| `routes/devices/multi/configuration/+page.svelte` | Bulk configuration page |
| `routes/devices/multi/telemetry/+page.svelte` | Bulk telemetry overlay page |

## Files to Modify

| File | Change |
|------|--------|
| `lib/domains/devices/components/device-table/device-columns.ts` | Add checkbox `select` column |
| `lib/domains/devices/components/device-table/device-data-table.svelte` | Wire row selection + bulk action bar |
| `lib/domains/devices/components/device-table/device-table-toolbar.svelte` | Show selection count |
| `lib/shared/constants/routes.ts` | Add MULTI routes |

---

## Task 1: Selection Store

**Files:**
- Create: `lib/domains/devices/stores/selection.svelte.ts`

**Step 1: Create the store**

```ts
import type { Device } from '@mir/sdk';

class SelectionStore {
	selectedDevices = $state<Device[]>([]);

	select(device: Device) {
		if (!this.isSelected(device.spec.deviceId)) {
			this.selectedDevices = [...this.selectedDevices, device];
		}
	}

	deselect(deviceId: string) {
		this.selectedDevices = this.selectedDevices.filter((d) => d.spec.deviceId !== deviceId);
	}

	setAll(devices: Device[]) {
		this.selectedDevices = devices;
	}

	clearSelection() {
		this.selectedDevices = [];
	}

	isSelected(deviceId: string): boolean {
		return this.selectedDevices.some((d) => d.spec.deviceId === deviceId);
	}

	get count() {
		return this.selectedDevices.length;
	}
}

export const selectionStore = new SelectionStore();
```

**Step 2: Verify it compiles**

```bash
cd internal/ui/web && npx tsc --noEmit
```

Expected: no errors

---

## Task 2: Add Multi Routes to ROUTES Constants

**Files:**
- Modify: `lib/shared/constants/routes.ts`

**Step 1: Add MULTI section**

In `lib/shared/constants/routes.ts`, add after the `DEVICES` block:

```ts
export const ROUTES = {
  // ... existing entries ...
  DEVICES: {
    // ... existing ...
    MULTI: {
      COMMANDS: '/devices/multi/commands',
      CONFIG: '/devices/multi/configuration',
      TELEMETRY: '/devices/multi/telemetry',
    }
  },
  // ...
} as const;
```

---

## Task 3: Checkbox Column

**Files:**
- Modify: `lib/domains/devices/components/device-table/device-columns.ts`

**Step 1: Add select column as first column**

```ts
import { createColumnHelper, type FilterFn, type Row } from '@tanstack/table-core';
import type { Device } from '@mir/sdk';

// ... existing deviceGlobalFilterFn ...

const col = createColumnHelper<Device>();

export const deviceColumns = [
	col.display({
		id: 'select',
		header: '',       // header checkbox rendered in data-table.svelte
		enableSorting: false,
		enableGlobalFilter: false,
	}),
	// ... rest of existing columns unchanged ...
];
```

The actual checkbox rendering is done via `DeviceTableCell` (which already has a switch on `cell.column.id`) — the cell logic goes in the data table, not here.

---

## Task 4: Wire Row Selection in Data Table

**Files:**
- Modify: `lib/domains/devices/components/device-table/device-data-table.svelte`

**Step 1: Import selection store and goto**

Add to existing imports:

```ts
import { goto } from '$app/navigation';
import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
import { getSelectedRowModel } from '@tanstack/table-core';
import CheckboxIcon from '$lib/shared/components/shadcn/checkbox';  // or Checkbox from bits-ui
import { Checkbox } from '$lib/shared/components/shadcn/checkbox';
import LayersIcon from '@lucide/svelte/icons/layers';
import ActivityIcon from '@lucide/svelte/icons/activity';
import TerminalIcon from '@lucide/svelte/icons/terminal';
import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
import XIcon from '@lucide/svelte/icons/x';
```

**Step 2: Add rowSelection state**

```ts
let rowSelection = $state<Record<string, boolean>>({});
```

**Step 3: Add getSelectedRowModel and onRowSelectionChange to options**

In the `options` `$derived`, add:

```ts
let options = $derived<TableOptionsResolved<Device>>({
    data: devices,
    columns: deviceColumns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getSelectedRowModel: getSelectedRowModel(),    // ADD
    enableRowSelection: true,                       // ADD
    globalFilterFn: deviceGlobalFilterFn,
    onSortingChange: (updater: Updater<SortingState>) => {
        sorting = typeof updater === 'function' ? updater(sorting) : updater;
    },
    onGlobalFilterChange: (updater: Updater<string>) => {
        globalFilter = typeof updater === 'function' ? updater(globalFilter) : updater;
        pagination = { ...pagination, pageIndex: 0 };
    },
    onPaginationChange: (updater: Updater<PaginationState>) => {
        pagination = typeof updater === 'function' ? updater(pagination) : updater;
    },
    onRowSelectionChange: (updater: Updater<Record<string, boolean>>) => {   // ADD
        rowSelection = typeof updater === 'function' ? updater(rowSelection) : updater;
    },
    onStateChange() {},
    renderFallbackValue: null,
    state: { ...INITIAL_STATE, sorting, globalFilter, pagination, rowSelection }  // ADD rowSelection
});
```

**Step 4: Sync table selection → selectionStore**

```ts
$effect(() => {
    const selected = table.getSelectedRowModel().rows.map((r) => r.original);
    selectionStore.setAll(selected);
});
```

**Step 5: Restore selection on mount (when devices list is refreshed)**

```ts
$effect(() => {
    // When devices data changes, re-apply selection from store
    const selectedIds = new Set(selectionStore.selectedDevices.map((d) => d.spec.deviceId));
    const newSelection: Record<string, boolean> = {};
    table.getCoreRowModel().rows.forEach((row, i) => {
        if (selectedIds.has(row.original.spec.deviceId)) {
            newSelection[String(i)] = true;
        }
    });
    rowSelection = newSelection;
});
```

**Step 6: Render select column header (select-all checkbox)**

In the `Table.Header` section, add handling for the `select` column id:

```svelte
{#each headerGroup.headers as header, i (i)}
    <Table.Head class={cn(
        'h-10 text-xs font-medium tracking-wide text-muted-foreground uppercase',
        header.column.id === 'select' && 'w-px px-3',
        header.column.id === 'actions' && 'w-px whitespace-nowrap'
    )}>
        {#if header.column.id === 'select'}
            <Checkbox
                checked={table.getIsAllPageRowsSelected()}
                indeterminate={table.getIsSomePageRowsSelected()}
                onCheckedChange={(v) => table.toggleAllPageRowsSelected(!!v)}
                aria-label="Select all"
            />
        {:else if !header.isPlaceholder}
            <!-- existing sort button logic -->
        {/if}
    </Table.Head>
{/each}
```

**Step 7: Render select column cell (per-row checkbox)**

In `Table.Body`, replace `<DeviceTableCell {cell} {row} />` with:

```svelte
{#each row.getVisibleCells() as cell, j (j)}
    {#if cell.column.id === 'select'}
        <Table.Cell class="w-px px-3">
            <Checkbox
                checked={row.getIsSelected()}
                onCheckedChange={(v) => row.toggleSelected(!!v)}
                aria-label="Select row"
            />
        </Table.Cell>
    {:else}
        <DeviceTableCell {cell} {row} />
    {/if}
{/each}
```

**Step 8: Add bulk action bar below the table (inside `<Tooltip.Provider>`)**

Add after `<DeviceTablePagination ... />`:

```svelte
{#if selectionStore.count > 0}
    <div class="flex items-center justify-between border-t bg-muted/50 px-4 py-2">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <LayersIcon class="size-4" />
            <span>{selectionStore.count} device{selectionStore.count > 1 ? 's' : ''} selected</span>
        </div>
        <div class="flex items-center gap-2">
            <button
                onclick={() => { selectionStore.clearSelection(); rowSelection = {}; }}
                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
            >
                <XIcon class="size-3.5" />
                Clear
            </button>
            <button
                onclick={() => goto('/devices/multi/telemetry')}
                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
            >
                <ActivityIcon class="size-3.5" />
                Telemetry
            </button>
            <button
                onclick={() => goto('/devices/multi/commands')}
                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
            >
                <TerminalIcon class="size-3.5" />
                Commands
            </button>
            <button
                onclick={() => goto('/devices/multi/configuration')}
                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors hover:bg-accent hover:text-accent-foreground"
            >
                <SlidersHorizontalIcon class="size-3.5" />
                Configuration
            </button>
        </div>
    </div>
{/if}
```

**Step 9: Also update the empty colspan to account for extra column**

Change `colspan={deviceColumns.length}` — no change needed, `deviceColumns.length` now includes the select column automatically.

**Step 10: Verify in browser**

Run: `cd internal/ui/web && npm run dev`

Open device list, check:
- [ ] Checkbox column appears
- [ ] Individual row checkboxes work
- [ ] Select-all header checkbox works
- [ ] Bulk action bar appears/disappears correctly
- [ ] Navigate to `/devices/multi/commands` from bulk bar

**Step 11: Commit**

```bash
git add internal/ui/web/src/lib/domains/devices/stores/selection.svelte.ts \
        internal/ui/web/src/lib/domains/devices/components/device-table/device-columns.ts \
        internal/ui/web/src/lib/domains/devices/components/device-table/device-data-table.svelte \
        internal/ui/web/src/lib/shared/constants/routes.ts
git commit -m "feat(cockpit): add multi-device checkbox selection and bulk action bar"
```

---

## Task 5: Update Toolbar to Show Selection Count

**Files:**
- Modify: `lib/domains/devices/components/device-table/device-table-toolbar.svelte`

**Step 1: Import selection store and show count**

```svelte
<script lang="ts">
    import SearchIcon from '@lucide/svelte/icons/search';
    import { Input } from '$lib/shared/components/shadcn/input';
    import { Badge } from '$lib/shared/components/shadcn/badge';
    import { RefreshButtonGroup } from '$lib/shared/components/ui/refresh-button-group';
    import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';

    let {
        deviceCount,
        onlineCount,
        globalFilter,
        isLoading = false,
        onRefresh,
        onglobalfilterchange
    }: { ... } = $props();
</script>

<div class="flex items-center justify-between border-b px-6 py-4">
    <div class="flex items-center gap-3">
        <span class="text-sm font-semibold">Devices</span>
        <Badge variant="secondary" class="tabular-nums">{deviceCount}</Badge>
        {#if selectionStore.count > 0}
            <Badge variant="outline" class="tabular-nums text-muted-foreground">
                {selectionStore.count} selected
            </Badge>
        {/if}
        <!-- existing search input -->
    </div>
    <!-- existing right side -->
</div>
```

---

## Task 6: Multi-Device Layout

**Files:**
- Create: `routes/devices/multi/+layout.svelte`

**Step 1: Create layout**

```svelte
<script lang="ts">
    import { goto } from '$app/navigation';
    import { page } from '$app/state';
    import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
    import { ROUTES } from '$lib/shared/constants/routes';
    import { Badge } from '$lib/shared/components/shadcn/badge';
    import { cn } from '$lib/utils';
    import XIcon from '@lucide/svelte/icons/x';
    import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';

    let { children } = $props();

    // Guard: redirect to device list if no selection
    $effect(() => {
        if (selectionStore.count === 0) {
            goto(ROUTES.DEVICES.LIST);
        }
    });

    const TABS = [
        { label: 'Telemetry', href: ROUTES.DEVICES.MULTI.TELEMETRY },
        { label: 'Commands', href: ROUTES.DEVICES.MULTI.COMMANDS },
        { label: 'Configuration', href: ROUTES.DEVICES.MULTI.CONFIG },
    ];

    let isActive = (href: string) => page.url.pathname === href;
</script>

<div class="flex flex-col">
    <!-- Header -->
    <div class="border-b bg-background px-4 pt-2 pb-0">
        <!-- Breadcrumb -->
        <div class="flex items-center gap-2 pb-2">
            <button
                onclick={() => goto(ROUTES.DEVICES.LIST)}
                class="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
                <ChevronLeftIcon class="size-3.5" />
                Devices
            </button>
            <span class="text-xs text-muted-foreground">/</span>
            <span class="text-xs font-medium">Bulk Operations</span>
            <Badge variant="secondary" class="ml-1 text-xs">{selectionStore.count}</Badge>
        </div>

        <!-- Selected device chips -->
        <div class="flex flex-wrap items-center gap-1.5 pb-2">
            {#each selectionStore.selectedDevices as device (device.spec.deviceId)}
                <span class="flex items-center gap-1 rounded-full border bg-muted/50 px-2 py-0.5 font-mono text-xs">
                    <span class={cn(
                        'h-1.5 w-1.5 shrink-0 rounded-full',
                        device.status?.online ? 'bg-emerald-500' : 'bg-muted-foreground/30'
                    )}></span>
                    {device.meta?.name}/{device.meta?.namespace}
                    <button
                        onclick={() => selectionStore.deselect(device.spec.deviceId)}
                        class="ml-0.5 text-muted-foreground transition-colors hover:text-foreground"
                    >
                        <XIcon class="size-3" />
                    </button>
                </span>
            {/each}
        </div>

        <!-- Tab navigation -->
        <nav class="-mb-px flex gap-0">
            {#each TABS as tab (tab.label)}
                <a
                    href={tab.href}
                    class={cn(
                        'border-b-2 px-3 py-2 text-sm font-medium transition-colors',
                        isActive(tab.href)
                            ? 'border-primary text-foreground'
                            : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground'
                    )}
                >
                    {tab.label}
                </a>
            {/each}
        </nav>
    </div>

    <!-- Tab content -->
    <div class="flex-1 p-4">
        {@render children()}
    </div>
</div>
```

**Step 2: Commit**

```bash
git add internal/ui/web/src/routes/devices/multi/
git commit -m "feat(cockpit): add multi-device layout with breadcrumb and device chips"
```

---

## Task 7: Multi-Device Commands Page

**Files:**
- Create: `routes/devices/multi/commands/+page.svelte`

**Step 1: Create the page**

The key insight: call `listCommands` with all selected device IDs in one request. The response comes back grouped by schema (each `CommandGroup` = one schema). Then send uses the same group's IDs as DeviceTarget.

```svelte
<script lang="ts">
    import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
    import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
    import { DeviceTarget, CommandResponseStatus } from '@mir/sdk';
    import type { CommandGroup, CommandDescriptor, SendCommandResult } from '@mir/sdk';
    import {
        DescriptorPanel,
        JsonPayloadEditor,
        ResponsePanel
    } from '$lib/domains/devices/components/commands';
    import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
    import TerminalIcon from '@lucide/svelte/icons/terminal';
    import { Separator } from '$lib/shared/components/shadcn/separator';
    import { Spinner } from '$lib/shared/components/shadcn/spinner';
    import type { Descriptor } from '$lib/domains/devices/types/types';

    // ─── State per schema group ───────────────────────────────────────────────

    type GroupState = {
        group: CommandGroup;
        selectedCommand: CommandDescriptor | null;
        editorContent: string;
        isSending: boolean;
        sendError: string | null;
        response: SendCommandResult | null;
    };

    let commandGroups = $state<CommandGroup[]>([]);
    let isLoading = $state(false);
    let error = $state<string | null>(null);
    let groupStates = $state<GroupState[]>([]);

    // ─── Load on mount / when selection changes ───────────────────────────────

    $effect(() => {
        if (!mirStore.mir || selectionStore.count === 0) return;
        loadCommands();
    });

    async function loadCommands() {
        if (!mirStore.mir) return;
        isLoading = true;
        error = null;
        try {
            const allIds = selectionStore.selectedDevices.map((d) => d.spec.deviceId);
            const target = new DeviceTarget({ ids: allIds });
            const groups = await mirStore.mir.client().listCommands().request(target);
            commandGroups = groups;
            // Initialize per-group state
            groupStates = groups.map((g) => ({
                group: g,
                selectedCommand: null,
                editorContent: '{}',
                isSending: false,
                sendError: null,
                response: null,
            }));
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load commands';
        } finally {
            isLoading = false;
        }
    }

    // ─── Helpers ──────────────────────────────────────────────────────────────

    function prettyJson(raw: string): string {
        try { return JSON.stringify(JSON.parse(raw), null, 2); } catch { return raw; }
    }

    function statusLabel(status: number): string {
        switch (status) {
            case CommandResponseStatus.SUCCESS: return 'SUCCESS';
            case CommandResponseStatus.ERROR: return 'ERROR';
            case CommandResponseStatus.VALIDATED: return 'VALIDATED';
            case CommandResponseStatus.PENDING: return 'PENDING';
            default: return 'UNKNOWN';
        }
    }

    function statusClass(status: number): string {
        switch (status) {
            case CommandResponseStatus.SUCCESS: return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400';
            case CommandResponseStatus.ERROR: return 'bg-destructive/15 text-destructive';
            case CommandResponseStatus.VALIDATED: return 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400';
            default: return 'bg-muted text-muted-foreground';
        }
    }

    // ─── Per-group actions ────────────────────────────────────────────────────

    function selectCommand(groupIdx: number, desc: Descriptor) {
        const gs = groupStates[groupIdx];
        const fullDesc = gs.group.descriptors.find((d) => d.name === desc.name);
        if (!fullDesc) return;
        groupStates[groupIdx] = {
            ...gs,
            selectedCommand: fullDesc,
            editorContent: prettyJson(fullDesc.template || '{}'),
            response: null,
            sendError: null,
        };
    }

    async function sendCommand(groupIdx: number, dryRun: boolean, text: string) {
        if (!mirStore.mir) return;
        const gs = groupStates[groupIdx];
        if (!gs.selectedCommand) return;
        groupStates[groupIdx] = { ...gs, isSending: true, sendError: null };
        try {
            const ids = gs.group.ids.map((id) => id.id);
            const target = new DeviceTarget({ ids });
            const result = await mirStore.mir.client().sendCommand().request(target, gs.selectedCommand.name, text, dryRun);
            groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, response: result };
            activityStore.add({
                kind: 'success',
                category: 'Command',
                title: gs.selectedCommand.name,
                request: { ids, name: gs.selectedCommand.name, payload: text, dryRun },
                response: Object.fromEntries(result),
            });
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to send';
            groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, sendError: message };
            activityStore.add({
                kind: 'error',
                category: 'Command',
                title: gs.selectedCommand.name,
                request: { name: gs.selectedCommand.name },
                error: message,
            });
        }
    }
</script>

<div class="flex flex-col gap-6">
    {#if isLoading}
        <div class="flex items-center justify-center py-12 text-muted-foreground">
            <Spinner class="mr-2 size-4" />
            Loading commands...
        </div>
    {:else if error}
        <p class="text-sm text-destructive">{error}</p>
    {:else if commandGroups.length === 0}
        <div class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
            <TerminalIcon class="size-8 opacity-30" />
            <p class="text-sm">No commands found for selected devices</p>
        </div>
    {:else}
        {#each groupStates as gs, idx (idx)}
            <!-- Schema group header -->
            <div>
                <div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <span class="font-mono font-medium text-foreground">
                        {gs.group.ids.map((id) => `${id.name}/${id.namespace}`).join(', ')}
                    </span>
                    <span class="text-muted-foreground">({gs.group.ids.length} device{gs.group.ids.length > 1 ? 's' : ''})</span>
                </div>

                <!-- Three-panel layout (same as single-device) -->
                <div class="flex min-h-80 overflow-hidden rounded-lg border">
                    <DescriptorPanel
                        title="Commands"
                        items={gs.group.descriptors.map((d) => ({
                            name: d.name,
                            labels: d.labels,
                            template: d.template,
                            error: d.error,
                        }))}
                        isLoading={false}
                        error={null}
                        groupErrors={gs.group.error ? [gs.group.error] : []}
                        selectedName={gs.selectedCommand?.name ?? null}
                        emptyText="No commands."
                        onSelect={(desc) => selectCommand(idx, desc)}
                    />

                    <div class="flex min-w-0 flex-1 overflow-hidden">
                        {#if !gs.selectedCommand}
                            <div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
                                <TerminalIcon class="size-8 opacity-30" />
                                <p class="text-sm">Select a command</p>
                            </div>
                        {:else}
                            <JsonPayloadEditor
                                name={gs.selectedCommand.name}
                                nameError={gs.selectedCommand.error}
                                value={gs.editorContent}
                                hasResponse={true}
                                isSending={gs.isSending}
                                sendError={gs.sendError}
                                onSend={(dryRun, text) => sendCommand(idx, dryRun, text)}
                            />
                            <ResponsePanel
                                response={gs.response}
                                {statusLabel}
                                {statusClass}
                                onClear={() => {
                                    groupStates[idx] = { ...groupStates[idx], response: null, sendError: null };
                                }}
                            />
                        {/if}
                    </div>
                </div>
            </div>

            {#if idx < groupStates.length - 1}
                <Separator />
            {/if}
        {/each}
    {/if}
</div>
```

**Step 2: Verify in browser**

- Select 2+ devices, click Commands
- Verify schema groups appear
- Select and send a command to one group
- Verify per-device responses shown in ResponsePanel

**Step 3: Commit**

```bash
git add internal/ui/web/src/routes/devices/multi/commands/+page.svelte
git commit -m "feat(cockpit): add multi-device bulk commands page"
```

---

## Task 8: Multi-Device Configuration Page

**Files:**
- Create: `routes/devices/multi/configuration/+page.svelte`

**Step 1: Create the page**

Identical structure to commands page, using config SDK calls.

```svelte
<script lang="ts">
    import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
    import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
    import { DeviceTarget, ConfigResponseStatus } from '@mir/sdk';
    import type { ConfigGroup, ConfigDescriptor, SendConfigResult } from '@mir/sdk';
    import {
        DescriptorPanel,
        ResponsePanel
    } from '$lib/domains/devices/components/commands';
    import { CfgPayloadEditor } from '$lib/domains/devices/components/configurations';
    import { activityStore } from '$lib/domains/activity/stores/activity.svelte';
    import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
    import { Separator } from '$lib/shared/components/shadcn/separator';
    import { Spinner } from '$lib/shared/components/shadcn/spinner';
    import type { Descriptor } from '$lib/domains/devices/types/types';

    type GroupState = {
        group: ConfigGroup;
        selectedConfig: ConfigDescriptor | null;
        isSending: boolean;
        sendError: string | null;
        response: SendConfigResult | null;
    };

    let configGroups = $state<ConfigGroup[]>([]);
    let isLoading = $state(false);
    let error = $state<string | null>(null);
    let groupStates = $state<GroupState[]>([]);

    $effect(() => {
        if (!mirStore.mir || selectionStore.count === 0) return;
        loadConfigs();
    });

    async function loadConfigs() {
        if (!mirStore.mir) return;
        isLoading = true;
        error = null;
        try {
            const allIds = selectionStore.selectedDevices.map((d) => d.spec.deviceId);
            const target = new DeviceTarget({ ids: allIds });
            const groups = await mirStore.mir.client().listConfigs().request(target);
            configGroups = groups;
            groupStates = groups.map((g) => ({
                group: g,
                selectedConfig: null,
                isSending: false,
                sendError: null,
                response: null,
            }));
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load configs';
        } finally {
            isLoading = false;
        }
    }

    function prettyJson(raw: string): string {
        try { return JSON.stringify(JSON.parse(raw), null, 2); } catch { return raw; }
    }

    function statusLabel(status: number): string {
        switch (status) {
            case ConfigResponseStatus.SUCCESS: return 'SUCCESS';
            case ConfigResponseStatus.ERROR: return 'ERROR';
            case ConfigResponseStatus.VALIDATED: return 'VALIDATED';
            case ConfigResponseStatus.NOCHANGE: return 'NOCHANGE';
            case ConfigResponseStatus.PENDING: return 'PENDING';
            default: return 'UNKNOWN';
        }
    }

    function statusClass(status: number): string {
        switch (status) {
            case ConfigResponseStatus.SUCCESS: return 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400';
            case ConfigResponseStatus.ERROR: return 'bg-destructive/15 text-destructive';
            case ConfigResponseStatus.VALIDATED:
            case ConfigResponseStatus.NOCHANGE:
                return 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400';
            default: return 'bg-muted text-muted-foreground';
        }
    }

    function selectConfig(groupIdx: number, desc: Descriptor) {
        const gs = groupStates[groupIdx];
        const fullDesc = gs.group.descriptors.find((d) => d.name === desc.name);
        if (!fullDesc) return;
        groupStates[groupIdx] = { ...gs, selectedConfig: fullDesc, response: null, sendError: null };
    }

    // Current values for a config: look up from group.values for first device in group
    function getCurrentValues(gs: GroupState): string {
        if (!gs.selectedConfig) return '{}';
        const deviceId = gs.group.ids[0]?.id ?? '';
        const dv = gs.group.values?.find((v: { deviceId: string }) => v.deviceId === deviceId);
        return prettyJson(dv?.values?.[gs.selectedConfig.name] ?? '{}');
    }

    async function sendConfig(groupIdx: number, dryRun: boolean, text: string) {
        if (!mirStore.mir) return;
        const gs = groupStates[groupIdx];
        if (!gs.selectedConfig) return;
        groupStates[groupIdx] = { ...gs, isSending: true, sendError: null };
        try {
            const ids = gs.group.ids.map((id) => id.id);
            const target = new DeviceTarget({ ids });
            const result = await mirStore.mir.client().sendConfig().request(target, gs.selectedConfig.name, text, dryRun);
            groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, response: result };
            activityStore.add({
                kind: 'success',
                category: 'Config',
                title: gs.selectedConfig.name,
                request: { ids, name: gs.selectedConfig.name, payload: text, dryRun },
                response: Object.fromEntries(result),
            });
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to send';
            groupStates[groupIdx] = { ...groupStates[groupIdx], isSending: false, sendError: message };
            activityStore.add({
                kind: 'error',
                category: 'Config',
                title: gs.selectedConfig.name,
                request: { name: gs.selectedConfig.name },
                error: message,
            });
        }
    }
</script>

<div class="flex flex-col gap-6">
    {#if isLoading}
        <div class="flex items-center justify-center py-12 text-muted-foreground">
            <Spinner class="mr-2 size-4" />
            Loading configurations...
        </div>
    {:else if error}
        <p class="text-sm text-destructive">{error}</p>
    {:else if configGroups.length === 0}
        <div class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
            <SlidersHorizontalIcon class="size-8 opacity-30" />
            <p class="text-sm">No configurations found for selected devices</p>
        </div>
    {:else}
        {#each groupStates as gs, idx (idx)}
            <div>
                <div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <span class="font-mono font-medium text-foreground">
                        {gs.group.ids.map((id) => `${id.name}/${id.namespace}`).join(', ')}
                    </span>
                    <span>({gs.group.ids.length} device{gs.group.ids.length > 1 ? 's' : ''})</span>
                </div>

                <div class="flex min-h-80 overflow-hidden rounded-lg border">
                    <DescriptorPanel
                        title="Configurations"
                        items={gs.group.descriptors.map((d) => ({
                            name: d.name,
                            labels: d.labels,
                            template: d.template,
                            error: d.error,
                        }))}
                        isLoading={false}
                        error={null}
                        groupErrors={gs.group.error ? [gs.group.error] : []}
                        selectedName={gs.selectedConfig?.name ?? null}
                        emptyText="No configurations."
                        onSelect={(desc) => selectConfig(idx, desc)}
                    />

                    <div class="flex min-w-0 flex-1 overflow-hidden">
                        {#if !gs.selectedConfig}
                            <div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
                                <SlidersHorizontalIcon class="size-8 opacity-30" />
                                <p class="text-sm">Select a configuration</p>
                            </div>
                        {:else}
                            <CfgPayloadEditor
                                name={gs.selectedConfig.name}
                                nameError={gs.selectedConfig.error}
                                currentValues={getCurrentValues(gs)}
                                template={prettyJson(gs.selectedConfig?.template ?? '{}')}
                                hasResponse={true}
                                isSending={gs.isSending}
                                sendError={gs.sendError}
                                onSend={(dryRun, text) => sendConfig(idx, dryRun, text)}
                            />
                            <ResponsePanel
                                response={gs.response}
                                {statusLabel}
                                {statusClass}
                                onClear={() => {
                                    groupStates[idx] = { ...groupStates[idx], response: null, sendError: null };
                                }}
                            />
                        {/if}
                    </div>
                </div>
            </div>

            {#if idx < groupStates.length - 1}
                <Separator />
            {/if}
        {/each}
    {/if}
</div>
```

**Step 2: Commit**

```bash
git add internal/ui/web/src/routes/devices/multi/configuration/+page.svelte
git commit -m "feat(cockpit): add multi-device bulk configuration page"
```

---

## Task 9: Multi-Device Telemetry Page

**Files:**
- Create: `routes/devices/multi/telemetry/+page.svelte`

**Strategy:** Query each device individually in parallel, then merge results by prefixing field keys with the device name (e.g. `power_temperature`, `weather_temperature`). TlmChart accepts this unchanged — each field becomes a separate series, colored by `chartConfig`.

```svelte
<script lang="ts">
    import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
    import { selectionStore } from '$lib/domains/devices/stores/selection.svelte';
    import { DeviceTarget } from '@mir/sdk';
    import type { TelemetryGroup, TelemetryDescriptor, QueryData, QueryRow } from '@mir/sdk';
    import DescriptorPanel from '$lib/domains/devices/components/commands/descriptor-panel.svelte';
    import TlmChart from '$lib/domains/devices/components/telemetry/tlm-chart.svelte';
    import TimePicker from '$lib/domains/devices/components/telemetry/time-picker.svelte';
    import { RangeCalendar } from '$lib/shared/components/shadcn/range-calendar';
    import * as Popover from '$lib/shared/components/shadcn/popover';
    import { Separator } from '$lib/shared/components/shadcn/separator';
    import { Spinner } from '$lib/shared/components/shadcn/spinner';
    import ActivityIcon from '@lucide/svelte/icons/activity';
    import CalendarIcon from '@lucide/svelte/icons/calendar';
    import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
    import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';
    import type { ChartConfig } from '$lib/shared/components/shadcn/chart';
    import type { Descriptor } from '$lib/domains/devices/types/types';
    import type { DateRange } from 'bits-ui';
    import { getLocalTimeZone, fromDate } from '@internationalized/date';
    import { SvelteDate } from 'svelte/reactivity';

    // ─── Constants ────────────────────────────────────────────────────────────

    const CHART_COLORS = [
        'var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)',
        'var(--chart-4)', 'var(--chart-5)'
    ];

    const PRESETS = [
        { label: '5m', minutes: 5 }, { label: '15m', minutes: 15 },
        { label: '30m', minutes: 30 }, { label: '1h', minutes: 60 },
        { label: '3h', minutes: 180 }, { label: '6h', minutes: 360 },
        { label: '24h', minutes: 1440 }, { label: '7d', minutes: 10080 },
    ] as const;

    type TimeFilter = { mode: 'relative'; minutes: number } | { mode: 'absolute'; start: Date; end: Date };

    // ─── Schema group state ───────────────────────────────────────────────────

    type GroupState = {
        group: TelemetryGroup;
        selectedMeasurement: TelemetryDescriptor | null;
        mergedData: QueryData | null;
        isQuerying: boolean;
        queryError: string | null;
        chartConfig: ChartConfig;
        mergedFields: string[];  // prefixed field keys like "power_temperature"
    };

    let tlmGroups = $state<TelemetryGroup[]>([]);
    let isLoading = $state(false);
    let error = $state<string | null>(null);
    let groupStates = $state<GroupState[]>([]);

    // Shared time filter across all groups
    let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 5 });
    let popoverOpen = $state(false);
    let calendarValue = $state<DateRange | undefined>(undefined);
    let startTime = $state('00:00');
    let endTime = $state('23:59');

    // ─── Load measurements ────────────────────────────────────────────────────

    $effect(() => {
        if (!mirStore.mir || selectionStore.count === 0) return;
        loadMeasurements();
    });

    async function loadMeasurements() {
        if (!mirStore.mir) return;
        isLoading = true;
        error = null;
        try {
            const allIds = selectionStore.selectedDevices.map((d) => d.spec.deviceId);
            const target = new DeviceTarget({ ids: allIds });
            const groups = await mirStore.mir.client().listTelemetry().request(target);
            tlmGroups = groups;
            groupStates = groups.map((g) => ({
                group: g,
                selectedMeasurement: null,
                mergedData: null,
                isQuerying: false,
                queryError: null,
                chartConfig: {},
                mergedFields: [],
            }));
        } catch (err) {
            error = err instanceof Error ? err.message : 'Failed to load telemetry';
        } finally {
            isLoading = false;
        }
    }

    // ─── Time helpers ─────────────────────────────────────────────────────────

    function getTimeRange(): { start: Date; end: Date } {
        if (timeFilter.mode === 'absolute') {
            const start = timeFilter.start;
            const end = timeFilter.end.getTime() <= start.getTime()
                ? new Date(start.getTime() + 1000) : timeFilter.end;
            return { start, end };
        }
        const end = new Date();
        const start = new Date(end.getTime() - timeFilter.minutes * 60 * 1000);
        return { start, end };
    }

    function getAggregationWindow(start: Date, end: Date): string | undefined {
        const hours = (end.getTime() - start.getTime()) / (1000 * 60 * 60);
        if (hours < 1) return undefined;
        if (hours < 6) return '10s';
        if (hours < 24) return '1m';
        if (hours < 168) return '10m';
        if (hours < 720) return '1h';
        return '6h';
    }

    let timeFilterLabel = $derived.by(() => {
        if (timeFilter.mode === 'relative') return `Last ${PRESETS.find(p => p.minutes === timeFilter.minutes)?.label ?? timeFilter.minutes + 'm'}`;
        const fmt = (d: Date) => d.toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
        return `${fmt(timeFilter.start)} – ${fmt(timeFilter.end)}`;
    });

    // ─── Query and merge ──────────────────────────────────────────────────────

    async function queryGroup(groupIdx: number) {
        if (!mirStore.mir) return;
        const gs = groupStates[groupIdx];
        if (!gs.selectedMeasurement) return;

        groupStates[groupIdx] = { ...gs, isQuerying: true, queryError: null };

        try {
            const { start, end } = getTimeRange();
            const aggWindow = getAggregationWindow(start, end);
            const fields = gs.selectedMeasurement.fields;

            // Query each device in the group in parallel
            const results = await Promise.all(
                gs.group.ids.map((devId) =>
                    mirStore.mir!.client().queryTelemetry().request(
                        new DeviceTarget({ ids: [devId.id] }),
                        gs.selectedMeasurement!.name,
                        fields,
                        start,
                        end,
                        aggWindow
                    ).then((data) => ({ deviceId: devId.id, deviceName: devId.name, data }))
                )
            );

            // Merge: combine all rows by _time, prefixing field keys with device name
            // Build merged QueryData: headers = [_time, devA_field1, devA_field2, devB_field1, ...]
            const mergedFields: string[] = [];
            const newChartConfig: ChartConfig = {};
            let colorIdx = 0;

            results.forEach(({ deviceName, data }) => {
                data.headers.filter(h => h !== '_time').forEach((field) => {
                    const key = `${deviceName}_${field}`;
                    mergedFields.push(key);
                    newChartConfig[key] = {
                        label: `${deviceName}: ${field}`,
                        color: CHART_COLORS[colorIdx % CHART_COLORS.length],
                    };
                    colorIdx++;
                });
            });

            // Build unified time-indexed rows
            // Collect all unique timestamps
            const timeMap = new Map<number, QueryRow>();
            results.forEach(({ deviceName, data }) => {
                data.rows.forEach((row) => {
                    const t = row.values['_time'] as Date;
                    const key = t instanceof Date ? t.getTime() : 0;
                    if (!timeMap.has(key)) {
                        timeMap.set(key, { values: { _time: t } });
                    }
                    const mergedRow = timeMap.get(key)!;
                    data.headers.filter(h => h !== '_time').forEach((field) => {
                        mergedRow.values[`${deviceName}_${field}`] = row.values[field];
                    });
                });
            });

            const sortedRows = Array.from(timeMap.values()).sort((a, b) => {
                const ta = a.values['_time'] as Date;
                const tb = b.values['_time'] as Date;
                return ta.getTime() - tb.getTime();
            });

            const mergedData: QueryData = {
                headers: ['_time', ...mergedFields],
                rows: sortedRows,
            };

            groupStates[groupIdx] = {
                ...groupStates[groupIdx],
                isQuerying: false,
                mergedData,
                mergedFields,
                chartConfig: newChartConfig,
            };
        } catch (err) {
            groupStates[groupIdx] = {
                ...groupStates[groupIdx],
                isQuerying: false,
                queryError: err instanceof Error ? err.message : 'Query failed',
            };
        }
    }

    function selectMeasurement(groupIdx: number, desc: Descriptor) {
        const gs = groupStates[groupIdx];
        const full = gs.group.descriptors.find((d) => d.name === desc.name);
        if (!full) return;
        groupStates[groupIdx] = { ...gs, selectedMeasurement: full, mergedData: null, queryError: null };
        queryGroup(groupIdx);
    }

    function selectPreset(minutes: number) {
        timeFilter = { mode: 'relative', minutes };
        popoverOpen = false;
        groupStates.forEach((_, idx) => {
            if (groupStates[idx].selectedMeasurement) queryGroup(idx);
        });
    }

    $effect(() => {
        if (timeFilter.mode === 'absolute') {
            const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
            calendarValue = { start: fromDate(timeFilter.start, tz), end: fromDate(timeFilter.end, tz) };
        }
    });
</script>

<div class="flex flex-col gap-6">
    <!-- Shared time picker -->
    <div class="flex items-center justify-end">
        <Popover.Root bind:open={popoverOpen}>
            <Popover.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        class="flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1 text-xs font-medium text-foreground shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                    >
                        <CalendarIcon class="size-3.5 text-muted-foreground" />
                        <span>{timeFilterLabel}</span>
                        <ChevronDownIcon class="size-3 text-muted-foreground" />
                    </button>
                {/snippet}
            </Popover.Trigger>
            <Popover.Content class="w-auto p-0 shadow-lg" align="end">
                <div class="flex">
                    <div class="p-5">
                        <RangeCalendar
                            bind:value={calendarValue}
                            onValueChange={(v) => {
                                if (v?.start && v?.end) {
                                    const tz = editorPrefs.utc ? 'UTC' : getLocalTimeZone();
                                    timeFilter = { mode: 'absolute', start: v.start.toDate(tz), end: v.end.toDate(tz) };
                                    groupStates.forEach((_, idx) => { if (groupStates[idx].selectedMeasurement) queryGroup(idx); });
                                }
                            }}
                            numberOfMonths={1}
                        />
                    </div>
                    <div class="w-px self-stretch bg-border"></div>
                    <div class="relative w-36 self-stretch">
                        <div class="absolute inset-0 flex flex-col p-3">
                            <p class="mb-2 px-2 text-xs font-semibold tracking-wider text-muted-foreground uppercase">Quick range</p>
                            <div class="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto">
                                {#each PRESETS as preset (preset.label)}
                                    <button
                                        onclick={() => selectPreset(preset.minutes)}
                                        class="flex items-center justify-between rounded-md px-2 py-1.5 text-left text-xs transition-colors
                                        {timeFilter.mode === 'relative' && timeFilter.minutes === preset.minutes
                                            ? 'bg-primary font-medium text-primary-foreground'
                                            : 'text-foreground hover:bg-accent hover:text-accent-foreground'}"
                                    >
                                        Last {preset.label}
                                    </button>
                                {/each}
                            </div>
                        </div>
                    </div>
                </div>
            </Popover.Content>
        </Popover.Root>
    </div>

    {#if isLoading}
        <div class="flex items-center justify-center py-12 text-muted-foreground">
            <Spinner class="mr-2 size-4" />
            Loading telemetry...
        </div>
    {:else if error}
        <p class="text-sm text-destructive">{error}</p>
    {:else if tlmGroups.length === 0}
        <div class="flex flex-col items-center justify-center gap-3 py-12 text-muted-foreground">
            <ActivityIcon class="size-8 opacity-30" />
            <p class="text-sm">No telemetry found for selected devices</p>
        </div>
    {:else}
        {#each groupStates as gs, idx (idx)}
            <div>
                <div class="mb-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <span class="font-mono font-medium text-foreground">
                        {gs.group.ids.map((id) => `${id.name}/${id.namespace}`).join(', ')}
                    </span>
                    <span>({gs.group.ids.length} device{gs.group.ids.length > 1 ? 's' : ''})</span>
                </div>

                <div class="flex min-h-80 overflow-hidden rounded-lg border">
                    <DescriptorPanel
                        title="Telemetry"
                        items={gs.group.descriptors.map((d) => ({
                            name: d.name, labels: d.labels, template: '', error: d.error,
                        }))}
                        isLoading={false}
                        error={null}
                        groupErrors={gs.group.error ? [gs.group.error] : []}
                        selectedName={gs.selectedMeasurement?.name ?? null}
                        emptyText="No telemetry."
                        onSelect={(desc) => selectMeasurement(idx, desc)}
                    />

                    <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
                        {#if !gs.selectedMeasurement}
                            <div class="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
                                <ActivityIcon class="size-8 opacity-30" />
                                <p class="text-sm">Select a measurement to view chart</p>
                            </div>
                        {:else}
                            <!-- Legend: color swatches per device+field -->
                            <div class="flex flex-wrap items-center gap-2 border-b px-4 py-1.75">
                                {#each gs.mergedFields as field, i (field)}
                                    <span class="flex items-center gap-1 font-mono text-[11px]">
                                        <span class="h-2 w-2 rounded-full" style="background: {CHART_COLORS[i % CHART_COLORS.length]}"></span>
                                        {field.replace('_', ': ')}
                                    </span>
                                {/each}
                            </div>

                            <!-- Chart -->
                            <div class="flex-1 px-4 py-4">
                                {#if gs.queryError}
                                    <p class="text-sm text-destructive">{gs.queryError}</p>
                                {:else if gs.isQuerying && !gs.mergedData}
                                    <div class="flex h-48 items-center justify-center text-sm text-muted-foreground">
                                        Loading data…
                                    </div>
                                {:else if gs.mergedData}
                                    <TlmChart
                                        data={gs.mergedData}
                                        selectedFields={gs.mergedFields}
                                        chartConfig={gs.chartConfig}
                                        useUtc={editorPrefs.utc}
                                        chartClass="h-72"
                                    />
                                {/if}
                            </div>
                        {/if}
                    </div>
                </div>
            </div>

            {#if idx < groupStates.length - 1}
                <Separator />
            {/if}
        {/each}
    {/if}
</div>
```

**Step 2: Verify in browser**

- Select 2+ devices with same schema, click Telemetry
- Select a measurement → chart shows one line per device in different colors
- Change time range → all charts update
- Select devices with different schemas → multiple schema sections appear

**Step 3: Commit**

```bash
git add internal/ui/web/src/routes/devices/multi/telemetry/+page.svelte
git commit -m "feat(cockpit): add multi-device telemetry page with overlaid charts"
```

---

## Verification Checklist

End-to-end test flow:

1. Start infra: `mir infra up -d`
2. Start swarm: `mir swarm --ids=power/default,power/prod,weather/prod`
3. Start server: `mir serve`
4. Run dev: `cd internal/ui/web && npm run dev`
5. Open `http://localhost:5173/devices`

**Selection:**
- [ ] Checkbox column visible as first column
- [ ] Header checkbox selects/deselects all on current page
- [ ] Individual row checkboxes work
- [ ] Selection count shows in toolbar
- [ ] Bulk action bar appears at table bottom when ≥1 selected
- [ ] "Clear" button empties selection
- [ ] Navigate away and back → selection preserved (store persists)

**Navigation guard:**
- [ ] Manually navigate to `/devices/multi/commands` with nothing selected → redirects to `/devices`

**Multi layout:**
- [ ] Breadcrumb shows `Devices / Bulk Operations`
- [ ] Device chips show with online indicator
- [ ] Clicking X on a chip removes that device
- [ ] Tab bar navigates between Telemetry / Commands / Configuration
- [ ] Removing all chips redirects to `/devices`

**Commands:**
- [ ] Select `power/default` and `power/prod` → one schema group shown
- [ ] Select command, edit payload, send → both devices respond
- [ ] Responses show per device in ResponsePanel
- [ ] Select devices with different schemas → multiple schema sections

**Configuration:**
- [ ] Same as commands, with config-specific status labels (NOCHANGE shown)

**Telemetry:**
- [ ] Select measurement → chart appears with one colored line per device
- [ ] Legend shows `deviceName_field` entries with color swatches
- [ ] Time picker changes update all charts
- [ ] Different schema devices → separate sections with their own measurements
