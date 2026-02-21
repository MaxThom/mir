/**
 * @mir/web-sdk
 *
 * TypeScript SDK for Mir IoT Hub - Real-time device communication via NATS
 */

export { Mir } from "./mir";
export type { MirOptions } from "./mir";

export * from "./models";
export * from "./server_core";
export * from "./server_event";
export * from "./client";

// Package version
export const VERSION = "0.1.0";
