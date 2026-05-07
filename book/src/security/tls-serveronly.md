# Server Only TLS

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
- generate a Server private key and certificate.
  - must be passed on Nats Message Bus
- sign the Server certificate with the CA

```sh
# Generating CA private key
openssl genrsa -out ca.key 4096

# Generating CA certificate
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
    -subj "/C=US/ST=CA/L=San Francisco/O=NATS Demo/OU=Certificate Authority/CN=NATS Root CA" \
    2>/dev/null

# Generating Server private key
openssl genrsa -out tls.key 4096

# Generating Server certificate
openssl req -new -key tls.key -out server.csr \
    -subj "/C=US/ST=CA/L=San Francisco/O=NATS Demo/OU=NATS Server/CN=localhost" \
    2>/dev/null

# Create extensions file for SAN (Subject Alternative Names)
# Make sure to put the address of your server
cat > server-ext.cnf <<EOF
subjectAltName = DNS:localhost,DNS:*.localhost,DNS:local-nats,IP:127.0.0.1,IP:::1
EOF

# Sign the server certificate
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
    -CAcreateserial -out tls.crt -extfile server-ext.cnf 2>/dev/null
```

### Step 2: Configure Nats Server

#### Docker

Copy `tls.crt` and `tls.key` under `./mir-compose/natsio/certs`.

In the `./mir-compose/natsio/config.conf`, update with the following:

```
# TLS/Security
tls: {
  cert_file: "/etc/nats/certs/tls.crt"
  key_file: "/etc/nats/certs/tls.key"
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

### Step 3: Configure Cockpit HTTPS (Optional)

By default Cockpit serves on plain HTTP. If you want the web UI over HTTPS (so the browser uses `wss://` for its NATS WebSocket connection), follow the steps for your deployment.

#### Docker

Configure in `./mir-compose/mir/local-config.yaml`:

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
    webTarget: wss://localhost:9222 # Note 'wss' to enable secured connection
    grafana: localhost:3000
```

Restart: `docker compose down && docker compose up`. Cockpit is now available at `https://localhost:3015`.

#### Kubernetes

**Option A and B are mutually exclusive.** With ingress TLS (Option A), NATS WebSocket TLS is terminated at the ingress. Without ingress (Option B), NATS WebSocket must serve TLS directly.

**Option A - Via Ingress (recommended)**

The ingress controller terminates TLS. Mir and NATS run plain HTTP/WS internally.

```bash
kubectl create secret tls mir-http-tls-secret --cert=tls.crt --key=tls.key
```

```yaml
ingress:
  enabled: true
  className: "traefik"  # or "nginx", etc.
  hosts:
    - host: mir.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mir-http-tls-secret
      hosts:
        - mir.example.com

nats:
  config:
    websocket:
      ingress:
        enabled: true
        hosts:
          - mir.example.com
        path: /nats-ws
        pathType: Prefix
        className: "traefik"
        tlsSecretName: mir-http-tls-secret

config:
  contexts:
    - name: "production"
      webTarget: "wss://mir.example.com/nats-ws"
```

**Option B - Via App (no ingress)**

Mir and NATS serve TLS directly. Use when there is no ingress controller.

```bash
kubectl create secret tls mir-http-tls-secret --cert=tls.crt --key=tls.key
kubectl create secret tls nats-ws-tls-secret --cert=tls.crt --key=tls.key
```

```yaml
mirHttpTlsSecretRef: mir-http-tls-secret

nats:
  config:
    websocket:
      tls:
        enabled: true
        secretName: nats-ws-tls-secret
        dir: /etc/nats-certs/websocket
        cert: tls.crt
        key: tls.key

config:
  contexts:
    - name: "production"
      webTarget: "wss://<host>:31922"  # NATS WebSocket NodePort
```

Apply with Helm:

```bash
helm upgrade <release-name> <chart-path> -f values.yaml
```

### Step 4: Install Root Certificate on Module

If the CA Certificate is public and installed in the Trusted Store of your container, you can skip this step.

#### Docker

Edit `./mir-compose/mir/local-config.yaml` and set the root CA path under `nats.rootCA`. Mount the file in `./mir-compose/mir/compose.yaml`.

```yaml
nats:
  url: "nats+tls://local_mir_support-nats-1:4222"
  rootCA: "/home/mir/certs/ca.crt"
```

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
caSecretRef: mir-rootca-secret # Secret name
```

### Step 5: Install the Root Certificate on the Clients

If the CA Certificate is public and installed in the Trusted Store of your machine, you can skip this step.

#### Via the Trusted Store

If installed in the Trusted Store, Applications automaticly use them to identify servers.

Each OS has different install location and steps. Describing each one of them is out the scope of this documentaton.

Steps for ArchLinux:

```bash
# 1. Copy CA to anchors
sudo cp ca.crt /etc/ca-certificates/trust-source/anchors/nats-ca.crt
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
  webTarget: wss://localhost:9222
  grafana: localhost:3000
  sec:
    rootCA: <path>/ca.crt
```

#### Device

There are a few options to load the RootCA file with the DeviceSDK.

```go
# Using Builder with fix path
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    WithRootCA("/<path>/ca.crt").
    WithDeviceId("dev1").
    Build()
# Using Builder with default lookup
#   ./ca.crt
#   ~/.config/mir/ca.crt
#   /etc/mir/ca.crt
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
  rootCA: "<path>/ca.crt"
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
