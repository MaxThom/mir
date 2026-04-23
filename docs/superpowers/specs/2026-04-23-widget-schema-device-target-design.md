# Widget Schema Device Target — Design Spec

**Date:** 2026-04-23
**Status:** Approved

## Overview

Add `schemas` as a third dynamic filter option to the widget device target selection, alongside the existing `namespaces` and `labels` filters. A device matches if it contains **all** selected schemas in its `status.schema.packageNames` array (AND logic — same as labels).

## Section 1: Data Model

### `pkgs/api/proto/mir_api/v1/core.proto`
Add field to `DeviceTarget`:
```protobuf
message DeviceTarget {
  repeated string names      = 1;
  repeated string namespaces = 2;
  map<string, string> labels = 3;
  repeated string ids        = 4;
  repeated string schemas    = 5;  // schema package names — AND logic
}
```
Regenerate with `just protogen`.

### `pkgs/mir_v1/device.go`
Add to Go `DeviceTarget` struct:
```go
type DeviceTarget struct {
  Ids        []string
  Names      []string
  Namespaces []string
  Labels     map[string]string
  Schemas    []string
}
```

### `pkgs/mir_v1/transform.go`
Update `ProtoDeviceTargetToMirDeviceTarget`:
```go
return DeviceTarget{
  Names:      t.Names,
  Namespaces: t.Namespaces,
  Labels:     t.Labels,
  Ids:        t.Ids,
  Schemas:    t.Schemas,
}
```

## Section 2: Backend Filtering

### `internal/externals/mng/devices.go` — `createDeviceWhereStatementWithTargets()`

Schema names are stored in `status.schema.packageNames` (SurrealDB array). Each selected schema produces an `array::contains()` condition; all conditions joined with AND:

```go
if len(t.Schemas) > 0 {
  var i []string
  for _, s := range t.Schemas {
    i = append(i, fmt.Sprintf("array::contains((status.schema.packageNames ?? []), %q)", s))
  }
  cond = append(cond, "("+strings.Join(i, " AND ")+")")
}
```

This slots into the existing `cond` slice alongside namespace/label conditions, which are already joined with AND at the end.

## Section 3: TypeScript SDK

### `pkgs/web/src/models.ts`
```typescript
export class DeviceTarget {
  ids: string[] = [];
  names: string[] = [];
  namespaces: string[] = [];
  labels: Record<string, string> = {};
  schemas: string[] = [];
  // ...
}
```

### `pkgs/web/src/transform.ts`
Update both directions:
```typescript
export function deviceTargetFromProto(t: PDeviceTarget | undefined): DeviceTarget {
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

No changes needed to `ListTelemetry`, `ListCommands`, or `ListConfigs` — they already pass the full `DeviceTarget` through.

## Section 4: Frontend

### `internal/ui/web/src/lib/domains/dashboards/components/device-target-builder.svelte`

**State:**
```typescript
let selectedSchemas: string[] = $state([]);
```

**Available schema options** (derived from all devices in store, filtered to exclude built-ins):
```typescript
const availableSchemas = $derived(
  [...new Set(
    devices.flatMap(d => d.status?.schema?.packageNames ?? [])
      .filter(p => p !== 'mir.device.v1' && p !== 'google.protobuf')
  )].sort()
);
```

**Preview filter** — add schema AND check alongside existing namespace/label checks:
```typescript
const previewDevices = $derived(
  devices.filter((d) => {
    const activeNs = selectedNamespaces.filter((ns) => ns);
    const nsMatch = activeNs.length === 0 || activeNs.includes(d.meta?.namespace ?? 'default');
    const valid = labelConditions.filter((c) => c.key && c.value);
    const labelMatch = valid.every(({ key, value }) => d.meta?.labels?.[key] === value);
    const pkgNames = d.status?.schema?.packageNames ?? [];
    const schemaMatch = selectedSchemas.length === 0 || selectedSchemas.every(s => pkgNames.includes(s));
    return nsMatch && labelMatch && schemaMatch;
  })
);
```

**UI:** A filter row with schema chips (X to remove) and a `SuggestionInput` for adding — same visual pattern as namespaces. Placed between labels and the preview table.

**Output target:** Include `schemas: selectedSchemas` when building the `DeviceTarget` output.

### `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte`

Add schema check to the client-side device ID resolution (lines 265–276):
```typescript
const pkgNames = d.status?.schema?.packageNames ?? [];
const schemaMatch = !target.schemas?.length || target.schemas.every(s => pkgNames.includes(s));
return nsMatch && labelMatch && schemaMatch;
```

## Files Changed Summary

| File | Change |
|------|--------|
| `pkgs/api/proto/mir_api/v1/core.proto` | Add `schemas` field (field 5) to `DeviceTarget` |
| `pkgs/mir_v1/device.go` | Add `Schemas []string` to Go struct |
| `pkgs/mir_v1/transform.go` | Map `schemas` in proto↔Go transform |
| `internal/externals/mng/devices.go` | Add `array::contains` schema filter to WHERE builder |
| `pkgs/web/src/models.ts` | Add `schemas: string[]` to TS `DeviceTarget` |
| `pkgs/web/src/transform.ts` | Map `schemas` in proto↔TS transforms |
| `pkgs/web/src/` (build) | Rebuild SDK dist (`pkgs/web/dist/`) after SDK changes |
| `internal/ui/web/src/lib/domains/dashboards/api/dashboard-api.ts` | Add `schemas?: string[]` to `DeviceTargetConfig` interface |
| `internal/ui/web/src/lib/domains/dashboards/components/device-target-builder.svelte` | Schema filter UI + preview logic; initialize `selectedSchemas` from incoming target |
| `internal/ui/web/src/lib/domains/dashboards/components/add-widget-dialog.svelte` | Schema check in device ID resolution |
