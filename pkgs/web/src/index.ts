/**
 * @mir/web-sdk
 *
 * TypeScript SDK for Mir IoT Hub - Real-time device communication via NATS
 */

// Core exports (to be implemented in Phase 2)
// export { Mir } from './core/mir';
// export type { MirOptions } from './core/mir';

// Route exports (to be implemented in Phase 3)
// export type { ClientRoutes } from './routes/client';
// export type { DeviceRoutes } from './routes/device';
// export type { EventRoutes } from './routes/event';

// Header utilities (to be implemented in Phase 2)
// export { MirHeaders } from './core/headers';

// Subject builders (to be implemented in Phase 2)
// export { ClientSubject, DeviceSubject, EventSubject } from './core/subjects';

// Re-export generated proto types for convenience
export * from './gen/proto/mir_api/v1/core_pb';
export * from './gen/proto/mir_api/v1/common_pb';
export * from './gen/proto/mir_api/v1/cmd_pb';
export * from './gen/proto/mir_api/v1/cfg_pb';
export * from './gen/proto/mir_api/v1/tlm_pb';
export * from './gen/proto/mir_api/v1/device_pb';
export * from './gen/proto/mir_api/v1/event_pb';

// Package version
export const VERSION = '0.1.0';
