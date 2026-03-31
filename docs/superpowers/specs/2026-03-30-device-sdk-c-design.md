# Mir Device SDK — C Design Spec

**Date:** 2026-03-30  
**Author:** MaxThom  
**Status:** Approved  

---

## Overview

A standalone C SDK (`mir-ecosystem/mir.sdk.c`) that allows IoT devices to connect to a Mir IoT Hub server. It targets both microcontrollers (MCUs) and embedded Linux, supports full feature parity with the Go SDK (including schema upload via protobuf `FileDescriptorSet`), and is callable from C++ via `extern "C"` headers with future Python bindings via `ctypes`.

---

## Goals

- Connect devices to Mir over NATS (TLS + credentials auth)
- Send telemetry and reported properties (protobuf bytes)
- Receive and handle commands and configuration updates
- Send device schema (`FileDescriptorSet`) on boot
- Heartbeat every 10 seconds
- Offline message buffering (resend on reconnect)
- Run on ESP32 (and similar MCUs with ≥64KB RAM) and embedded Linux
- Usable from C++ via `extern "C"` header guards (no separate C++ wrapper)
- Python bindings via `ctypes` against the compiled `.so`

## Non-Goals

- Sub-64KB RAM MCUs (STM32F0-class) — not in scope for v1
- Native C++ wrapper classes (RAII, templates) — deferred
- Auto-provisioning / device ID generation from hardware — deferred
- Module SDK in C — device SDK only

---

## Architecture

Four layers. Each layer only depends on the one below it.

```
┌────────────────────────────────────────────────┐
│  Public C API  (mir_device.h)                  │  ← User code
│  Device lifecycle, telemetry, commands, config  │
├────────────────────────────────────────────────┤
│  SDK Core                                      │  ← Heartbeat, routing,
│  mir_device.c / mir_handlers.c / mir_schema.c  │    schema, offline store
├────────────────────────────────────────────────┤
│  Transport  (NATS)                             │  ← nats.c on Linux,
│  mir_transport.h + platform implementations   │    custom client on MCU
├────────────────────────────────────────────────┤
│  HAL  (mir_hal.h)                              │  ← Network, time, memory,
│  linux/mir_hal_linux.c                         │    storage — swapped per
│  esp32/mir_hal_esp32.c                         │    platform at compile time
└────────────────────────────────────────────────┘
```

---

## HAL Interface

The HAL (`mir_hal.h`) is a struct of function pointers — the C equivalent of an interface. The SDK core never calls platform APIs directly; it always goes through the HAL.

```c
typedef struct {
    // Network
    int      (*net_connect)(void *ctx, const char *host, uint16_t port);
    int      (*net_write)(void *ctx, const uint8_t *buf, size_t len);
    int      (*net_read)(void *ctx, uint8_t *buf, size_t len, uint32_t timeout_ms);
    void     (*net_close)(void *ctx);

    // Time
    uint64_t (*time_ms)(void *ctx);

    // Memory
    void*    (*mem_alloc)(size_t size);
    void     (*mem_free)(void *ptr);

    // Offline store (key-value, optional — set to NULL to disable)
    int      (*store_write)(void *ctx, const char *key, const uint8_t *data, size_t len);
    int      (*store_read)(void *ctx, const char *key, uint8_t *buf, size_t *len);
    int      (*store_delete)(void *ctx, const char *key);

    void     *ctx;  // platform-specific state (socket fd, file handle, etc.)
} mir_hal_t;
```

Platforms provide pre-built HAL instances:
- `mir_hal_linux_default()` — TCP sockets, POSIX time, malloc/free, file-based store
- `mir_hal_esp32_default()` — esp_tls, FreeRTOS tick, heap, SPIFFS/LittleFS store

---

## Public API

```c
// Configuration
typedef struct {
    const char      *device_id;
    const char      *target;        // "nats://host:4222"
    const char      *credentials;   // file path to .creds file; NULL on MCU (creds embedded at compile time)
    const char      *root_ca;       // file path to ca.crt; NULL to skip server verification
    const char      *tls_cert;      // file path to tls.crt; NULL if not using mTLS
    const char      *tls_key;       // file path to tls.key; NULL if not using mTLS
    mir_log_level_t  log_level;
    mir_store_opts_t store;
} mir_config_t;

// Lifecycle
mir_device_t *mir_device_create(const mir_config_t *cfg, const mir_hal_t *hal);
void          mir_device_destroy(mir_device_t *dev);
int           mir_device_launch(mir_device_t *dev);
void          mir_device_shutdown(mir_device_t *dev);

// Schema (pre-serialized FileDescriptorSet, generated at build time)
void mir_device_set_schema(mir_device_t *dev, const uint8_t *bytes, size_t len);

// Handlers — register before launch
typedef void (*mir_cmd_handler_fn)(const uint8_t *proto_bytes, size_t len, void *ctx);
typedef void (*mir_cfg_handler_fn)(const uint8_t *proto_bytes, size_t len, void *ctx);
void mir_device_handle_command   (mir_device_t *dev, const char *msg_name, mir_cmd_handler_fn fn, void *ctx);
void mir_device_handle_properties(mir_device_t *dev, const char *msg_name, mir_cfg_handler_fn fn, void *ctx);

// Send
int mir_device_send_telemetry  (mir_device_t *dev, const char *msg_name, const uint8_t *bytes, size_t len);
int mir_device_send_properties (mir_device_t *dev, const char *msg_name, const uint8_t *bytes, size_t len);
```

All functions return `0` on success, negative errno-style codes on failure. The API is wrapped in `extern "C"` so C++ code can include it without any changes.

---

## Protobuf Strategy

**Library:** [protobuf-c](https://github.com/protobuf-c/protobuf-c)

- `protoc-gen-c` generates `.pb-c.h` / `.pb-c.c` from `.proto` files
- User fills generated C structs, packs to bytes with `my_msg__pack()`, passes bytes to SDK
- SDK treats telemetry/command payloads as opaque bytes — it does not parse them
- `msg_name` (the protobuf full name, e.g. `"my_schema.v1.MyTelemetry"`) is passed as a string alongside bytes and set as a NATS message header (matching the Go SDK's `HeaderMsgName` convention)

---

## Schema Upload (FileDescriptorSet)

On `mir_device_launch()`, the SDK sends the device's full protobuf schema to the server — identical behaviour to the Go SDK.

**Build-time flow:**

```
proto/my_schema.proto
        │
        ├─► protoc --plugin=protoc-gen-c
        │         gen/my_schema.pb-c.h   (C structs for firmware)
        │         gen/my_schema.pb-c.c   (compiled into firmware)
        │
        └─► protoc --descriptor_set_out=schema.bin
                    │
                    ▼
              tools/schema_embed/schema_embed.py
                    │
                    ▼
              gen/my_schema_bytes.h
              // auto-generated
              const uint8_t schema_bytes[] = { 0x0a, 0x2c, ... };
              const size_t  schema_len = 312;
```

`schema_embed.py` is a ~50-line Python script that reads `schema.bin` and emits the `.h` byte array. CMake runs it automatically via `mir_generate_schema()` — device developers never invoke it manually.

The schema bytes are passed to the SDK with `mir_device_set_schema()` before launch.

---

## NATS Transport

**Linux:** wraps [nats.c](https://github.com/nats-io/nats.c) — full feature set, TLS, credentials.

**MCU (custom minimal client):** NATS protocol is simple line-based text over TCP. The custom client implements:
- `CONNECT` (with auth token / credentials)
- `PUB` (publish with headers)
- `SUB` / `UNSUB`
- `MSG` (receive)
- `PING` / `PONG`
- Reconnect loop

The custom client uses the HAL for all network I/O and is approximately 500–800 lines of C. It does not support JetStream — only core NATS, which is all the Mir SDK requires.

Both transport implementations satisfy a common `mir_transport_t` interface (struct of function pointers), identical in shape to the HAL pattern.

---

## Offline Store

When disconnected, outgoing messages are buffered and replayed on reconnect — matching Go SDK behaviour.

- **Linux:** file-based queue in a configurable directory (default `~/.local/share/mir/<device_id>/`)
- **ESP32:** SPIFFS or LittleFS ring buffer via the HAL `store_*` functions
- **Disabled:** set `store_write = NULL` in the HAL to disable buffering (messages are dropped while offline)

Store is keyed by a sequence number. On reconnect, the SDK reads pending messages in order, publishes them, then deletes them.

---

## Build System

**CMake** is the primary build system. A `library.json` is provided for PlatformIO (ESP32 Arduino ecosystem).

```cmake
# User's CMakeLists.txt
find_package(mir_sdk REQUIRED)

mir_generate_schema(
    PROTO_FILES   proto/my_schema.proto
    OUTPUT_DIR    ${CMAKE_BINARY_DIR}/gen
)

target_link_libraries(my_firmware PRIVATE mir_sdk::device)
```

Platform is selected at configure time:
```bash
cmake -DMIR_PLATFORM=linux ..    # default
cmake -DMIR_PLATFORM=esp32 ..
```

The `proto/mir/device/v1/mir.proto` submodule must be initialized before calling `mir_generate_schema()`:
```bash
git submodule update --init
```

---

## Repository Structure

```
mir.sdk.c/
├── include/mir/
│   ├── mir_device.h       # public API (extern "C" wrapped)
│   ├── mir_hal.h          # HAL interface
│   └── mir_types.h        # enums, error codes
├── src/
│   ├── core/
│   │   ├── mir_device.c   # lifecycle, launch, shutdown
│   │   ├── mir_heartbeat.c
│   │   ├── mir_handlers.c # command/config routing
│   │   ├── mir_schema.c   # sends schema bytes on connect
│   │   └── mir_store.c    # offline buffering
│   ├── transport/
│   │   ├── mir_transport.h        # transport interface
│   │   ├── mir_nats_core.c        # shared subject/header logic
│   │   └── mir_nats_msg.c
│   └── platform/
│       ├── linux/
│       │   ├── mir_hal_linux.c    # sockets, file store
│       │   └── mir_nats_linux.c   # wraps nats.c
│       └── esp32/
│           ├── mir_hal_esp32.c    # esp_tls, SPIFFS
│           └── mir_nats_esp32.c   # custom minimal NATS client
├── tools/
│   └── schema_embed/
│       └── schema_embed.py        # .bin → .h byte array
├── examples/
│   ├── linux/main.c
│   └── esp32/main.c
├── bindings/
│   └── python/
│       └── mir_device.py          # ctypes wrapper
├── proto/
│   └── mir/device/v1/mir.proto    # git submodule from mir.server
├── CMakeLists.txt
├── library.json                   # PlatformIO
└── README.md
```

---

## Python Bindings

Implemented as a pure-Python `ctypes` wrapper around the compiled Linux `.so`. No compilation required by the Python user.

```python
# bindings/python/mir_device.py
import ctypes, os

_lib = ctypes.CDLL(os.path.join(os.path.dirname(__file__), "libmir_device.so"))

class MirDevice:
    def __init__(self, device_id, target, credentials=None):
        # wraps mir_device_create / mir_device_launch / etc.
        ...

    def send_telemetry(self, msg_name: str, proto_bytes: bytes): ...
    def handle_command(self, msg_name: str, fn): ...
```

Python bindings are deferred to after the C core is working — they are a thin layer with no logic of their own.

---

## Error Handling

All public API functions return `int`:
- `0` — success
- `MIR_ERR_INVALID_ARG` (-1)
- `MIR_ERR_NOT_CONNECTED` (-2)
- `MIR_ERR_TIMEOUT` (-3)
- `MIR_ERR_NO_MEMORY` (-4)
- `MIR_ERR_TRANSPORT` (-5)
- `MIR_ERR_STORE` (-6)

No exceptions, no `errno` side-effects. Errors are loggable via an optional `mir_log_fn` callback set in config.

---

## Learning Path (C Concepts by Layer)

| Build order | Layer | C concepts covered |
|---|---|---|
| 1 | HAL + types | structs, function pointers, `void *` context, header guards |
| 2 | Transport (Linux) | TCP sockets, `read`/`write`, string parsing, buffers |
| 3 | Core | `malloc`/`free`, opaque pointers, error codes, threading basics |
| 4 | Schema gen tool | Python `subprocess`, file I/O (not C) |
| 5 | ESP32 HAL | FreeRTOS tasks, `esp_tls`, flash filesystem |
| 6 | Build system | CMake, linking, shared vs static libraries |
| 7 | Python bindings | `ctypes`, ABI, shared library loading |

Each layer builds on the previous. Start with HAL + Linux transport — you get a working device on Linux before touching MCU complexity.
