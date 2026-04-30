# Securing your Docker deployment

## Prerequisites

Install required security tools:

- [NSC](https://docs.nats.io/using-nats/nats-tools/nsc)
- `mir tools install`

Have a mir deployment ready to be used:
- [Setup Compose Release](../running_mir/docker.md)

## Setup

Mir Security CLI wraps NSC commands with a set of preset to make securing Mir ecoystem easier. It offers a set of basic commands to manipulate credentials. Moreover, it offers premade scope for the three types of users in Mir:

- Modules (server components)
- Clients (access CLI and other frontend)
- Devices (connect devices)

The CLI uses the current context to help manage which server to target. Use `mir tools config edit` to add a new context with server name and url:

```yaml
# If using local setup
- name: local
  target: nats://localhost:4222
  webTarget: ws://localhost:9222
  grafana: localhost:3000
  sec:
    credentials: ""
    password: ""
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

Update NATS Configuration

Edit `./mir-compose/natsio/config.conf` and uncomment or add this line `include resolver.conf`.

Start server `docker compose up`.


The server is now running with authorization. To validate, run `mir device ls` and you should see `nats: Authorization Violation`. Similar for the logs of Mir server.

Moreover, if you run `mir tools sec list accounts` you should see two accounts: SYS and mir.

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

Now let's launch the server with the credentials file. Edit `./mir-compose/mir/local-config.yaml` and set the path under `nats.credentials`. Edit `./mir-compose/mir/compose.yaml` to mount the file.

```yaml
nats:
  url: "nats://local_mir_support-nats-1:4222"
  credentials: "/home/mir/creds/mir_srv.creds"
```

```bash
# Restart server
docker compose down
docker compose up
```

You should see a successfull connection without any errors.

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
  webTarget: ws://localhost:9222
  grafana: localhost:3000
  sec:
    credentials: <path>/ops.creds
```

If you run `mir dev ls`, you should now see the list of devices.

### Step 5: Configure Cockpit Server List

Cockpit reads available contexts from `./mir-compose/mir/local-contexts.yaml` (mounted into the container as `cli.yaml`) and exposes them to the browser via `GET /api/v1/contexts`. Each context entry defines a server the web UI can connect to.

Edit `./mir-compose/mir/local-contexts.yaml` to list your servers:

```yaml
logLevel: info
currentContext: local
contexts:
  - name: local
    target: nats://localhost:4222
    webTarget: ws://localhost:9222
    grafana: localhost:3000
    sec:
      credentials: ""
      password: ""
```

If you want to connect to a secure nats server. Mount the `.creds` file, set the path and a password.

```yaml
contexts:
  - name: production
    target: nats://prod.example.com:4222
    webTarget: wss://prod.example.com:9222
    grafana: prod.example.com:3000
    sec:
      credentials: "/home/mir/creds/ops.creds"
      password: "your-secret-password"
```

Restart to apply: `docker compose down && docker compose up`.

### Step 6: Create Device Credentials

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
