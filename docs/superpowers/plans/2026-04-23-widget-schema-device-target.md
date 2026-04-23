# Widget Schema Device Target — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `schemas` as a dynamic AND-logic filter to the widget device target system, from the protobuf wire format through backend SurrealDB filtering, TypeScript SDK, and Svelte UI.

**Architecture:** Add `repeated string schemas = 5` to `DeviceTarget` proto, propagate through Go struct + backend WHERE-clause builder, update TS SDK model and transforms, then wire up the new filter section in the Svelte builder component.

**Tech Stack:** Go, Protocol Buffers (buf), SurrealDB array functions, TypeScript, Svelte 5

---

### Task 1: Add `schemas` to DeviceTarget proto and Go struct

**Files:**
- Modify: `pkgs/api/proto/mir_api/v1/core.proto:172-177`
- Modify: `pkgs/mir_v1/device.go:93-115`

- [ ] **Step 1: Add `schemas` field to proto**

In `pkgs/api/proto/mir_api/v1/core.proto`, replace:
```protobuf
message DeviceTarget {
  repeated string names = 1; // names of the objects
  repeated string namespaces = 2; // namespace of the object
  map<string, string> labels = 3; // Labels are used to uniquely identify the object
  repeated string ids = 4; // ids of the devices
}
```
With:
```protobuf
message DeviceTarget {
  repeated string names = 1; // names of the objects
  repeated string namespaces = 2; // namespace of the object
  map<string, string> labels = 3; // Labels are used to uniquely identify the object
  repeated string ids = 4; // ids of the devices
  repeated string schemas = 5; // schema package names — device must have ALL (AND logic)
}
```

- [ ] **Step 2: Add `Schemas` to Go DeviceTarget struct and update helper methods**

In `pkgs/mir_v1/device.go`, replace:
```go
type DeviceTarget struct {
	Ids        []string
	Names      []string
	Namespaces []string
	Labels     map[string]string
}

func (o DeviceTarget) HasNoTarget() bool {
	return len(o.Names) == 0 &&
		len(o.Namespaces) == 0 &&
		len(o.Labels) == 0 &&
		len(o.Ids) == 0
}

func (o DeviceTarget) HasOnlyIdsTarget() bool {
	if len(o.Names) > 0 || len(o.Namespaces) > 0 || len(o.Labels) > 0 {
		return false
	}
	if len(o.Ids) == 0 {
		return false
	}
	return true
}
```
With:
```go
type DeviceTarget struct {
	Ids        []string
	Names      []string
	Namespaces []string
	Labels     map[string]string
	Schemas    []string
}

func (o DeviceTarget) HasNoTarget() bool {
	return len(o.Names) == 0 &&
		len(o.Namespaces) == 0 &&
		len(o.Labels) == 0 &&
		len(o.Ids) == 0 &&
		len(o.Schemas) == 0
}

func (o DeviceTarget) HasOnlyIdsTarget() bool {
	if len(o.Names) > 0 || len(o.Namespaces) > 0 || len(o.Labels) > 0 || len(o.Schemas) > 0 {
		return false
	}
	if len(o.Ids) == 0 {
		return false
	}
	return true
}
```

- [ ] **Step 3: Regenerate protobuf code**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server && just protogen
```
Expected: Exits 0, regenerated files in `pkgs/api/gen/`.

- [ ] **Step 4: Verify Go compiles**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server && go build ./...
```
Expected: No errors.

---

### Task 2: Update Go transforms and backend SurrealDB filter

**Files:**
- Modify: `pkgs/mir_v1/transform.go:498-517`
- Modify: `internal/externals/mng/devices.go:672-707`

- [ ] **Step 1: Update both Go transform functions**

In `pkgs/mir_v1/transform.go`, replace:
```go
func ProtoDeviceTargetToMirDeviceTarget(t *mir_apiv1.DeviceTarget) DeviceTarget {
	if t == nil {
		return DeviceTarget{}
	}
	return DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
	}
}

func MirDeviceTargetToProtoDeviceTarget(t DeviceTarget) *mir_apiv1.DeviceTarget {
	return &mir_apiv1.DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
	}
}
```
With:
```go
func ProtoDeviceTargetToMirDeviceTarget(t *mir_apiv1.DeviceTarget) DeviceTarget {
	if t == nil {
		return DeviceTarget{}
	}
	return DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
		Schemas:    t.Schemas,
	}
}

func MirDeviceTargetToProtoDeviceTarget(t DeviceTarget) *mir_apiv1.DeviceTarget {
	return &mir_apiv1.DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
		Schemas:    t.Schemas,
	}
}
```

- [ ] **Step 2: Add schema filter to SurrealDB WHERE builder**

In `internal/externals/mng/devices.go`, replace the `createDeviceWhereStatementWithTargets` function:
```go
func createDeviceWhereStatementWithTargets(t mir_v1.DeviceTarget) string {
	var q strings.Builder

	cond := []string{}
	if len(t.Ids) > 0 {
		var i []string
		for _, id := range t.Ids {
			i = append(i, wildcardToSurreal("spec.deviceId", id))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Names) > 0 {
		var i []string
		for _, ns := range t.Names {
			i = append(i, wildcardToSurreal("meta.name", ns))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Namespaces) > 0 {
		var i []string
		for _, ns := range t.Namespaces {
			i = append(i, wildcardToSurreal("meta.namespace", ns))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Labels) > 0 {
		var i []string
		for k, v := range t.Labels {
			i = append(i, wildcardToSurreal(fmt.Sprintf("meta.labels.%s", k), v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " AND "))
	ti := q.String()
	return ti
}
```
With:
```go
func createDeviceWhereStatementWithTargets(t mir_v1.DeviceTarget) string {
	var q strings.Builder

	cond := []string{}
	if len(t.Ids) > 0 {
		var i []string
		for _, id := range t.Ids {
			i = append(i, wildcardToSurreal("spec.deviceId", id))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Names) > 0 {
		var i []string
		for _, ns := range t.Names {
			i = append(i, wildcardToSurreal("meta.name", ns))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Namespaces) > 0 {
		var i []string
		for _, ns := range t.Namespaces {
			i = append(i, wildcardToSurreal("meta.namespace", ns))
		}
		cond = append(cond, "("+strings.Join(i, " OR ")+")")
	}
	if len(t.Labels) > 0 {
		var i []string
		for k, v := range t.Labels {
			i = append(i, wildcardToSurreal(fmt.Sprintf("meta.labels.%s", k), v))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	if len(t.Schemas) > 0 {
		var i []string
		for _, s := range t.Schemas {
			i = append(i, fmt.Sprintf("array::contains((status.schema.packageNames ?? []), %q)", s))
		}
		cond = append(cond, "("+strings.Join(i, " AND ")+")")
	}
	q.WriteString(strings.Join(cond, " AND "))
	ti := q.String()
	return ti
}
```

- [ ] **Step 3: Verify Go still compiles**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server && go build ./...
```
Expected: No errors.

---

### Task 3: Update TypeScript SDK

**Files:**
- Modify: `pkgs/web/src/models.ts:133-142`
- Modify: `pkgs/web/src/transform.ts:107-127`

- [ ] **Step 1: Add `schemas` to TS DeviceTarget model**

In `pkgs/web/src/models.ts`, replace:
```typescript
export class DeviceTarget {
  ids: string[] = [];
  names: string[] = [];
  namespaces: string[] = [];
  labels: Record<string, string> = {};

  constructor(data?: Partial<DeviceTarget>) {
    if (data) Object.assign(this, data);
  }
}
```
With:
```typescript
export class DeviceTarget {
  ids: string[] = [];
  names: string[] = [];
  namespaces: string[] = [];
  labels: Record<string, string> = {};
  schemas: string[] = [];

  constructor(data?: Partial<DeviceTarget>) {
    if (data) Object.assign(this, data);
  }
}
```

- [ ] **Step 2: Update both TS transform functions**

In `pkgs/web/src/transform.ts`, replace:
```typescript
export function deviceTargetFromProto(
  t: PDeviceTarget | undefined,
): DeviceTarget {
  return new DeviceTarget({
    ids: t?.ids ?? [],
    names: t?.names ?? [],
    namespaces: t?.namespaces ?? [],
    labels: t?.labels ?? {},
  });
}

export function deviceTargetToProto(t: DeviceTarget): PDeviceTarget {
  return create(PDeviceTargetSchema, {
    ids: t.ids,
    names: t.names,
    namespaces: t.namespaces,
    labels: t.labels,
  });
}
```
With:
```typescript
export function deviceTargetFromProto(
  t: PDeviceTarget | undefined,
): DeviceTarget {
  return new DeviceTarget({
    ids: t?.ids ?? [],
    names: t?.names ?? [],
    namespaces: t?.namespaces ?? [],
    labels: t?.labels ?? {},
    schemas: t?.schemas ?? [],
  });
}

export function deviceTargetToProto(t: DeviceTarget): PDeviceTarget {
  return create(PDeviceTargetSchema, {
    ids: t.ids,
    names: t.names,
    namespaces: t.namespaces,
    labels: t.labels,
    schemas: t.schemas,
  });
}
```

- [ ] **Step 3: Rebuild SDK dist**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server && npm run build --prefix ./pkgs/web
```
Expected: Exits 0, updated files in `pkgs/web/dist/`.

---

### Task 4: Update DeviceTargetConfig persistence type

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts:3-8`

- [ ] **Step 1: Add `schemas` to DeviceTargetConfig**

In `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts`, replace:
```typescript
export interface DeviceTargetConfig {
	ids?: string[];
	names?: string[];
	namespaces?: string[];
	labels?: Record<string, string>;
}
```
With:
```typescript
export interface DeviceTargetConfig {
	ids?: string[];
	names?: string[];
	namespaces?: string[];
	labels?: Record<string, string>;
	schemas?: string[];
}
```

---

### Task 5: Update device-target-builder.svelte

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/device-target-builder.svelte`

- [ ] **Step 1: Add schema state, derived suggestions, and helper functions**

In the `<script>` block, replace the existing state/derived section (lines 28-115) with the updated version that adds schema support. Specifically:

Replace:
```typescript
let mode = $state<'dynamic' | 'specific'>(
    untrack(() =>
        initialTarget?.namespaces?.length || initialTarget?.labels ? 'dynamic' : 'specific'
    )
);

// Dynamic state
let selectedNamespaces = $state<string[]>(untrack(() => initialTarget?.namespaces ?? []));
let labelConditions = $state<{ key: string; value: string }[]>(
    untrack(() =>
        Object.entries(initialTarget?.labels ?? {}).map(([key, value]) => ({ key, value }))
    )
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
```

With:
```typescript
let mode = $state<'dynamic' | 'specific'>(
    untrack(() =>
        initialTarget?.namespaces?.length || initialTarget?.labels || initialTarget?.schemas?.length
            ? 'dynamic'
            : 'specific'
    )
);

// Dynamic state
let selectedNamespaces = $state<string[]>(untrack(() => initialTarget?.namespaces ?? []));
let labelConditions = $state<{ key: string; value: string }[]>(
    untrack(() =>
        Object.entries(initialTarget?.labels ?? {}).map(([key, value]) => ({ key, value }))
    )
);
let selectedSchemas = $state<string[]>(untrack(() => initialTarget?.schemas ?? []));

// Specific state
let selectedIds = $state<string[]>(untrack(() => initialTarget?.ids ?? []));

// ─── Derived suggestions ──────────────────────────────────────────────────

const allNamespaces = $derived(
    [...new Set(devices.map((d) => d.meta?.namespace ?? 'default'))].sort()
);
const allLabelKeys = $derived(
    [...new Set(devices.flatMap((d) => Object.keys(d.meta?.labels ?? {})))].sort()
);
const allSchemas = $derived(
    [
        ...new Set(
            devices
                .flatMap((d) => d.status?.schema?.packageNames ?? [])
                .filter((p) => p !== 'mir.device.v1' && p !== 'google.protobuf')
        )
    ].sort()
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
        const pkgNames = d.status?.schema?.packageNames ?? [];
        const activeSchemas = selectedSchemas.filter((s) => s);
        const schemaMatch = activeSchemas.length === 0 || activeSchemas.every((s) => pkgNames.includes(s));
        return nsMatch && labelMatch && schemaMatch;
    })
);

const previewOnline = $derived(previewDevices.filter((d) => d.status?.online).length);

// ─── Sync to bindable target ──────────────────────────────────────────────

$effect(() => {
    if (mode === 'dynamic') {
        const activeNs = selectedNamespaces.filter((ns) => ns);
        const valid = labelConditions.filter((c) => c.key && c.value);
        const activeSchemas = selectedSchemas.filter((s) => s);
        target = {
            ...(activeNs.length ? { namespaces: activeNs } : {}),
            ...(valid.length ? { labels: Object.fromEntries(valid.map((c) => [c.key, c.value])) } : {}),
            ...(activeSchemas.length ? { schemas: activeSchemas } : {})
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

// ─── Schema helpers ───────────────────────────────────────────────────────

function addSchema() {
    selectedSchemas = [...selectedSchemas, ''];
}
function removeSchema(i: number) {
    selectedSchemas = selectedSchemas.filter((_, idx) => idx !== i);
}
```

- [ ] **Step 2: Add `border-b` to the Labels section and insert Schemas section**

In the template, replace:
```svelte
		<!-- Labels -->
		<div class="space-y-2 p-3">
```
With:
```svelte
		<!-- Labels -->
		<div class="space-y-2 border-b p-3">
```

Then, after the closing `</div>` of the Labels section (just before the closing `</div>` of the outer filter container, before the preview table comment), add the Schemas section:
```svelte
		<!-- Schemas -->
		<div class="space-y-2 p-3">
			<div class="flex items-center justify-between">
				<span class="text-xs font-semibold tracking-wide text-muted-foreground uppercase"
					>Schemas</span
				>
				<span class="text-xs text-muted-foreground">match all (AND)</span>
			</div>

			{#if selectedSchemas.length > 0}
				<div class="space-y-1.5">
					{#each selectedSchemas, i (i)}
						<div class="flex items-center gap-1.5">
							<SuggestionInput
								bind:value={selectedSchemas[i]}
								suggestions={allSchemas}
								placeholder="schema package"
								class="h-7 flex-1 font-mono text-xs"
							/>
							<Button
								variant="ghost"
								size="icon-sm"
								onclick={() => removeSchema(i)}
								aria-label="Remove schema"
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
				onclick={addSchema}
				class="h-7 gap-1 px-2 text-xs text-muted-foreground"
			>
				<PlusIcon class="size-3" />
				Add schema
			</Button>
		</div>
	</div>
```

> **Note:** The `</div>` at the end closes the outer `<div class="rounded-md border border-border">` that wraps the Namespaces + Labels + Schemas sections. Remove the previous closing `</div>` that ended after Labels, since this new snippet includes it.

The full updated template structure for the dynamic filter container should be:
```svelte
<div class="rounded-md border border-border">
    <!-- Namespaces -->
    <div class="space-y-2 border-b p-3">
        ... (unchanged)
    </div>

    <!-- Labels -->
    <div class="space-y-2 border-b p-3">
        ... (unchanged content, border-b added)
    </div>

    <!-- Schemas -->
    <div class="space-y-2 p-3">
        ... (new section)
    </div>
</div>
```

---

### Task 6: Update add-widget-dialog.svelte device filter

**Files:**
- Modify: `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte`

There are three functions (`loadMeasurements`, `loadCommandsForWizard`, `loadConfigsForWizard`) that each have the same inline device filter. Add the schema check to all three.

- [ ] **Step 1: Update `loadMeasurements` filter**

Replace (appears around line 264):
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;

			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			const groups = (await mirStore
				.mir!.client()
				.listTelemetry()
				.request(deviceTarget)) as TelemetryGroup[];
```
With:
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						const pkgNames = d.status?.schema?.packageNames ?? [];
						const schemaMatch =
							!target.schemas?.length || target.schemas.every((s) => pkgNames.includes(s));
						return nsMatch && labelMatch && schemaMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;

			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			const groups = (await mirStore
				.mir!.client()
				.listTelemetry()
				.request(deviceTarget)) as TelemetryGroup[];
```

- [ ] **Step 2: Update `loadCommandsForWizard` filter**

Replace (appears around line 306):
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			commandGroups = await mirStore.mir!.client().listCommands().request(deviceTarget);
```
With:
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						const pkgNames = d.status?.schema?.packageNames ?? [];
						const schemaMatch =
							!target.schemas?.length || target.schemas.every((s) => pkgNames.includes(s));
						return nsMatch && labelMatch && schemaMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			commandGroups = await mirStore.mir!.client().listCommands().request(deviceTarget);
```

- [ ] **Step 3: Update `loadConfigsForWizard` filter**

Replace (appears around line 337):
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			configGroups = await mirStore.mir!.client().listConfigs().request(deviceTarget);
```
With:
```typescript
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						const pkgNames = d.status?.schema?.packageNames ?? [];
						const schemaMatch =
							!target.schemas?.length || target.schemas.every((s) => pkgNames.includes(s));
						return nsMatch && labelMatch && schemaMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			configGroups = await mirStore.mir!.client().listConfigs().request(deviceTarget);
```

---

### Task 7: Type-check the web UI

**Files:** None modified — verification only.

- [ ] **Step 1: Run type checker**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server/internal/ui/web && npm run check 2>&1
```
Expected: The 3 pre-existing errors in `nav-section.svelte` and `multi/telemetry` appear, but **no new errors**. If any new errors appear, fix them before continuing.

- [ ] **Step 2: Verify Go still compiles clean**

```bash
cd /home/maxthom/code/mir-ecosystem/mir.server && go build ./...
```
Expected: No errors.
