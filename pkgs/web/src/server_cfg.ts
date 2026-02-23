import { Mir } from "./mir";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  SendListConfigRequestSchema,
  SendListConfigResponseSchema,
  SendConfigRequestSchema,
  SendConfigResponseSchema,
  ConfigResponseStatus,
} from "./gen/proto/mir_api/v1/cfg_pb";
import { Encoding } from "./gen/proto/mir_api/v1/common_pb";
import { ClientSubject } from "./types";
import type { DeviceTarget } from "./models";
import { deviceTargetToProto } from "./transform";

export { ConfigResponseStatus } from "./gen/proto/mir_api/v1/cfg_pb";

// ─── SDK types ────────────────────────────────────────────────────────────────

export type ConfigDescriptor = {
  name: string;
  labels: Record<string, string>;
  template: string;
  error: string;
};

export type ConfigValues = {
  deviceId: string;
  values: Record<string, string>;
  error: string;
};

export type ConfigGroup = {
  descriptors: ConfigDescriptor[];
  values: ConfigValues[];
  error: string;
};

export type ConfigResponse = {
  deviceId: string;
  name: string;
  payload: Uint8Array;
  status: ConfigResponseStatus;
  error: string;
};

export type SendConfigResult = Map<string, ConfigResponse>;

// ─── Routes ───────────────────────────────────────────────────────────────────

const listConfigsRoute = new ClientSubject("cfg", "v1alpha", "list", []);
const sendConfigRoute = new ClientSubject("cfg", "v1alpha", "send", []);

export class ListConfigs {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget): Promise<ConfigGroup[]> {
    const sbj = listConfigsRoute.WithId(this.mir.getInstanceName());

    const req = create(SendListConfigRequestSchema, {
      targets: deviceTargetToProto(t),
      refreshSchema: false,
    });
    const payload = toBinary(SendListConfigRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(SendListConfigResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return response.response.value.deviceConfigs.map((g) => ({
        descriptors: g.cfgDescriptors.map((d) => ({
          name: d.name,
          labels: d.labels as Record<string, string>,
          template: d.template,
          error: d.error,
        })),
        values: g.cfgValues.map((v) => ({
          deviceId: v.id?.deviceId ?? "",
          values: v.values as Record<string, string>,
          error: v.error,
        })),
        error: g.error,
      }));
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class SendConfig {
  constructor(private readonly mir: Mir) {}

  async request(
    t: DeviceTarget,
    name: string,
    payload: string,
    dryRun: boolean,
  ): Promise<SendConfigResult> {
    const sbj = sendConfigRoute.WithId(this.mir.getInstanceName());

    const req = create(SendConfigRequestSchema, {
      targets: deviceTargetToProto(t),
      name,
      payload: new TextEncoder().encode(payload),
      payloadEncoding: Encoding.JSON,
      dryRun,
    });
    const reqPayload = toBinary(SendConfigRequestSchema, req);

    const msg = await this.mir.request(sbj, reqPayload);

    const response = fromBinary(SendConfigResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return new Map(
        Object.entries(response.response.value.deviceResponses).map(([devId, r]) => [
          devId,
          {
            deviceId: r.deviceId,
            name: r.name,
            payload: r.payload,
            status: r.status,
            error: r.error,
          } satisfies ConfigResponse,
        ]),
      );
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return new Map();
  }
}
