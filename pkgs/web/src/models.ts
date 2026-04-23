// ─── Object / Meta ────────────────────────────────────────────────────────────

export class Meta {
  name = "";
  namespace = "";
  labels: Record<string, string> = {};
  annotations: Record<string, string> = {};

  constructor(data?: Partial<Meta>) {
    if (data) Object.assign(this, data);
  }
}

/** null value on labels/annotations means "delete this key" */
export class MetaUpdate {
  name?: string;
  namespace?: string;
  labels?: Record<string, string | null>;
  annotations?: Record<string, string | null>;

  constructor(data?: Partial<MetaUpdate>) {
    if (data) Object.assign(this, data);
  }
}

/** Base object type — renamed from Object to avoid collision with JS builtin */
export class MirObject {
  apiVersion = "mir/v1alpha";
  kind = "";
  meta = new Meta();

  constructor(data?: Partial<MirObject>) {
    if (data) Object.assign(this, data);
  }
}

export class ObjectTarget {
  names: string[] = [];
  namespaces: string[] = [];
  labels: Record<string, string> = {};

  constructor(data?: Partial<ObjectTarget>) {
    if (data) Object.assign(this, data);
  }
}

export class DateFilter {
  from?: Date;
  to?: Date;

  constructor(data?: Partial<DateFilter>) {
    if (data) Object.assign(this, data);
  }
}

// ─── Device ───────────────────────────────────────────────────────────────────

export class DeviceSpec {
  deviceId = "";
  disabled?: boolean;

  constructor(data?: Partial<DeviceSpec>) {
    if (data) Object.assign(this, data);
  }
}

export class DeviceProperties {
  desired: Record<string, unknown> = {};
  reported: Record<string, unknown> = {};

  constructor(data?: Partial<DeviceProperties>) {
    if (data) Object.assign(this, data);
  }
}

export class PropertiesTime {
  desired: Record<string, Date> = {};
  reported: Record<string, Date> = {};

  constructor(data?: Partial<PropertiesTime>) {
    if (data) Object.assign(this, data);
  }
}

export class DeviceSchema {
  compressedSchema?: Uint8Array;
  packageNames: string[] = [];
  lastSchemaFetch?: Date;

  constructor(data?: Partial<DeviceSchema>) {
    if (data) Object.assign(this, data);
  }
}

export type EventType = "normal" | "warning";
export const EventTypeNormal: EventType = "normal";
export const EventTypeWarning: EventType = "warning";

export class DeviceStatusEvent {
  type: EventType = "normal";
  reason = "";
  message = "";
  firstAt?: Date;

  constructor(data?: Partial<DeviceStatusEvent>) {
    if (data) Object.assign(this, data);
  }
}

export class DeviceStatus {
  online?: boolean;
  lastHearthbeat?: Date;
  schema = new DeviceSchema();
  properties = new PropertiesTime();
  events: DeviceStatusEvent[] = [];

  constructor(data?: Partial<DeviceStatus>) {
    if (data) Object.assign(this, data);
  }
}

export class Device extends MirObject {
  spec = new DeviceSpec();
  properties = new DeviceProperties();
  status = new DeviceStatus();

  constructor(data?: Partial<Device>) {
    super();
    if (data) Object.assign(this, data);
  }
}

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

// ─── Event ────────────────────────────────────────────────────────────────────

export class EventSpec {
  type: EventType = "normal";
  reason = "";
  message = "";
  payload?: unknown;
  relatedObject = new MirObject();

  constructor(data?: Partial<EventSpec>) {
    if (data) Object.assign(this, data);
  }
}

export class EventStatus {
  count = 0;
  firstAt?: Date;
  lastAt?: Date;

  constructor(data?: Partial<EventStatus>) {
    if (data) Object.assign(this, data);
  }
}

export class MirEvent extends MirObject {
  spec = new EventSpec();
  status = new EventStatus();

  constructor(data?: Partial<MirEvent>) {
    super();
    if (data) Object.assign(this, data);
  }
}

export class EventTarget extends ObjectTarget {
  dateFilter = new DateFilter();
  limit = 0;

  constructor(data?: Partial<EventTarget>) {
    super();
    if (data) Object.assign(this, data);
  }
}

export class EventUpdateSpec {
  type?: EventType;
  reason?: string;
  message?: string;
  payload?: unknown;

  constructor(data?: Partial<EventUpdateSpec>) {
    if (data) Object.assign(this, data);
  }
}

export class EventUpdateStatus {
  count?: number;
  firstAt?: Date;
  lastAt?: Date;

  constructor(data?: Partial<EventUpdateStatus>) {
    if (data) Object.assign(this, data);
  }
}

export class EventUpdate {
  meta?: MetaUpdate;
  spec?: EventUpdateSpec;
  status?: EventUpdateStatus;

  constructor(data?: Partial<EventUpdate>) {
    if (data) Object.assign(this, data);
  }
}
