# Server Only TLS

## Prerequisites

Have certificates on hands:
- bring your own or use [OpenSSL](https://www.openssl-library.org/source/)

Have a mir deployment ready to be used:
- [Setup Compose Release](../running_mir/docker.md)

## Steps

If you have your own, skip to **Step 2**.

### Step 1: Generate certificates

This will:
- generate a CA private key and certificate
  - must be installed on Mir clients (CLI, Devices, Modules)
- generate a Server private key and certificate.
  - must be passed on Nats Message Bus
- sign the Server certificate with the CA

```sh
# Generating CA private key
openssl genrsa -out ca-key.pem 4096

# Generating CA certificate
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem \
    -subj "/C=US/ST=CA/L=San Francisco/O=NATS Demo/OU=Certificate Authority/CN=NATS Root CA" \
    2>/dev/null

# Generating Server private key
openssl genrsa -out server-key.pem 4096

# Generating Server certificate
openssl req -new -key server-key.pem -out server.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=NATS Demo/OU=NATS Server/CN=localhost" \
    2>/dev/null

# Create extensions file for SAN (Subject Alternative Names)
# Make sure to put the address of your server
cat > server-ext.cnf <<EOF
subjectAltName = DNS:localhost,DNS:*.localhost,DNS:local-nats,IP:127.0.0.1,IP:::1
EOF

# Sign the server certificate
openssl x509 -req -days 365 -in server.csr -CA ca-cert.pem -CAkey ca-key.pem \
    -CAcreateserial -out server-cert.pem -extfile server-ext.cnf 2>/dev/null
```

### Step 2: Configure Nats Server

#### Docker

Copy `server-cert.pem` and `server-key.pem` under `./mir-compose/natsio/certs`.

In the `./mir-compose/natsio/config.conf`, update with the following:

```
# TLS/Security
tls: {
  cert_file: "/etc/nats/certs/server-cert.pem"
  key_file: "/etc/nats/certs/server-key.pem"
  timeout: 2
}
```

Update Compose to pass the certificates:

```yaml
services:
  nats:
    ...
    volumes:
      ...
      - ./certs:/etc/nats/certs
    ...
```

Start server `docker compose up`.

#### Kubernetes

Create a Kubernetes Secret with the TLS:

```bash
# Directly to the cluster
kubectl create secret tls mir-tls-secret --cert=tls.crt --key=tls.key
# As file
kubectl create secret tls mir-tls-secret --cert=tls.crt --key=tls.key -o yaml --dry-run=client > mir-tls.secret.yaml
```

Update values file:

```yaml
## Nats
nats:
  config:
    nats:
      tls:
        enabled: true
        secretName: mir-tls-secret # Secret name
        dir: /etc/nats-certs/nats
        cert: tls.crt
        key: tls.key
```

### Step 3: Install Root Certificate on Module

If the CA Certificate is public and installed in the Trusted Store of your container, you can skip this step.

#### Docker

Let's launch the server with the RootCA file. Edit `./mir-compose/mir/local-config.yaml` and set the path of the credentials files under `mir.rootCA`. Edit `./mir-compose/mir/compose.yaml` to mount the file.

```bash
# Restart server
docker compose down
docker compose up
```

#### Kubernetes

Create a Kubernetes Secret with the RootCA:

```bash
# Directly to the cluster
kubectl create secret generic mir-rootca-secret --from-file=ca.crt
# As file
kubectl create secret generic mir-rootca-secret --from-file=ca.crt -o yaml --dry-run=client > mir-rootca.secret.yaml
```

Update values file:

```yaml
## MIR
rootCASecretRef: mir-rootca-secret # Secret name
```

### Step 4: Install the Root Certificate on the Clients

If the CA Certificate is public and installed in the Trusted Store of your machine, you can skip this step.

#### Via the Trusted Store

If installed in the Trusted Store, Applications automaticly use them to identify servers.

Each OS has different install location and steps. Describing each one of them is out the scope of this documentaton.

Steps for ArchLinux:

```bash
# 1. Copy CA to anchors
sudo cp ca-cert.pem /etc/ca-certificates/trust-source/anchors/nats-ca.crt
# 2. Ensure correct permissions
sudo chmod 644 /etc/ca-certificates/trust-source/anchors/nats-ca.crt
# 3. Update trust database
sudo update-ca-trust extract
# 4. Verify
trust list | grep "NATS Root CA"
```

#### Via Configuration

If you prefer not too use the Trusted Store, you can pass the CA certificate directly in the Mir applications.

#### CLI

Edit CLI configuration file to add the RootCA `mir tools config edit`:

```yaml
- name: local
  target: nats://localhost:4222
  grafana: localhost:3000
  rootCA: <path>/ca-cert.pem
```

#### Device

There are a few options to load the RootCA file with the DeviceSDK.

```go
# Using Builder with fix path
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    WithRootCA("/<path>/ca-cert.pem").
    WithDeviceId("dev1").
    Build()
# Using Builder with default lookup
#   ./ca-cert.pem
#   ~/.config/mir/ca-cert.pem
#   /etc/mir/ca-cert.pem
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    DefaultRootCA().
    Build()
```

It is also possible to load the RootCA from the config file:

```go
# Using Builder with config file
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    DefaultConfigFile().
    Build()
```

```yaml
mir:
  rootCA: "<path>/ca-cert.pem"
  device:
    id: "dev1"
```

Run the device and no TLS errors should be displayed. Now run `mir dev ls` and you should see:

```bash
➜ mir dev ls
NAMESPACE/NAME                                DEVICE_ID        STATUS     LAST_HEARTHBEAT      LAST_SCHEMA_FETCH    LABELS
default/dev1                                  dev1             online     2025-09-18 16:16:27  2025-09-18 16:15:18
```

## Completed

Congratulation, you now have ServerOnly TLS configured.
