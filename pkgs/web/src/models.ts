// ─── Object / Meta ────────────────────────────────────────────────────────────

export interface Meta {
  name: string;
  namespace: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
}

/** null value on labels/annotations means "delete this key" */
export interface MetaUpdate {
  name?: string;
  namespace?: string;
  labels?: Record<string, string | null>;
  annotations?: Record<string, string | null>;
}

/** Base object type — renamed from Object to avoid collision with JS builtin */
export interface MirObject {
  apiVersion: string;
  kind: string;
  meta: Meta;
}

export interface ObjectTarget {
  names: string[];
  namespaces: string[];
  labels: Record<string, string>;
}

export interface DateFilter {
  from: Date;
  to: Date;
}

// ─── Device ───────────────────────────────────────────────────────────────────

export interface DeviceSpec {
  deviceId: string;
  disabled?: boolean;
}

export interface DeviceProperties {
  desired: Record<string, unknown>;
  reported: Record<string, unknown>;
}

export interface PropertiesTime {
  desired: Record<string, Date>;
  reported: Record<string, Date>;
}

export interface DeviceSchema {
  compressedSchema?: Uint8Array;
  packageNames: string[];
  lastSchemaFetch?: Date;
}

export type EventType = "normal" | "warning";
export const EventTypeNormal: EventType = "normal";
export const EventTypeWarning: EventType = "warning";

export interface DeviceStatusEvent {
  type: EventType;
  reason: string;
  message: string;
  firstAt?: Date;
}

export interface DeviceStatus {
  online?: boolean;
  lastHearthbeat?: Date;
  schema: DeviceSchema;
  properties: PropertiesTime;
  events: DeviceStatusEvent[];
}

export interface Device extends MirObject {
  spec: DeviceSpec;
  properties: DeviceProperties;
  status: DeviceStatus;
}

export interface DeviceTarget {
  ids: string[];
  names: string[];
  namespaces: string[];
  labels: Record<string, string>;
}

// ─── Event ────────────────────────────────────────────────────────────────────

export interface EventSpec {
  type: EventType;
  reason: string;
  message: string;
  payload?: unknown;
  relatedObject: MirObject;
}

export interface EventStatus {
  count: number;
  firstAt?: Date;
  lastAt?: Date;
}

export interface MirEvent extends MirObject {
  spec: EventSpec;
  status: EventStatus;
}

export interface EventTarget extends ObjectTarget {
  dateFilter: DateFilter;
  limit: number;
}

export interface EventUpdateSpec {
  type?: EventType;
  reason?: string;
  message?: string;
  payload?: unknown;
}

export interface EventUpdateStatus {
  count?: number;
  firstAt?: Date;
  lastAt?: Date;
}

export interface EventUpdate {
  meta?: MetaUpdate;
  spec?: EventUpdateSpec;
  status?: EventUpdateStatus;
}
