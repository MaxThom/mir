import type { Mir } from "./mir.js";
import { ListDevice } from "./server_core";

export class ClientSubject {
  private rts: string[] = [];
  constructor(
    public readonly module: string,
    public readonly version: string,
    public readonly fn: string,
    public readonly extra: string[],
  ) {
    this.rts = ["client", "*", module, version, fn, ...extra];
  }
  String(): string {
    return this.rts.join(".");
  }
  WithId(id: string): string {
    const newRts = [...this.rts];
    newRts[1] = id;
    return newRts.join(".");
  }
}

export class ClientRoutes {
  constructor(private readonly mir: Mir) {}
  // TODO publish proto, json and raw

  listDevices(): ListDevice {
    return new ListDevice(this.mir);
  }
}
