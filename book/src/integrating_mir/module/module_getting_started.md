# Getting Started

This guide will walk you through creating your first Mir Module. You'll learn how to connect to Mir Ecosystem and interact with the platform.

## Prerequisites

- Go 1.21 or later
- Access to [Mir Repository](../../resources/access_mir.md)
- Access to a running Mir instance
  - [Binary](../../running_mir/binary.md)
  - [Compose](../../running_mir/docker.md)

## Design

The Module SDK is a wrapper around the NatsIO Client with additional features. Similar to the DeviceSDK, it has functions that binds directly to Mir Routes.

### Installation

Add the Mir Module SDK to your Go project:

```bash
go get github.com/maxthom/mir/pkgs/module/mir
```

### Packages

Divided into two packages:

```go
// ModuleSDK
"github.com/maxthom/mir/pkgs/module/mir"
// Models
"github.com/maxthom/mir/pkgs/mir_v1"
```

## Basic Module

Let's create a simple module that monitors device connections and telemetry.

### 1. Create the Project Structure

```bash
mkdir my-first-module
cd my-first-module
go mod init my-first-module
go get github.com/maxthom/mir/pkgs/module/mir
```

### 2. Write the Code

Create `main.go`:

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    // Connect to Mir
    m, err := mir.Connect(
        "my-first-module",
        "nats://localhost:4222",
        mir.WithDefaultReconnectOpts()...,
    )
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()
    fmt.Println("Module started!")

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    fmt.Println("Shutting down...")
}
```

> `mir.Connect()` function can accept a list of NATS options to configure the connection. See [docs](https://github.com/nats-io/nats.go). Moreover, the SDK provides some common options.
> - `WithUserCredentials(...)`
> - `WithRootCA(...)`
> - `WithClientCertificate(...)`
> - `WithDefaultReconnectOpts(...)`
> - `WithDefaultConnectionLogging(...)`

### 3. Run the Module

```bash
go run main.go
```

You should see:
```
Module started!
```

## Next Steps

Now that you have a basic module running, explore more advanced features:

- [Event Subscriptions](./module_events.md) - Learn about all available events
- [Examples](./module_examples.md) - See complete working examples
- [Module SDK Overview](./module_sdk.md) - Complete API reference
