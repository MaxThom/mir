import { create, type JsonObject } from "@bufbuild/protobuf";

// ─── Proto imports (P-prefixed to avoid collision with model names) ───────────

import type {
  Device as PDevice,
  DeviceTarget as PDeviceTarget,
  PropertiesTime as PPropertiesTime,
} from "./gen/proto/mir_api/v1/core_pb";
import {
  DeviceSchema as PDeviceSchema,
  DeviceTargetSchema as PDeviceTargetSchema,
  DeviceSpecSchema as PDeviceSpecSchema,
  DevicePropertiesSchema as PDevicePropertiesSchema,
  DeviceStatusSchema as PDeviceStatusSchema,
  DeviceStatusEventSchema as PDeviceStatusEventSchema,
  SchemaSchema as PSchemaSchema,
  PropertiesTimeSchema as PPropertiesTimeSchema,
} from "./gen/proto/mir_api/v1/core_pb";

import type {
  Object$ as PObject,
  Timestamp,
} from "./gen/proto/mir_api/v1/common_pb";
import {
  ObjectSchema as PObjectSchema,
  TimestampSchema,
  TargetsSchema as PTargetsSchema,
  DateFilterSchema as PDateFilterSchema,
  MetaSchema as PMetaSchema,
} from "./gen/proto/mir_api/v1/common_pb";

import type {
  Event as PEvent,
  EventTarget as PEventTarget,
} from "./gen/proto/mir_api/v1/event_pb";
import {
  EventSchema as PEventSchema,
  EventTargetSchema as PEventTargetSchema,
  EventSpecSchema as PEventSpecSchema,
  EventStatusSchema as PEventStatusSchema,
} from "./gen/proto/mir_api/v1/event_pb";

import type {
  Device,
  DeviceTarget,
  DeviceStatusEvent,
  PropertiesTime,
  MirObject,
  MirEvent,
  EventType,
  EventTarget,
} from "./models";

// ─── Timestamp ────────────────────────────────────────────────────────────────

export function timestampToDate(ts: Timestamp | undefined): Date | undefined {
  if (!ts) return undefined;
  return new Date(Number(ts.seconds) * 1000 + Math.floor(ts.nanos / 1_000_000));
}

export function dateToTimestamp(d: Date | undefined): Timestamp | undefined {
  if (!d) return undefined;
  const ms = d.getTime();
  return create(TimestampSchema, {
    seconds: BigInt(Math.floor(ms / 1000)),
    nanos: (ms % 1000) * 1_000_000,
  });
}

// ─── Object ───────────────────────────────────────────────────────────────────

export function objectFromProto(o: PObject | undefined): MirObject {
  if (!o) {
    return {
      apiVersion: "mir/v1alpha",
      kind: "",
      meta: { name: "", namespace: "", labels: {}, annotations: {} },
    };
  }
  return {
    apiVersion: o.apiVersion,
    kind: o.kind,
    meta: {
      name: o.meta?.name ?? "",
      namespace: o.meta?.namespace ?? "",
      labels: o.meta?.labels ?? {},
      annotations: o.meta?.annotations ?? {},
    },
  };
}

export function objectToProto(o: MirObject): PObject {
  return create(PObjectSchema, {
    apiVersion: o.apiVersion,
    kind: o.kind,
    meta: create(PMetaSchema, {
      name: o.meta.name,
      namespace: o.meta.namespace,
      labels: o.meta.labels,
      annotations: o.meta.annotations,
    }),
  });
}

// ─── DeviceTarget ─────────────────────────────────────────────────────────────

export function deviceTargetFromProto(
  t: PDeviceTarget | undefined,
): DeviceTarget {
  return {
    ids: t?.ids ?? [],
    names: t?.names ?? [],
    namespaces: t?.namespaces ?? [],
    labels: t?.labels ?? {},
  };
}

export function deviceTargetToProto(t: DeviceTarget): PDeviceTarget {
  return create(PDeviceTargetSchema, {
    ids: t.ids,
    names: t.names,
    namespaces: t.namespaces,
    labels: t.labels,
  });
}

// ─── Device ───────────────────────────────────────────────────────────────────

function protoPropsTimeToModel(p: PPropertiesTime | undefined): PropertiesTime {
  const desired: Record<string, Date> = {};
  const reported: Record<string, Date> = {};
  if (p?.desired) {
    for (const [k, v] of Object.entries(p.desired)) {
      const d = timestampToDate(v);
      if (d) desired[k] = d;
    }
  }
  if (p?.reported) {
    for (const [k, v] of Object.entries(p.reported)) {
      const d = timestampToDate(v);
      if (d) reported[k] = d;
    }
  }
  return { desired, reported };
}

function modelPropsTimeToProto(p: PropertiesTime): PPropertiesTime {
  const desired: { [k: string]: Timestamp } = {};
  const reported: { [k: string]: Timestamp } = {};
  for (const [k, v] of Object.entries(p.desired)) {
    const ts = dateToTimestamp(v);
    if (ts) desired[k] = ts;
  }
  for (const [k, v] of Object.entries(p.reported)) {
    const ts = dateToTimestamp(v);
    if (ts) reported[k] = ts;
  }
  return create(PPropertiesTimeSchema, { desired, reported });
}

export function deviceFromProto(d: PDevice | undefined): Device {
  if (!d) {
    return {
      apiVersion: "mir/v1alpha",
      kind: "device",
      meta: { name: "", namespace: "", labels: {}, annotations: {} },
      spec: { deviceId: "" },
      properties: { desired: {}, reported: {} },
      status: {
        schema: { packageNames: [] },
        properties: { desired: {}, reported: {} },
        events: [],
      },
    };
  }

  const events: DeviceStatusEvent[] = (d.status?.events ?? []).map((e) => ({
    type: e.type as EventType,
    reason: e.reason,
    message: e.message,
    firstAt: timestampToDate(e.firstAt),
  }));

  return {
    apiVersion: d.apiVersion,
    kind: d.kind,
    meta: {
      name: d.meta?.name ?? "",
      namespace: d.meta?.namespace ?? "",
      labels: d.meta?.labels ?? {},
      annotations: d.meta?.annotations ?? {},
    },
    spec: {
      deviceId: d.spec?.deviceId ?? "",
      disabled: d.spec?.disabled,
    },
    properties: {
      desired: (d.properties?.desired ?? {}) as Record<string, unknown>,
      reported: (d.properties?.reported ?? {}) as Record<string, unknown>,
    },
    status: {
      online: d.status?.online,
      lastHearthbeat: timestampToDate(d.status?.lastHearthbeat),
      schema: {
        compressedSchema: d.status?.schema?.compressedSchema,
        packageNames: d.status?.schema?.packageNames ?? [],
        lastSchemaFetch: timestampToDate(d.status?.schema?.lastSchemaFetch),
      },
      properties: protoPropsTimeToModel(d.status?.properties),
      events,
    },
  };
}

export function deviceToProto(d: Device): PDevice {
  const events = d.status.events.map((e) =>
    create(PDeviceStatusEventSchema, {
      type: e.type,
      reason: e.reason,
      message: e.message,
      firstAt: dateToTimestamp(e.firstAt),
    }),
  );

  return create(PDeviceSchema, {
    apiVersion: d.apiVersion,
    kind: d.kind,
    meta: create(PMetaSchema, {
      name: d.meta.name,
      namespace: d.meta.namespace,
      labels: d.meta.labels,
      annotations: d.meta.annotations,
    }),
    spec: create(PDeviceSpecSchema, {
      deviceId: d.spec.deviceId,
      disabled: d.spec.disabled ?? false,
    }),
    properties: create(PDevicePropertiesSchema, {
      desired: d.properties.desired as JsonObject,
      reported: d.properties.reported as JsonObject,
    }),
    status: create(PDeviceStatusSchema, {
      online: d.status.online ?? false,
      lastHearthbeat: dateToTimestamp(d.status.lastHearthbeat),
      schema: create(PSchemaSchema, {
        compressedSchema: d.status.schema.compressedSchema ?? new Uint8Array(),
        packageNames: d.status.schema.packageNames,
        lastSchemaFetch: dateToTimestamp(d.status.schema.lastSchemaFetch),
      }),
      properties: modelPropsTimeToProto(d.status.properties),
      events,
    }),
  });
}

export function devicesFromProto(devices: PDevice[]): Device[] {
  return devices.map(deviceFromProto);
}

export function devicesToProto(devices: Device[]): PDevice[] {
  return devices.map(deviceToProto);
}

// ─── Event ────────────────────────────────────────────────────────────────────

export function eventFromProto(e: PEvent | undefined): MirEvent {
  if (!e) {
    return {
      apiVersion: "mir/v1alpha",
      kind: "event",
      meta: { name: "", namespace: "", labels: {}, annotations: {} },
      spec: {
        type: "normal",
        reason: "",
        message: "",
        relatedObject: objectFromProto(undefined),
      },
      status: { count: 0 },
    };
  }

  let payload: unknown = undefined;
  if (e.spec?.jsonPayload && e.spec.jsonPayload.length > 0) {
    try {
      payload = JSON.parse(new TextDecoder().decode(e.spec.jsonPayload));
    } catch {
      payload = undefined;
    }
  }

  return {
    apiVersion: e.object?.apiVersion ?? "mir/v1alpha",
    kind: e.object?.kind ?? "event",
    meta: {
      name: e.object?.meta?.name ?? "",
      namespace: e.object?.meta?.namespace ?? "",
      labels: e.object?.meta?.labels ?? {},
      annotations: e.object?.meta?.annotations ?? {},
    },
    spec: {
      type: (e.spec?.type ?? "normal") as EventType,
      reason: e.spec?.reason ?? "",
      message: e.spec?.message ?? "",
      payload,
      relatedObject: objectFromProto(e.spec?.relatedObject),
    },
    status: {
      count: e.status?.count ?? 0,
      firstAt: timestampToDate(e.status?.firstAt),
      lastAt: timestampToDate(e.status?.lastAt),
    },
  };
}

export function eventToProto(e: MirEvent): PEvent {
  const jsonPayload =
    e.spec.payload !== undefined
      ? new TextEncoder().encode(JSON.stringify(e.spec.payload))
      : new Uint8Array();

  return create(PEventSchema, {
    object: objectToProto({
      apiVersion: e.apiVersion,
      kind: e.kind,
      meta: e.meta,
    }),
    spec: create(PEventSpecSchema, {
      type: e.spec.type,
      reason: e.spec.reason,
      message: e.spec.message,
      jsonPayload,
      relatedObject: objectToProto(e.spec.relatedObject),
    }),
    status: create(PEventStatusSchema, {
      count: e.status.count,
      firstAt: dateToTimestamp(e.status.firstAt),
      lastAt: dateToTimestamp(e.status.lastAt),
    }),
  });
}

export function eventsFromProto(events: PEvent[]): MirEvent[] {
  return events.map(eventFromProto);
}

export function eventsToProto(events: MirEvent[]): PEvent[] {
  return events.map(eventToProto);
}

// ─── EventTarget ──────────────────────────────────────────────────────────────

export function eventTargetFromProto(t: PEventTarget | undefined): EventTarget {
  return {
    names: t?.targets?.names ?? [],
    namespaces: t?.targets?.namespaces ?? [],
    labels: t?.targets?.labels ?? {},
    dateFilter: {
      from: timestampToDate(t?.filterDate?.from) ?? new Date(0),
      to: timestampToDate(t?.filterDate?.to) ?? new Date(0),
    },
    limit: t?.filterLimit ?? 0,
  };
}

export function eventTargetToProto(t: EventTarget): PEventTarget {
  return create(PEventTargetSchema, {
    targets: create(PTargetsSchema, {
      names: t.names,
      namespaces: t.namespaces,
      labels: t.labels,
    }),
    filterDate: create(PDateFilterSchema, {
      from: dateToTimestamp(t.dateFilter.from),
      to: dateToTimestamp(t.dateFilter.to),
    }),
    filterLimit: t.limit,
  });
}
