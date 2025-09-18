# Authentication and Authorization

This guide explains how NATS security works in Mir and provides practical tutorials for securing your IoT infrastructure. Refer to [Security](../concepts/security.md) for a basic overview.

## NSC Integration

Mir provides seamless integration with NSC (NATS Security CLI) to simplify credential management. The integration is exposed through the `mir tools security` command suite.

Operating Mir at a large scale would require to get more familiar with [Nats Security](https://docs.nats.io/nats-concepts/security) ecosystem and [NSC](https://docs.nats.io/using-nats/nats-tools/nsc/basics) tool.

### NSC Architecture

NSC manages a hierarchical structure of operators, accounts, and users:

```
Operator (Root Authority)
├── System Account (Internal Operations)
└── Default Account (Mir Operations)
    ├── Device Users
    │   ├── sensor001
    │   ├── sensor002
    │   └── ...
    ├── Client Users
    │   ├── operator-alice
    │   ├── viewer-bob
    │   └── ...
    └── Module Users
        ├── core
        ├── prototlm
        └── ...
```

### NSC Commands via Mir CLI

Mir wraps NSC functionality with simplified commands:
- Initialize operators
- Generate Server configuration
- Add Users with predefined permissions
- Generate credential files

For advanced scenarios, you can use NSC directly.

### User Types and Permissions

Mir defines three primary user types with specific permission sets:

#### 1. Device Users
Devices have restricted permissions to prevent compromise:

```bash
# Device-specific permissions
--allow-pubsub _INBOX.>           # Required for request-reply
--allow-pub device.{deviceId}.>   # Publish telemetry/configuration/heathbeat
--allow-sub {deviceId}.>          # Receive commands/config
```

#### 2. Client Users
Three levels of client access:

**Standard Client:**
```bash
--allow-pubsub _INBOX.>
--allow-pub client.*.>  # All client operations
```

**Read-Only Client:**
```bash
--allow-pubsub _INBOX.>
--allow-pub client.*.core.v1alpa.list  # List devices only
--allow-pub client.*.cfg.v1alpa.list   # List configurations
--allow-pub client.*.cmd.v1alpa.list   # List commands
--allow-pub client.*.tlm.v1alpa.list   # List telemetry
--allow-pub client.*.evt.v1alpa.list   # List events
```

**Swarm Client (Development or Demo):**
```bash
--allow-pubsub _INBOX.>
--allow-pub client.*.>  # Client operations
--allow-pub device.*.>  # Device simulation
--allow-sub *.>
```

#### 3. Module Users
Server-side modules with comprehensive access:

```bash
--allow-pubsub _INBOX.>
--allow-pubsub client.*.>  # Handle client requests
--allow-pubsub event.*.>   # Process events
--allow-sub device.*.>     # Monitor device data
--allow-pub *.>            # System-wide publishing
```
