# @mir/web-sdk

TypeScript SDK for Mir IoT Hub - Real-time device communication via NATS WebSocket

## Overview

The Mir Web SDK provides a comprehensive TypeScript API for interacting with the Mir IoT Hub platform from browser and Node.js applications. It mirrors the architecture of the Go ModuleSDK, enabling real-time device management, telemetry streaming, and event subscriptions.

## Features

- **Real-time Device Management**: Create, read, update, and delete devices via NATS messaging
- **Live Telemetry Streams**: Subscribe to device telemetry in real-time
- **Event System**: Pub/sub for device lifecycle events (online/offline, created/updated/deleted)
- **Command & Configuration**: Send commands and configurations to devices
- **Protocol Buffers**: Efficient binary serialization using generated TypeScript types
- **Compression**: Optional zstd compression for large messages
- **WebSocket Support**: Browser-compatible NATS WebSocket client

## Installation

```bash
npm install @mir/web-sdk
```

## Quick Start

```typescript
import { Mir } from '@mir/web-sdk';

// Connect to Mir NATS server
const mir = await Mir.connect({
  name: 'my-app',
  servers: ['ws://localhost:9222'],
  jwt: 'your-jwt-token',
  nkeySeed: 'your-nkey-seed'
});

// List devices
const devices = await mir.client().listDevices({
  targets: { ids: [], labels: {}, namespaces: [], names: [] },
  includeEvents: false
});

console.log('Devices:', devices);

// Subscribe to device created events
mir.event().subscribeDeviceCreated((msg, deviceId, device) => {
  console.log('New device created:', deviceId, device);
});

// Cleanup
await mir.disconnect();
```

## Architecture

The SDK is organized into three main route categories, mirroring the Go ModuleSDK:

### Client Routes
Operations initiated by clients (web apps, backend services):
- Device CRUD (create, list, update, delete)
- Telemetry queries
- Command execution
- Configuration management
- Event queries

### Device Routes
Device-side operations (typically used by IoT devices):
- Heartbeat publishing
- Telemetry streaming
- Schema reporting
- Property synchronization

### Event Routes
Event pub/sub system:
- Device lifecycle events (online, offline, created, updated, deleted)
- Command events
- Configuration events (desired/reported properties)

## NATS Subject Patterns

The SDK uses structured NATS subjects:

- **Client operations**: `client.{clientId}.{module}.{version}.{function}`
- **Device streams**: `device.{deviceId}.{module}.{version}.{function}`
- **Event streams**: `event.{eventId}.{module}.{version}.{function}`

## Development

### Build

```bash
npm run build
```

### Watch Mode

```bash
npm run dev
```

### Test

```bash
npm test
```

### Clean

```bash
npm run clean
```

## Generated Code

This SDK uses Protocol Buffer definitions from `pkgs/api/proto/mir_api/v1/`. TypeScript code is generated using Buf:

```bash
# From repository root
buf generate --template buf.gen.web.yaml
```

Generated files are located in `src/gen/proto/`.

## Dependencies

- **@bufbuild/protobuf**: Modern protobuf library with tree-shaking support
- **nats.ws**: Official NATS WebSocket client
- **fzstd**: Pure JavaScript zstd compression (no WASM)

## License

MIT

## Related

- [Mir IoT Hub](https://github.com/maxthom/mir)
- [Go ModuleSDK](../../module/mir)
- [Protocol Definitions](../../api/proto)
