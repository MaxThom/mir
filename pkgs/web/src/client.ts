import type { Mir } from "./mir.js";
import {
  CreateDevice,
  DeleteDevice,
  ListDevice,
  UpdateDevice,
  CreateDevices,
} from "./server_core";
import { ListEvent } from "./server_event";
import { ListCommands, SendCommand } from "./server_cmd";
import { ListConfigs, SendConfig } from "./server_cfg";
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
  listEvents(): ListEvent {
    return new ListEvent(this.mir);
  }
  listCommands(): ListCommands {
    return new ListCommands(this.mir);
  }
  sendCommand(): SendCommand {
    return new SendCommand(this.mir);
  }
  listConfigs(): ListConfigs {
    return new ListConfigs(this.mir);
  }
  sendConfig(): SendConfig {
    return new SendConfig(this.mir);
  }
}
