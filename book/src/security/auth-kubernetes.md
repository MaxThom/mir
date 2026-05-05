# Securing your Kubernetes deployment

## Prerequisites

Install required security tools:

- [NSC](https://docs.nats.io/using-nats/nats-tools/nsc)
- `mir tools install`

Have a mir deployment ready to be used:
- [Setup Kubernetes Release](../running_mir/kubernetes.md)

## Setup

Mir Security CLI wraps NSC commands with a set of preset to make securing Mir ecoystem easier. It offers a set of basic commands to manipulate credentials. Moreover, it offers premade scope for the three types of users in Mir:

- Modules (server components)
- Clients (access CLI and other frontend)
- Devices (connect devices)

The CLI uses the current context to help manage which server to target. Use `mir tools config edit` to add a new context with server name and url:

```yaml
# If using local k3d
- name: k3d
  target: nats://localhost:31422
  grafana: grafana-local:8081
```

You can overwrite the Operator, Account and URL arguments using flags. This requires more familiarity with the NSC tool. Moreover, you can use flag `--no-exec` on each command to see NSC commands.

### Step 1: Initialize Mir Operator

Create a new NATS operator for your deployment:

```bash
mir tools security init
```

This creates:
- Operator signing keys
- Default account named Mir
- System account for internal operations

### Step 2: Configure NATS Server

Generate Resolver Configuration used to launch Nats Server:

```bash
# Generate NATS resolver configuration
mir tools security generate-resolver -p ./resolver.conf
```

#### Using Values file

Replace the OPERATOR_JWT, SYS_ACCOUNT_ID, and SYS_ACCOUNT_JWT with your values. Make sure that you do not include the trailing , in the SYS_ACCOUNT_JWT.

```yaml
# values-auth.yaml
## Nats Config
nats:
  config:
    resolver:
      enabled: true
      merge:
        type: full
        interval: 2m
        timeout: 1.9s
    merge:
      operator: OPERATOR_JWT
      system_account: SYS_ACCOUNT_ID
      resolver_preload:
        SYS_ACCOUNT_ID: SYS_ACCOUNT_JWT
```

#### Using Secret

To use a secret instead, we need to transform the `resolver.conf` as a Kubernetes secret:

```bash
# Directly to the cluster
kubectl create secret generic mir-resolver-secret --from-file=resolver.conf
# As file
kubectl create secret generic mir-resolver-secret --from-file=resolver.conf --dry-run=client -o yaml > mir-resolver.secret.yaml
```

Alternatively, you can use the `--kubernetes` flag to combine credetials and secret creation:

```bash
mir tools security generate-resolver -p ./mir-resolver.secret.yaml --kubernetes
```

We create a new volume to mount the file and then we refer the secret.
Update <MIR_RESOLVER_SECRET> with your secret name.

```yaml
nats:
  config:
    resolver:
      enabled: true
      merge:
        type: full
        interval: 2m
        timeout: 1.9s
    merge:
      $include: ../nats-auth/resolver.conf
  container:
    patch:
    - op: add
      path: /volumeMounts/-
      value:
        name: nats-auth-include
        mountPath: /etc/nats-auth/
  podTemplate:
    patch:
    - op: add
      path: /spec/volumes/-
      value:
        name: nats-auth-include
        secret:
          secretName: <MIR_RESOLVER_SECRET>
```

The server will now be running with authorization.

### Step 3: Create Module Credentials

Let's get the Mir server up & running by generating credentials taylored for it.

```bash
# Create new user of type Module
mir tools security add module mir_srv
# Sync user with server
mir tools security push
# Create credentials file for it
mir tools security generate-creds mir_srv -p ./mir_srv.creds
```

Now time to create the Kubernetes Secret and Configure the Mir Server with it.

```bash
# Directly to the cluster
kubectl create secret generic mir-auth-secret --from-file=mir.creds=mir_srv.creds
# As file
kubectl create secret generic mir-auth-secret --from-file=mir.creds=mir_srv.creds --dry-run=client -o yaml > mir-auth.secret.yaml
# Note: the key `mir.creds` is required
```

Alternatively, you can use the `--kubernetes` flag to combine credetials and secret creation:

```bash
mir tools security generate-creds mir_srv -p ./mir-auth.secret.yaml --kubernetes
```

Finally, let's refer the secret in the `values-auth.yaml` and restart the server:

```yaml
# values-auth.yaml
## Nats Config
nats:
  config:
    resolver:
      enabled: true
      merge:
        type: full
        interval: 2m
        timeout: 1.9s
  ...

## Mir Server
authSecretRef: "mir-auth-secret"
```

Start server using `helm [install|upgrade] <name> <path> -f values-auth.yaml`.

You should see a successfull connection withour any errors from the Mir Server pod.

To validate authorization is probably setup, run `mir device ls` and you should see `nats: Authorization Violation`.

### Step 4: Create Client Credentials

Let's create a Client credentials to have full access to the system.

```bash
# Create new user of type Client
# use -h to see options
mir tools security add client ops --swarm
# Sync user with server
mir tools security push
# Generate credentials
mir tools security generate-creds ops -p ./ops.creds
```

Edit CLI configuration file to add the credentials `mir tools config edit`:

```yaml
- name: local
  target: nats://localhost:4222
  grafana: localhost:3000
  credentials: <path>/ops.creds
```

If you run `mir dev ls`, you should now see the list of devices.

### Step 5: Create Device Credentials

The last type of users is of Device type. Refer to [Integrating Mir](../integrating_mir/device/device_sdk.md) to create a device.

```bash
# Create new user of type Device
# add --wildcard to have the same credentials for all devices.
# Else it is bound to this device id.
mir tools security add device dev1
# Sync user with server
mir tools security push
# Generate credentials
mir tools security generate-creds dev1 -p ./dev1.creds
```

There are a few options to load the credentials file with the DeviceSDK.

```go
# Using Builder with fix path
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    WithCredentials("/<path>/dev1.creds").
    WithDeviceId("dev1").
    Build()
# Using Builder with default lookup
#   ./device.creds
#   ~/.config/mir/device.creds
#   /etc/mir/device.creds
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    DefaultUserCredentialsFile().
    Build()
```

It is also possible to load the credentials from the config file:

```go
# Using Builder with config file
device := mir.NewDevice().
    WithTarget("nats://nats.example.com:4222").
    DefaultConfigFile().
    Build()
```

```yaml
mir:
  credentials: "<path>/dev1.creds"
  device:
    id: "dev1"
```

Run the device and no auth errors should be displayed. Now run `mir dev ls` and you should see:

```bash
➜ mir dev ls
NAMESPACE/NAME                                DEVICE_ID        STATUS     LAST_HEARTHBEAT      LAST_SCHEMA_FETCH    LABELS
default/dev1                                  dev1             online     2025-09-18 16:16:27  2025-09-18 16:15:18
```

### Step 6: Configure Cockpit Context Credentials

The Cockpit web UI connects to NATS on behalf of the browser. When a context requires authentication, the Mir server serves the `.creds` file to the browser via `POST /api/v1/credentials?context=<name>`. You can optionally protect this endpoint with a password gate.

#### Context credentials file

**Option A - Reuse the Mir server credentials**

If all contexts share the same access level as the Mir server itself, point the context at the file already mounted via `authSecretRef`:

```yaml
# values-auth.yaml
config:
  contexts:
    - name: "production"
      sec:
        credentials: "/etc/mir/mir.creds"
```

**Option B - Dedicated Cockpit client credentials**

Gives the Cockpit a separate NATS user with its own scope, independent from the server's module user.

```bash
# Create client users for each context
mir tools security add client production_ui --swarm
mir tools security add client staging_ui --swarm
mir tools security push
mir tools security generate-creds production_ui -p ./production.creds
mir tools security generate-creds staging_ui -p ./staging.creds
```

All context credentials can share a single Kubernetes secret, use a separate key per context:

```bash
kubectl create secret generic mir-cockpit-creds \
  --from-file=production.creds=./production.creds \
  --from-file=staging.creds=./staging.creds
```

Reference the shared secret in values, pointing each context at its own key via `credsSecretKey`:

```yaml
# values-auth.yaml
config:
  contexts:
    - name: "production"
      sec:
        credsSecretRef: "mir-cockpit-creds"
        credsSecretKey: "production.creds"   # auto-mounts at /etc/mir/creds/production.creds
    - name: "staging"
      sec:
        credsSecretRef: "mir-cockpit-creds"
        credsSecretKey: "staging.creds"      # auto-mounts at /etc/mir/creds/staging.creds
```

If you only have one context, `credsSecretKey` defaults to `mir.creds`, so you can omit it:

```bash
kubectl create secret generic mir-cockpit-creds --from-file=mir.creds=./cockpit_ui.creds
```

Both options can be combined with a password gate (see below).

#### Context password gate

Add per-context passwords to the existing `mir-secret` (the secret referenced by `secretRef`) rather than putting them in plaintext values. The key pattern is `MIR__CTX__<UPPERCASE_NAME>__PASSWORD` - hyphens in context names become underscores.

```yaml
# mir.secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: mir-secret
type: Opaque
stringData:
  MIR__SURREAL__PASSWORD: "root"
  MIR__INFLUX__TOKEN: "mir-operator-token"
  # Cockpit context passwords
  MIR__CTX__PRODUCTION__PASSWORD: "your-cockpit-password"
  # For a context named "prod-eu": MIR__CTX__PROD_EU__PASSWORD: "..."
```

Apply and reference in values:

```bash
kubectl apply -f mir.secret.yaml
```

```yaml
# values-auth.yaml
secretRef: "mir-secret"
```

Restart the deployment to pick up the new secret keys. In the Cockpit, selecting a secured context will now prompt for the password before returning credentials to the browser.

### Other commands

```bash
# View current operator configuration
mir tools security env

# Sync with remote credential store
mir tools security push
mir tools security pull

# List users
mir tools security list [operators|accounts|users]
```

## Summary

Mir's integration with NATS security provides:

- **Strong authentication** using JWT and nkeys
- **Flexible authorization** with subject-based permissions
- **Simple management** through the Mir CLI
- **Production-ready** security model for IoT deployments

For additional security features, see:
- [TLS Configuration](./tls.md) for encrypted connections
- [Security Overview](./security.md) for comprehensive security architecture
