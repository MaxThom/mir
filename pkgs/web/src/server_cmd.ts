import type { Mir } from "./mir";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  SendListCommandsRequestSchema,
  SendListCommandsResponseSchema,
  SendCommandRequestSchema,
  SendCommandResponseSchema,
  CommandResponseStatus,
} from "./gen/proto/mir_api/v1/cmd_pb";
import { Encoding } from "./gen/proto/mir_api/v1/common_pb";
import { ClientSubject } from "./types";
import type { DeviceTarget } from "./models";
import { deviceTargetToProto } from "./transform";

export { CommandResponseStatus } from "./gen/proto/mir_api/v1/cmd_pb";

// ─── SDK types ────────────────────────────────────────────────────────────────

type DeviceId = {
  id: string;
  name: string;
  namespace: string;
};

export type CommandDescriptor = {
  name: string;
  labels: Record<string, string>;
  template: string;
  error: string;
};

export type CommandGroup = {
  ids: DeviceId[];
  descriptors: CommandDescriptor[];
  error: string;
};

export type CommandResponse = {
  deviceId: string;
  name: string;
  payload: Uint8Array;
  status: CommandResponseStatus;
  error: string;
};

export type SendCommandResult = Map<string, CommandResponse>;

// ─── Routes ───────────────────────────────────────────────────────────────────

const listCommandsRoute = new ClientSubject("cmd", "v1alpha", "list", []);
const sendCommandRoute = new ClientSubject("cmd", "v1alpha", "send", []);

export class ListCommands {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget): Promise<CommandGroup[]> {
    const sbj = listCommandsRoute.WithId(this.mir.getInstanceName());

    const req = create(SendListCommandsRequestSchema, {
      targets: deviceTargetToProto(t),
      refreshSchema: false,
    });
    const payload = toBinary(SendListCommandsRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(SendListCommandsResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return response.response.value.devicesCommands.map((g) => ({
        ids: g.ids.map((p) => ({ id: p.deviceId, name: p.name, namespace: p.namespace })),
        descriptors: g.cmdDescriptors.map((d) => ({
          name: d.name,
          labels: d.labels as Record<string, string>,
          template: d.template,
          error: d.error,
        })),
        error: g.error,
      }));
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}

export class SendCommand {
  constructor(private readonly mir: Mir) {}

  async request(
    t: DeviceTarget,
    name: string,
    payload: string,
    dryRun: boolean,
  ): Promise<SendCommandResult> {
    const sbj = sendCommandRoute.WithId(this.mir.getInstanceName());

    const req = create(SendCommandRequestSchema, {
      targets: deviceTargetToProto(t),
      name,
      payload: new TextEncoder().encode(payload),
      payloadEncoding: Encoding.JSON,
      dryRun,
    });
    const reqPayload = toBinary(SendCommandRequestSchema, req);

    const msg = await this.mir.request(sbj, reqPayload);

    const response = fromBinary(SendCommandResponseSchema, msg.data);
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
          } satisfies CommandResponse,
        ]),
      );
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return new Map();
  }
}
