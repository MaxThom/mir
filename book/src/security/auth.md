# Authentication and Authorization

Mir IoT Hub implements comprehensive security through NATS authentication and authorization mechanisms. Every connection to the NATS requires valid JWT and NKeys credentials, ensuring a zero-trust security model. Security is managed with the tool **[NSC](https://docs.nats.io/using-nats/nats-tools/nsc/basics#user-authorization)**.

Nats Security is a large ecosystem with a lot of complexity, controls and features. To help secure your devices and users, Mir encapsulate some of the configuration and provides tooling to help. Nonetheless, operating Mir at a large scale would require to get more familiar with [Nats Security](https://docs.nats.io/nats-concepts/security) ecosystem and [NSC](https://docs.nats.io/using-nats/nats-tools/nsc/basics) tool.

### Key Security Features

- **JWT-based authentication** using ed25519 nkeys
- **Role-based access control** with predefined user types
- **Granular subject-based authorization**
- **Credential rotation and management** through CLI

## How NATS Security Works

### Authentication Flow

1. **Operator Creation**: An operator (root authority) is created with signing keys
2. **Account Management**: Accounts are created under the operator for logical separation
3. **User Generation**: Users (devices, clients, modules) are created with specific permissions
4. **Credential Distribution**: JWT credentials are generated and distributed to entities
5. **Connection Validation**: NATS server validates JWT on each connection

### Authorization Model

NATS uses subject-based permissions where each subject follows a hierarchical pattern:

```
device.{deviceId}.{module}.{version}.{function}
client.{clientId}.{module}.{version}.{function}
event.{type}.{deviceId}
```

Permissions are granted using allow/deny rules for publish (pub) and subscribe (sub) operations. To help navigate scopes, Mir offers a set of premade scopes for devices, clients and modules. Using NSC, you can create your own specific scope taylored to your need.


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
--allow-pub device.{deviceId}.>   # Publish telemetry/configuration/hearthbeat
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
--allow-pub client.*.cfg.v1alpha.list   # List configurations
--allow-pub client.*.cmd.v1alpha.list   # List commands
--allow-pub client.*.tlm.v1alpha.list   # List telemetry
--allow-pub client.*.evt.v1alpha.list   # List events
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

## Next Steps

Based on your infrastructure, follow the appropriate guide:

- **[Docker Authentication Setup](./auth-docker.md)**: Step-by-step guide for securing Docker Compose deployments with JWT authentication
- **[Kubernetes Authentication Setup](./auth-kubernetes.md)**: Enterprise-grade authentication configuration for Kubernetes environments
