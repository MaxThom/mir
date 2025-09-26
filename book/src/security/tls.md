# TLS

## Overview

Transport Layer Security (TLS) is a cryptographic protocol that provides secure communication over networks. In the Mir IoT Hub ecosystem, TLS is used to secure the NATS messaging bus, ensuring that all communication between devices, modules, and clients is encrypted and authenticated.

## Why TLS for IoT?

IoT systems like Mir face unique security challenges:
- **Device Authentication**: Ensuring only authorized devices can connect to your infrastructure
- **Data Confidentiality**: Protecting telemetry, commands, and configuration data in transit
- **Network Integrity**: Preventing man-in-the-middle attacks and data tampering
- **Compliance**: Meeting industry security standards and regulations

TLS addresses these challenges by providing encryption, authentication, and data integrity for all NATS communications in your Mir deployment.

## TLS Components

### Certificate Authority (CA)
The root of trust that signs and validates all certificates in your system. The CA certificate must be distributed to all clients to verify server authenticity.

### Server Certificate
Issued by the CA, this certificate identifies the NATS server and enables clients to verify they're connecting to the legitimate server.

### Client Certificate (mTLS only)
In Mutual TLS configurations, clients also present certificates to the server, enabling bidirectional authentication.

## Server-Only TLS

Server-Only TLS provides one-way authentication where:
- The **server** presents its certificate to prove its identity
- The **client** verifies the server's certificate using the CA
- Communication is encrypted, but clients are not authenticated via certificates

### When to Use Server-Only TLS
- Simpler deployment with fewer certificates to manage
- Clients authenticate through other means (credentials, tokens)
- Internal networks with additional security layers
- Development and testing environments

### Security Considerations
- Clients must securely store the CA certificate
- Additional authentication mechanisms needed for clients
- Suitable when client identity verification is handled at the application layer

[→ Server-Only TLS Implementation Guide](./tls-serveronly.md)

## Mutual TLS (mTLS)

Mutual TLS provides two-way authentication where:
- The **server** presents its certificate and verifies client certificates
- The **client** presents its certificate and verifies the server's certificate
- Both parties authenticate each other before establishing connection

### When to Use Mutual TLS
- Zero-trust security environments
- Production deployments with strict security requirements
- When certificate-based client authentication is preferred
- Regulatory compliance requirements (HIPAA, PCI-DSS, etc.)

### Security Benefits
- Strong bidirectional authentication without passwords
- Each client has a unique identity via its certificate
- Compromised credentials can be revoked immediately
- No shared secrets or passwords transmitted

[→ Mutual TLS Implementation Guide](./tls-mutual.md)

## Choosing Between Server-Only and Mutual TLS

| Aspect | Server-Only TLS | Mutual TLS |
|--------|----------------|------------|
| **Authentication** | Server only | Bidirectional |
| **Certificate Management** | Minimal (CA + Server) | Complex (CA + Server + All Clients) |
| **Setup Complexity** | Simple | More Complex |
| **Client Identity** | Via credentials/tokens | Via certificates |
| **Security Level** | Good | Excellent |
| **Use Case** | Development, Internal Networks, Production | Production, Zero-Trust |
| **Revocation** | N/A for clients | Per-client revocation |
| **Compliance** | Basic security requirements | Strict regulatory requirements |

## Best Practices

1. **Certificate Rotation**: Plan for regular certificate renewal before expiration
2. **Secure Storage**: Store private keys securely, never commit to version control
3. **Unique Certificates**: In mTLS, assign unique certificates per client for granular control
4. **Monitoring**: Set up alerts for certificate expiration
5. **Documentation**: Maintain clear documentation of your certificate infrastructure

## Next Steps

Choose your TLS implementation based on your security requirements:

- **[Server-Only TLS Guide](./tls-serveronly.md)**: Simpler setup with server authentication only. Ideal for development environments, internal deployments and less strict security environment.

- **[Mutual TLS Guide](./tls-mutual.md)**: Complete bidirectional authentication using certificates. Recommended for production environments and zero-trust architectures.

Both guides provide step-by-step instructions for Docker Compose and Kubernetes deployments, including certificate generation, server configuration, and client setup for CLI, Devices, and Modules.
