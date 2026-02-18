import { Mir } from "./mir";
import {
  DeviceTarget,
  Device,
  ListDeviceRequestSchema,
  ListDeviceResponseSchema,
} from "./gen/proto/mir_api/v1/core_pb";
import { ClientSubject } from "./client";
import { toBinary, fromBinary, create } from "@bufbuild/protobuf";

const listDeviceRoute = new ClientSubject("core", "v1alpha", "listDevices", []);

export class ListDevice {
  constructor(private readonly mir: Mir) {}

  async request(t: DeviceTarget, includeEvents: boolean): Promise<Device[]> {
    const sbj = listDeviceRoute.WithId(this.mir.getInstanceName());

    const req = create(ListDeviceRequestSchema, {
      targets: t,
      includeEvents: includeEvents,
    });
    const payload = toBinary(ListDeviceRequestSchema, req);

    const msg = await this.mir.request(sbj, payload);

    const response = fromBinary(ListDeviceResponseSchema, msg.data);
    if (response.response.case === "ok") {
      return response.response.value.devices;
    } else if (response.response.case === "error") {
      throw new Error(response.response.value);
    }
    return [];
  }
}
