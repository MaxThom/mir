# Mutual TLS

## Prerequisites

Have certificates on hands:
- bring your own or use [OpenSSL](https://www.openssl-library.org/source/)

Have a mir deployment ready to be used:
- [Setup Compose Release](../running_mir/docker.md)
- [Setup Kubernetes Release](../running_mir/kubernetes.md)

## Steps

If you have your own, skip to **Step 2**.

### Step 1: Generate certificates

This will:
- generate a CA private key and certificate
  - the certificate must be installed on Mir clients (CLI, Devices, Modules)
- generate a Server private key and certificate and sign with CA
  - must be passed on Nats Message Bus
- generate multiple Client private keys and certificates and sign with CA
  - one for Mir Server
  - one for each operator (CLI)
  - one for each devices

```sh
# Generating CA private key
openssl genrsa -out ca.key 4096 2>/dev/null

# Generating CA certificate
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
    -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=Certificate Authority/CN=Mir Root CA" \
    2>/dev/null

# Generating Server private key
openssl genrsa -out tls.key 4096 2>/dev/null

# Generating Server certificate request
openssl req -new -key tls.key -out server.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=NATS Server/CN=localhost" \
    2>/dev/null

# Create extensions file for SAN (Subject Alternative Names)
# ! Add your Host or IP to the list
cat > server-ext.cnf <<EOF
subjectAltName = DNS:localhost,DNS:*.localhost,DNS:local-nats,DNS:nats,IP:127.0.0.1,IP:::1
EOF

# Sign the server certificate
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out tls.crt -extfile server-ext.cnf 2>/dev/null

# Clean up
rm -f server.csr server-ext.cnf

# Generating Mir Module client private key
openssl genrsa -out mir-module.key 4096 2>/dev/null

# Generating Mir Module client certificate request
openssl req -new -key mir-module.key -out mir-module.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=Services/CN=mir-module" \
    2>/dev/null

# Sign the Mir Module client certificate
openssl x509 -req -days 365 -in mir-module.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out mir-module.crt 2>/dev/null

# Clean up
rm -f mir-module.csr

# Generating CLI client private key
openssl genrsa -out mir-cli.key 4096 2>/dev/null

# Generating CLI client certificate request
openssl req -new -key mir-cli.key -out mir-cli.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=Operators/CN=mir-cli" \
    2>/dev/null

# Sign the CLI client certificate
openssl x509 -req -days 365 -in mir-cli.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out mir-cli.crt 2>/dev/null

# Clean up
rm -f mir-cli.csr

# Generating Device client private key
openssl genrsa -out mir-device.key 4096 2>/dev/null

# Generating Device client certificate request
openssl req -new -key mir-device.key -out mir-device.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=Mir IoT/OU=Devices/CN=mir-device-001" \
    2>/dev/null

# Sign the Device client certificate
openssl x509 -req -days 365 -in mir-device.csr \
    -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out mir-device.crt 2>/dev/null
```

### Step 2: Configure Nats Server

#### Docker

Copy `ca.crt`, `tls.crt` and `tls.key` under `./mir-compose/natsio/certs`.

In the `./mir-compose/natsio/config.conf`, update with the following:

```
# TLS/Security
tls: {
  cert_file: "/etc/nats/certs/tls.crt"
  key_file: "/etc/nats/certs/tls.key"
  # Required for mTLS
  ca_file: "/etc/nats/certs/ca.crt"
  verify: true
  timeout: 2
}
```

```
# WebSocket Configuration with TLS
websocket: {
  listen: 0.0.0.0:9222
  tls: {
    cert_file: "/etc/nats/certs/tls.crt"
    key_file:  "/etc/nats/certs/tls.key"
  }
  no_tls: false
  compress: true
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

Create two Kubernetes Secrets, CA and TLS:

```bash
# Directly to the cluster
kubectl create secret tls nats-tls-secret --cert=tls.crt --key=tls.key
kubectl create secret generic nats-ca-secret --from-file=ca.crt=ca.crt
# As file
kubectl create secret tls nats-tls-secret --cert=tls.crt --key=tls.key -o yaml --dry-run=client > nats-tls.secret.yaml
kubectl create secret generic nats-ca-secret --from-file=ca.crt=ca.crt -o yaml --dry-run=client > nats-ca.secret.yaml
```

Update values file:

```yaml
## NATS
nats:
  config:
    nats:
      tls:
        enabled: true
        secretName: nats-tls-secret
        dir: /etc/nats-certs
        cert: tls.crt
        key: tls.key
        merge:
          verify: true # True for Mutual TLS, false for ServerOnly TLS
          timeout: 2
  # Reference CA for mTLS
  tlsCA:
    enabled: true
    secretName: nats-ca-secret
    dir: /etc/nats-ca-certs
    key: ca.crt
```

### Step 3: Configure Cockpit HTTPS (Optional)

By default Cockpit serves on plain HTTP. If you want the web UI over HTTPS (so the browser uses `wss://` for its NATS WebSocket connection), configure it in `./mir-compose/mir/local-config.yaml`:

```yaml
mir:
  http:
    port: 3015
    tlsCert: "/home/mir/certs/tls.crt"
    tlsKey: "/home/mir/certs/tls.key"
```

Mount the certs into the Mir container in `./mir-compose/mir/compose.yaml`:

```yaml
services:
  mir:
    volumes:
      - ./local-config.yaml:/home/mir/.config/mir/mir.yaml
      - ./local-contexts.yaml:/home/mir/.config/mir/cli.yaml
      - ./certs:/home/mir/certs
```

Update `./mir-compose/mir/local-contexts.yaml` so Cockpit connects over secure WebSocket:

```yaml
contexts:
  - name: local
    target: nats+tls://localhost:4222
    webTarget: wss://localhost:9222 # Note 'wss' for secure connection
    grafana: localhost:3000
```

Restart: `docker compose down && docker compose up`. Cockpit is now available at `https://localhost:3015`.

### Step 4: Install Certificates on Module

#### Docker

Edit `./mir-compose/mir/local-config.yaml` with the RootCA, certificate and private key paths under `nats.*`. Mount the files in `./mir-compose/mir/compose.yaml`.

```yaml
nats:
  url: "nats+tls://local_mir_support-nats-1:4222"
  rootCA: "/home/mir/certs/ca.crt"
  tlsCert: "/home/mir/certs/mir-module.crt"
  tlsKey: "/home/mir/certs/mir-module.key"
```

```bash
# Restart server
docker compose down
docker compose up
```

#### Kubernetes

Create two Kubernetes Secret, CA and TLS:

```bash
# Directly to the cluster
kubectl create secret tls mir-tls-secret --cert=tls.crt --key=tls.key
kubectl create secret generic mir-ca-secret --from-file=ca.crt=ca.crt
# As file
kubectl create secret tls mir-tls-secret --cert=tls.crt --key=tls.key -o yaml --dry-run=client > mir-tls.secret.yaml
kubectl create secret generic mir-ca-secret --from-file=ca.crt=ca.crt -o yaml --dry-run=client > mir-ca.secret.yaml
```

Update values file:

```yaml
## MIR
caSecretRef: mir-ca-secret
tlsSecretRef: mir-tls-secret
```

### Step 5: Install the Certificates on the Clients

#### CLI

Edit CLI configuration file to add the RootCA `mir tools config edit`:

```yaml
- name: local
  target: nats+tls://localhost:4222
  webTarget: wss://localhost:9222
  grafana: localhost:3000
  sec:
    rootCA: <path>/ca.crt
    tlsKey: <path>/mir-cli.key
    tlsCert: <path>/mir-cli.crt
```

Run `mir dev ls` to validate.

#### Device

There are a few options to load the RootCA and Certificate files with the DeviceSDK.

```go
# Using Builder with fix path
device := mir.Builder().
    RootCA("/<path>/ca.crt").
    ClientCertificateFile("/<path>/mir-device.crt", "/<path>/mir-device.key").
    DeviceId("dev1").
    Build()
# Using Builder with default lookup
#   ./ca.crt
#   ~/.config/mir/ca.crt
#   /etc/mir/ca.crt
#
#   ./tls.crt & ./tls.key
#   ~/.config/mir/tls.crt & ~/.config/mir/tls.key
#   /etc/mir/tls.crt & /etc/mir/tls.key
device := mir.NewDevice().
    Target("nats://nats.example.com:4222").
    DefaultRootCAFile().
    DefaultClientCertificateFile().
    Build()
```

It is also possible to load the RootCA from the config file:

```go
# Using Builder with config file
device := mir.Builder().
    DefaultConfigFile().
    Build()
```

```yaml
mir:
  rootCA: "<path>/ca.crt"
  tlsKey: "<path>/mir-device.key"
  tlsCert: "<path>/mir-device.crt"
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

Congratulation, you now have Mutual TLS configured.
