# Roadmap

## v0.1.0

The main goals of the v0.1.0 is to get the uncertainties
out of the way, implement basic functionnalities and
create the tools to help on this adventure

- a proof of concept for the Protoproxy as their
is much incertainties about the feasibility of that one
- Core module to manage devices
- CLI to easily interac with the system via vash and scripts
- TUI to easiliy interac with the system via a fun user experience
- Go Device SDK to create basic devices

### Features
- Create a poc of ProtoProxy which can listen Nats and push to db
  1. [x] Need to create store library
     - [x] Create store server
  2. [x] need to select db [questdb]
  3. [x] need to deploy db [docker compose]
  4. [x] Need to create the deserialize library to line protocol
  5. [x] use unit test to validate
  6. [x] Need to deploy NatsIO [docker compose]
  7. [x] Need to create a NatsIO library
  8. [x] Need to create to pipe the natsio telemetry to the db through protoproxy
  9. [x] Deploy Questdb and connect
  10. [x] Add metrics to protoproxy
  11. [x] Add dashboard for protoproxy
  12. [x] Add dashboard for natsio
  15. [x] Add metrics endpoint for prometheus, nodeexporter, natsio, questdb
  16. [x] Configure a grafana with questdb and prometheus data source
  17. [x] Add dashboard to see telemetry

- Core, register new device and basic management. The Core.

  - [x] Create a new device
  - [x] Update a device
  - [x] Delete a device
  - [x] List all devices with labels filter
  - [x] Get a device or a list of device with list of ids
  - [x] Setup unit test boilerplate
  - [x] Setup unit test for each functions
  - [x] Setup SurrealDB
  - [x] Setup NatsIO and request reply paradigm
  - [x] Add unit test with sub test and better handle db close
  - [x] Add search by annotations
  - [ ] Add search by any json fields
  - [x] Add custom set of Mir errors for nice and consistent error handling
  - [x] Comment the protofile
  - [x] Added hearthbeat functionality


- MirCLI, the Command Line Interface to easy interact with the system and with scripts

  - [x] Basic functionallity to manage devices
  - [x] Create a database seeding script for populating the db
  - [x] Add name field to device

- MirTUI, the Terminal User Interface with bubble tea

  - [x] Learn BubbleTea
  - [x] Create the general parent layout
  - [x] Create basic components like tooltip and toast
  - [x] Create main page layout
  - [x] Create the list device layouts
  - [x] Create the create device page layout
  - [x] Create the edit device page layout
  - [x] Delete a device function


-  MirGoSdk, device sdk in Go

  - [x] Create builder or Option patterns for sdk
  - [x] Have Hearthbeat functionality implemented
  - [x] Config and Logging setup
  - [ ] How do we publish a library? Pkg folder instead of sdks?
  - [x] Design the event system
    - [x] How to publish new event
    - [x] How to catch those events
    - [x] ServerSide SDK

### Improvements/Tech dett


### Ergonomics

## v0.2.0 Telemetry module

The main goals of this version is to create the Telemetry module
as well as the visualiazing tools for the data

### Features

#### Server Module

- [x] ProtoFlux, handle telemetry data from protobuf to line protocol

#### CLI/TUI

- [x] Upload schema via CLI
- [x] Schema explorer via CLI and maybe TUI
- [ ] Create ProtoDash which can generate a dashboard from a proto file
- [ ] Create subscribe client in protoflux for cli and tui to see telemetry

#### Device SDK

- [x] Custom Protobuff annotation for Mir System
- [x] Added telemetry function to the SDK

#### Module SDK

- [ ] Add new set of events regarding telemetry
- [x] Add stream subscriptions

### Testing

- [x] Integration test for the telemetry module

### Improvements/Tech dett

- [x] Project layout refactor
- [x] Decoupling of storage and server handlers
- [x] rework how boiler template of app is made for services
  - same tool for cli could be used for bootstrap of service
  - change how init is used to become more main and have a run method
- [x] Set config in a mir folder instead of per apps
   - where is the line between using code and a spec? maybe enforcing a spec is sufficient instead of creating a maze of code abstraction for it       -
- [x] merge tui and cli into one binary

### Ergonomics

- [x] Create tmuxifier layouts in repo
- [x] Make command for buf generate

## v0.3.0 Command Module

The main goal of this version is the create the Commanding module
as well as the supporting tooling and visualization

### Features

#### Server Module

- [ ] Can define commands in protobuf schema
- [ ] Send command with Targets and JSON payload to target multiple devices

#### CLI/TUI

- [ ] Explore commands
- [ ] Be able to send commands via  window with parameters based on the schema

#### Device SDK

- [ ] Custom Protobuff annotation for Mir System for commands
- [ ] Added commands handler to the SDK

#### Module SDK

- [ ] Add new set of events regarding commands

### Testing

- [ ] Integration test for the command module

### Improvements/Tech dett


### Ergonomics


### Testing

### Improvements/Tech dett

### Ergonomics

## v0.4.0 Twin Module

Twin module to tackle the configuration mangement of devices. Flow of desired properties set by the user and reported properties set by the device.

### Features

#### Server Module

- [ ] Can define properties in protobuf schema
	  	or maybe JSON is better since it will be hard with the twin template

#### CLI/TUI

- [ ] Can create twin template

#### Device SDK

- [ ] Custom Protobuff annotation for Mir System for properties
- [ ] Add desired properties handler to the SDK
- [ ] Add reported properties function

#### Module SDK

- [ ] Add new set of events regarding propeties
- [ ] create the twin template features

### Testing

### Improvements/Tech dett

### Ergonmics


## v0.5.0 DeviceSDK and ModuleSDK Improvements

The goal is too focus on SDK requirements or QOL that are not bound to a module

### Features

#### Device SDK

- [ ] Add local storage for message in case of network outage
- [ ] Add the ability to publish to custom routes

#### Module SDK

- [ ] Add the ability to subscribe to custome route
- [ ] Look at replacing SurrealDB with NatsIO Keyvalue or Badger

### Testing

### Improvements/Tech dett

### Ergonmics


## v0.6.0

The main goal of this version is to focus on the
deployment and production toolings such as pipeline
and conternarization

### Features


### Testing

- [ ] Add env var for integration test if run

### Improvements/Tech dett

### Ergonomics

### DevOps

- [ ] Containerize all services
- [ ] Provide template container for sdk
- [ ] Create Docker Compose for each and one all together

- [ ] Create set of pipelines for unit and integration testing
- [ ] Pipeline to release binaries of each interfaces or services
- [ ] Pipeline for publishing containers

- [ ] Make sure the sdks are available via go get/install



## v0.7.0 Swarm

Create a utility tool to virtualize devices. This will be used for extensive integration and performance testing to increase reliability and performance

### Features

### Testing

### Improvements/Tech dett

### Ergonomics
