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
  - [ ] Added hearthbeat functionality


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
  - [ ] Config and Logging setup
  - [ ] How do we publish a library? Pkg folder instead of sdks?
  - [ ] Design the event system
    - [ ] How to publish new event
    - [ ] How to catch those events
    - [ ] ServerSide SDK

### Improvements/Tech dett


### Ergonomics


## v0.2.0

The main goals of this verion is to create the Twin module as
well as doing some improvements on the boilerplate of services

- Twin module to tackle the configuration mangement of devices
- Extend CLI, TUI and Go Device SDK with new functionnalities for this module

### Features

- TwinManager, the digital twin manager

- [ ] extend spec with desired properties and status with reported properties
- [ ] create the required api calls to support updates
- [ ] create the twin template features

- CLI

- [ ] add twin commands

- Tui

- [ ] add twin layouts

- Go Device SDK

- [ ] add the cycle of desired properties and reported


### Testing


### Improvements/Tech dett

- [ ] rework how boiler template of app is made for services
  - same tool for cli could be used for bootstrap of service
  - change how init is used to become more main and have a run method
- [ ] Set config in a mir folder instead of per apps
   - where is the line between using code and a spec? maybe enforcing a spec is sufficient instead of creating a maze of code abstraction for it       -
- [ ] merge tui and cli into one binary

### Ergonmics

- [ ] air on each service app so everything reloads
- [ ] tmux script file in repo

## v0.2.1

The main goals of this version is to create the Telemetry module
as well as the visualiazing tools for the data

### Features

- ProtoFlux, receive proto telemetry and parse to flux line protocol
  - [ ] Add timeseries field to proto library


- Create ProtoDash which can generate a dashboard from a proto file
  1. [ ] Generate dashboard for questdb
  2. [ ] Generate dashboard for influxdb


### Testing

### Improvements/Tech dett

### Ergonomics

## v0.2.3

The main goal of this version is the create the Commanding module
as well as the supporting tooling and visualization

### Features

### Testing

### Improvements/Tech dett

### Ergonomics

## v0.2.4

The main goal of this version is the create the Server Side SDK
for people to bring their own module

### Features

### Testing

### Improvements/Tech dett

### Ergonomics


## v0.3.0

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


## v0.3.1
### Features

### Testing

### Improvements/Tech dett

### Ergonomics

- [ ] look at creating a static web page for documentation
    - maybe github pages? would be nice to also have them locally
- [ ] write the documentation
- [ ] write set of examples

### DevOps

- [ ] make the doc publish using a pipeline

## v0.6.0

### Features

### Testing

### Improvements/Tech dett

### Ergonomics

## v0.7.0

### Features

- MirWebUI, the Web User Interface with htmx and templ

### Testing

### Improvements/Tech dett

### Ergonomics
