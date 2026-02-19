import type { Mir } from "./mir.js";
import { ListDevice } from "./server_core";
export { ClientSubject } from "./subjects";

export class ClientRoutes {
  constructor(private readonly mir: Mir) {}
  // TODO publish proto, json and raw

  listDevices(): ListDevice {
    return new ListDevice(this.mir);
  }
}
