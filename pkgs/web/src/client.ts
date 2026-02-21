import type { Mir } from "./mir.js";
import {
  CreateDevice,
  DeleteDevice,
  ListDevice,
  UpdateDevice,
  CreateDevices,
} from "./server_core";
export { ClientSubject } from "./types.js";

export class ClientRoutes {
  constructor(private readonly mir: Mir) {}
  // TODO publish proto, json and raw

  listDevices(): ListDevice {
    return new ListDevice(this.mir);
  }
  deleteDevices(): DeleteDevice {
    return new DeleteDevice(this.mir);
  }
  updateDevices(): UpdateDevice {
    return new UpdateDevice(this.mir);
  }
  createDevices(): CreateDevice {
    return new CreateDevice(this.mir);
  }
  createsDevices(): CreateDevices {
    return new CreateDevices(this.mir);
  }
}
