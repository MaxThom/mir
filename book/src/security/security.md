# Securing Mir

## Overview

Security is paramount in IoT deployments, where devices, data, and infrastructure must be protected from unauthorized access, data breaches, and cyber attacks. Mir IoT Hub implements a comprehensive, defense-in-depth security strategy that addresses authentication, authorization, encryption, and auditing across the entire platform.

This section provides practical guides for implementing and managing security in your Mir deployment, from development environments to production-grade zero-trust architectures.

## Security Architecture

Mir's security model is built on three foundational pillars:

### 1. **Authentication & Authorization (Auth)**
Identity verification and access control through NATS JWT-based security:
- **JWT Token Authentication**: Every entity requires signed JWT credentials
- **Role-Based Access Control (RBAC)**: Predefined roles for devices, operators, and modules
- **Granular Permissions**: Subject-based publish/subscribe authorization
- **Credential Management**: Automated rotation and distribution via NSC integration

### 2. **Transport Layer Security (TLS)**
Encryption and certificate-based authentication for all network communications:
- **Server-Only TLS**: One-way authentication with encrypted channels
- **Mutual TLS (mTLS)**: Bidirectional certificate authentication
- **Certificate Management**: Support for self-signed and CA-signed certificates
- **Flexible Configuration**: Per-environment security requirements

### 3. **Audit & Compliance**
Comprehensive logging and event tracking for security monitoring:
- **Event Store**: All system events logged to SurrealDB
- **Audit Trail**: Complete history of device actions and configuration changes
- **Monitoring Integration**: Grafana dashboards for security metrics
- **Compliance Support**: Structured logs for regulatory requirements

## Security Layers

```
┌─────────────────────────────────────────────┐
│           Application Layer                 │
│  - JWT Authentication                       │
│  - Role-Based Access Control                │
│  - Subject-Based Authorization              │
├─────────────────────────────────────────────┤
│           Transport Layer                   │
│  - TLS/mTLS Encryption                      │
│  - Certificate Validation                   │
│  - Secure NATS Messaging                    │
├─────────────────────────────────────────────┤
│           Infrastructure Layer              │
│  - Kubernetes Secrets                       │
│  - Network Policies                         │
│  - Container Security                       │
└─────────────────────────────────────────────┘
```

## Implementation Guides

### Authentication & Authorization

Implement JWT-based security for your Mir deployment:

- **[Authentication Overview](./auth.md)**: Understanding NATS security in Mir
- **[Docker Authentication](./auth-docker.md)**: Secure your Docker Compose deployment
- **[Kubernetes Authentication](./auth-kubernetes.md)**: Enterprise-grade auth for K8s

Key concepts:
- Operator hierarchy with accounts and users
- Device-specific permissions to minimize attack surface
- Client access levels (standard, read-only, swarm)
- Module permissions for server-side components

### Transport Layer Security

Encrypt all communications with TLS:

- **[TLS Overview](./tls.md)**: Choose between Server-Only and Mutual TLS
- **[Server-Only TLS](./tls-serveronly.md)**: Simpler setup for trusted networks
- **[Mutual TLS](./tls-mutual.md)**: Zero-trust security with client certificates

Decision factors:
- Development vs. production requirements
- Certificate management complexity
- Compliance and regulatory needs
- Performance considerations

## Next Steps

1. **Start with Authentication**: Begin by implementing [JWT-based authentication](./auth.md) for your deployment type

2. **Add Transport Security**: Layer on [TLS encryption](./tls.md) based on your security requirements
