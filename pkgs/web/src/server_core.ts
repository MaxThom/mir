import type { Mir } from "./mir";
import {
  type JsonObject,
  create,
  toBinary,
  fromBinary,
} from "@bufbuild/protobuf";
import {
  ListDeviceRequestSchema,
  ListDeviceResponseSchema,
  DeleteDeviceRequestSchema,
  DeleteDeviceResponseSchema,
  UpdateDeviceRequestSchema,
  UpdateDeviceResponseSchema,
  UpdateDeviceRequest_MetaSchema,
  UpdateDeviceRequest_SpecSchema,
  UpdateDeviceRequest_PropertiesSchema,
  CreateDeviceRequestSchema,
  CreateDeviceResponseSchema,
  NewDeviceSchema,
  CreateDevicesRequestSchema,
  CreateDevicesResponseSchema,
} from "./gen/proto/mir_api/v1/core_pb";
import { OptStringSchema } from "./gen/proto/mir_api/v1/common_pb";
import { ClientSubject } from "./types";
import type { Device, DeviceTarget } from "./models";
import {
  deviceTargetToProto,
  deviceFromProto,
  devicesFromProto,
} from "./transform";

const listDeviceRoute = new ClientSubject("core", "v1alpha", "list", []);
const deleteDeviceRoute = new ClientSubject("core", "v1alpha", "delete", []);
const updateDeviceRoute = new ClientSubject("core", "v1alpha", "update", []);
const createDeviceRoute = new ClientSubject("core", "v1alpha", "create", []);
const createDevicesRoute = new ClientSubject("core", "v1alpha", "creates", []);

export class ListDevice {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget, includeEvents: boolean): Promise<Device[]> {
    const sbj = listDeviceRoute.WithId(this.mir.getInstanceName());

    const req = create(ListDeviceRequestSchema, {
      targets: deviceTargetToProto(t),
      includeEvents,
    });
    const payload = toBinary(ListDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(ListDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return devicesFromProto(response.response.value.devices);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class DeleteDevice {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget): Promise<Device[]> {
    const sbj = deleteDeviceRoute.WithId(this.mir.getInstanceName());

    const req = create(DeleteDeviceRequestSchema, {
      targets: deviceTargetToProto(t),
    });
    const payload = toBinary(DeleteDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(DeleteDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return devicesFromProto(response.response.value.devices);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class UpdateDevice {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget, d: Device): Promise<Device[]> {
    const sbj = updateDeviceRoute.WithId(this.mir.getInstanceName());

    const req = create(UpdateDeviceRequestSchema, {
      targets: deviceTargetToProto(t),
      meta: create(UpdateDeviceRequest_MetaSchema, {
        name: d.meta.name,
        namespace: d.meta.namespace,
        labels: Object.fromEntries(
          Object.entries(d.meta.labels).map(([k, v]) => [
            k,
            create(
              OptStringSchema,
              v.toLowerCase() === "null" ? {} : { value: v },
            ),
          ]),
        ),
        annotations: Object.fromEntries(
          Object.entries(d.meta.annotations).map(([k, v]) => [
            k,
            create(
              OptStringSchema,
              v.toLowerCase() === "null" ? {} : { value: v },
            ),
          ]),
        ),
      }),
      spec: create(UpdateDeviceRequest_SpecSchema, {
        deviceId: d.spec.deviceId || undefined,
        disabled: d.spec.disabled,
      }),
      props:
        d.properties.desired !== undefined
          ? create(UpdateDeviceRequest_PropertiesSchema, {
              desired: d.properties.desired as JsonObject,
            })
          : undefined,
    });
    const payload = toBinary(UpdateDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(UpdateDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return devicesFromProto(response.response.value.devices);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }

  async requestSingle(d: Device): Promise<Device[]> {
    const sbj = updateDeviceRoute.WithId(this.mir.getInstanceName());
    const t: DeviceTarget = {
      ids: [d.spec.deviceId],
      names: [],
      namespaces: [],
      labels: {},
      schemas: [],
    };

    const req = create(UpdateDeviceRequestSchema, {
      targets: deviceTargetToProto(t),
      meta: create(UpdateDeviceRequest_MetaSchema, {
        name: d.meta.name,
        namespace: d.meta.namespace,
        labels: Object.fromEntries(
          Object.entries(d.meta.labels).map(([k, v]) => [
            k,
            create(
              OptStringSchema,
              v.toLowerCase() === "null" ? undefined : { value: v },
            ),
          ]),
        ),
        annotations: Object.fromEntries(
          Object.entries(d.meta.annotations).map(([k, v]) => [
            k,
            create(
              OptStringSchema,
              v.toLowerCase() === "null" ? undefined : { value: v },
            ),
          ]),
        ),
      }),
      spec: create(UpdateDeviceRequest_SpecSchema, {
        deviceId: d.spec.deviceId || undefined,
        disabled: d.spec.disabled,
      }),
      props:
        d.properties.desired !== undefined
          ? create(UpdateDeviceRequest_PropertiesSchema, {
              desired: d.properties.desired as JsonObject,
            })
          : undefined,
    });
    const payload = toBinary(UpdateDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(UpdateDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return devicesFromProto(response.response.value.devices);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class CreateDevice {
  constructor(private readonly mir: Mir) {}

  async request(d: Device): Promise<Device> {
    const sbj = createDeviceRoute.WithId(this.mir.getInstanceName());

    const req = create(CreateDeviceRequestSchema, {
      device: create(NewDeviceSchema, {
        meta: d.meta,
        spec: { deviceId: d.spec.deviceId, disabled: d.spec.disabled ?? false },
      }),
    });
    const payload = toBinary(CreateDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(CreateDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return deviceFromProto(response.response.value);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return deviceFromProto(undefined);
  }
}

export class CreateDevices {
  constructor(private readonly mir: Mir) {}

  async request(d: Device[]): Promise<Device[]> {
    const sbj = createDevicesRoute.WithId(this.mir.getInstanceName());

    const req = create(CreateDevicesRequestSchema, {
      devices: d.map((device) =>
        create(NewDeviceSchema, {
          meta: device.meta,
          spec: {
            deviceId: device.spec.deviceId,
            disabled: device.spec.disabled ?? false,
          },
        }),
      ),
    });
    const payload = toBinary(CreateDevicesRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(CreateDevicesResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return devicesFromProto(response.response.value.devices);
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}
