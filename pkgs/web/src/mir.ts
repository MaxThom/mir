import {
  createInbox,
  wsconnect,
  Msg,
  MsgHdrsImpl,
  RequestOptions,
} from "@nats-io/nats-core";
import type {
  NatsConnection,
  WsConnectionOptions,
  MsgHdrs,
} from "@nats-io/nats-core";
import { ClientRoutes } from "./client";
import { HeaderTrigger } from "./constants";

export interface MirOptions extends WsConnectionOptions {}

/**
 * Entry point for the Mir TypeScript SDK.
 * Mirrors the Go ModuleSDK Mir struct in pkgs/module/mir/mir.go.
 *
 * Usage:
 *   const mir = await Mir.connect('my-client', { servers: 'ws://localhost:9222' })
 *   const devices = await mir.core().listDevices()
 *   await mir.disconnect()
 */
export class Mir {
  private constructor(
    private readonly bus: NatsConnection,
    private readonly name: string,
    private readonly instanceName: string,
  ) {}

  // TODO: opts builder patterns like go sdk

  static async connect(
    name: string,
    target: string,
    opts: MirOptions,
  ): Promise<Mir> {
    if (!opts) {
      opts = {};
    }

    if (!opts.name) {
      opts.name = name;
    }
    if (!opts.servers) {
      opts.servers = target;
    }

    const nc = await wsconnect(opts);
    return new Mir(nc, name, createInbox().substring(7, 14));
  }

  /** Drain the connection and wait for all pending messages to be flushed. */
  async disconnect(): Promise<void> {
    await this.bus.drain();
  }

  getInstanceName(): string {
    return this.name + "-" + this.instanceName;
  }

  publish(subject: string, payload?: Uint8Array, headers?: MsgHdrs): void {
    this.bus.publish(subject, payload, { headers });
  }

  request(
    subject: string,
    payload?: Uint8Array,
    headers?: MsgHdrs,
    timeout: number = 30000,
  ): Promise<Msg> {
    if (!headers) {
      headers = new MsgHdrsImpl();
    }
    headers.set(HeaderTrigger, this.getInstanceName());

    const reqOpts: RequestOptions = {
      headers: headers,
      timeout: timeout,
    };

    console.log(headers);
    return this.bus.request(subject, payload, reqOpts);
  }

  /** Access core device-management routes (create, list, update, delete). */
  client(): ClientRoutes {
    return new ClientRoutes(this);
  }
}
