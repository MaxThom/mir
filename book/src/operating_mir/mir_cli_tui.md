# Mir CLI

The CLI is both a CLI and a TUI. The CLI offers all interaction with the system and devices.

To get the Mir CLI, visit [Running Mir with Binary](../running_mir/binary.md)

To launch in TUI mode, simply runs `mir` without arguments.

The CLI is a powerful low level tool to interact with the Mir ecosystem.
Use it to manage devices, explore telemetry, send commands,  serve the ecosystem, and more.
Use it as your companion to develop and operate your IoT devices.
Use shell script, create powerful automation and integration with other tools.

## CLI

Let's start with a tour of the CLI `mir -h`. Mir CLI act as both the client and the server
giving a united tool to do all that is required. Moreover, it provices a set of tools
to enhance development and operation.

```
Usage: mir <command> [flags]

A command line and terminal user interface to operate the Mir ecosystem 🛰️

Commands:
  device (dev)       Manage fleet of Mir devices
  telemetry (tlm)    Explore Mir devices telemetry
  command (cmd)      Send and explore commands to devices
  schema (sch)       Upload and explore device proto schema
  serve              Serve Mir ecosystem of servers and services
  infra              Start and stop Mir supporting infrastructure
  swarm              Create virtual devices to mimic workload for test or demo purposes
  tools              Various tools to interact with Mir ecosystem

Run "mir <command> --help" for more information on a command.
```


Let's get the server up and running to have something to work with.

### Server

Mir requires it's supporting infrastructure and it's services to be running.

Mir infra managed a set of docker compose files for you to have a local environment ready to go at your disposal.
Each extra flag is passed to docker compose. For example, `mir infra up -d` will run the docker compose in detached mode.

```
Usage: mir infra <command> [flags]

Start and stop Mir supporting infrastructure

Commands:
  infra up       Run infra docker compose up
  infra down     Run infra docker compose down
  infra ps       Run infra docker compose ps
  infra rm       Run infra docker compose rm
  infra write    Write to disk Mir set of docker compose
```

Open a terminal and run `mir infra up` to get the supporting infrastructure running.

With the supporting infrastructure running, we can now start the Mir server. In a new terminal, run `mir serve`.
The default configuration is made to work in par with the supporting infrastructure setup previously.

If we want to bring an external infrastructure, flags can be passed to modify connections or by using a configuration file.
To help with the configuration, run `mir serve --display-default-cfg`. This will print the default configuration in yaml.
By default, Mir load its configuration from `/etc/mir/mir.yaml` or `/home/<USER>/.config/mir/mir.yaml`

*! Tips: use `mir serve --display-default-cfg > /etc/mir/mir.yaml` and then edit this file to adjust your configuration needs.*

With both `infra` and `serve` commands, you have a full Mir setup running!

### Client

Time to interact the system. For that we will use the `swarm` command to mimic devices.
Start a swarm with `mir swarm --ids=power,weather`. Open the a new terminal to start interacting with them.

#### Device Management

With the `device` command, you can manage your fleet:

```bash
# See list of devices
mir device list <name/namespace>
# Print digital twin of a device
mir device list power/default -o yaml
```

*! Tips: all commands that interact with devices can be filtered by name/namespace as first positional arguments or with --target flag.*
*Use /namespace for all device in that namespace*

To create a device, you can use the different flags to pass the initial configuration:

```bash
mir device create <flags>
```

You can also use a declarative approach to create a device:

```bash
# Output device template to file
mir device create -j > device.yaml
# Edit and create
cat device.yaml | mir device create
```

There is a few ways to update a device:

```bash
# Edit a device interactively
mir device edit power/default
# Edit or create a device declaratively
mir apply -f <file>.yaml
# You can combine list and apply
mir list power/default -o yaml > power.yaml
# Edit the file and apply
mir apply -f power.yaml
# Use set of flags to update a device
mir device update <flags>
```

Finally, to delete devices:

```bash
device delete power/default
```

#### Device telemetry

```bash
mir telemetry list <name/namespace>
```

This command will output the list of outgoing telemetry from devices.
It will also print a url that you can open to see your data in Grafana.

*! Tips: if you dont see all telemetry, use `-r` to refresh the schema*

The explore panel in Grafana is a great way to see telemetry as well as
offering an example of the query to see that data. Use the query as a starting point
to build powerful dashboards.

#### Device command

As device update, there is a few ways to send a command:

```bash
# List available commands
mir command <name/namespace>
# Shortcut to see available commands
mir send <name/namespace>
# See a command json payload
mir send <name/namespace> -n <command_name> -j
# Send a command. Single quotes help in writing json on terminal.
mir send <name/namespace> -n <command_name> -p '<json_payload>'
# Send a command declaratively
cat payload.json | mir send <name/namespace> -n <command_name>
# Send a command interactively
mir send <name/namespace> -n <command_name> -e
```

Each command will return a response from each devices that the command targeted.
Moreover, you can use the flag `--dry-run`to validate the command without sending it.

*! Tips: if you dont see all commands, use `-r` to refresh the schema*

### Tools

The Mir CLI provices a set of tools to enhance development:

```bash
# Install required tools for Device development
mir tools install
# Generate Mir device schema
mir tools generate mir-schema
# Generate a device project template to get started
mir tools generate device-template <module-name>
```

## TUI

The TUI or Terminal User Interface is a way to interact with the system in a more visual and interactive way.
Simple run `mir` to get it running!

Use `?` to get help on the current view and see the equivalent CLI command. Yours to explore.
