# Security

## Authentication and Authorization

Mir IoT Hub implements comprehensive security through NATS authentication and authorization mechanisms. Every connection to the NATS requires valid JWT and NKeys credentials, ensuring a zero-trust security model. Security is managed with the tool **[NSC](https://docs.nats.io/using-nats/nats-tools/nsc/basics#user-authorization)**.

Nats Security is a large ecosystem with a lot of complexity, controls and features. To help secure your devices and users, Mir encapsulate some of the configuration and provides tooling to help. Nonetheless, operating Mir at a large scale would require to get more familiar with [Nats Security](https://docs.nats.io/nats-concepts/security) ecosystem and [NSC](https://docs.nats.io/using-nats/nats-tools/nsc/basics) tool.

### Key Security Features

- **JWT-based authentication** using ed25519 nkeys
- **Role-based access control** with predefined user types
- **Granular subject-based authorization**
- **Credential rotation and management** through CLI

### How NATS Security Works

#### Authentication Flow

1. **Operator Creation**: An operator (root authority) is created with signing keys
2. **Account Management**: Accounts are created under the operator for logical separation
3. **User Generation**: Users (devices, clients, modules) are created with specific permissions
4. **Credential Distribution**: JWT credentials are generated and distributed to entities
5. **Connection Validation**: NATS server validates JWT on each connection

#### Authorization Model

NATS uses subject-based permissions where each subject follows a hierarchical pattern:

```
device.{deviceId}.{module}.{version}.{function}
client.{clientId}.{module}.{version}.{function}
event.{type}.{deviceId}
```

Permissions are granted using allow/deny rules for publish (pub) and subscribe (sub) operations. To help navigate scopes, Mir offers a set of premade scopes for devices, clients and modules. Using NSC, you can create your own specific scope taylored to your need.

## TLS
