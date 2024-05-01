# Roadmap

- Create ProtoProxy which can listen Nats and push to db
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
  13. [ ] Add dashboard for Questdb
  14. [ ] Add timeseries field to proto library
  15. [x] Add metrics endpoint for prometheus, nodeexporter, natsio, questdb
  16. [x] Configure a grafana with questdb and prometheus data source
  17. [x] Add dashboard to see telemetry

- Create ProtoDash which can generate a dashboard from a proto file
  1. [ ] Generate dashboard for questdb

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
  - [ ] Set config in a mir folder instead of per apps
  - [x] Add unit test with sub test and better handle db close
  - [x] Add search by annotations
  - [ ] Add search by any json fields
  - [x] Add custom set of Mir errors for nice and consistent error handling
  - [x] Comment the protofile
  - [ ] Added hearthbeat functionality
  - [ ] Add env var for integration test if run

- TwinManager, the digital twin manager

- MirCLI, the Command Line Interface to easy interact with the system and with scripts
  
  - [x] Basic functionallity to manage devices
  - [x] Create a database seeding script for populating the db
  - [ ] Add name field to device
  - [ ] Merge the TUI and CLI app into one called mir

- MirTUI, the Terminal User Interface with bubble tea

  - [x] Learn BubbleTea
  - [x] Create the general parent layout
  - [x] Create basic components like tooltip and toast
  - [ ] Create main page layout
  - [ ] Create the list device layouts
  - [ ] Create the next device page layout
  - [ ] Create the edit device page layout
  - [ ] Delete a device function

- MirWebUI, the Web User Interface with htmx and templ

## Improvements
  
  - merge tui and cli into one binary
  - rework how boiler template of app is made for services
    - same tool for cli could be used for bootstrap of service
    - change how init is used to become more main and have a run method
  - configuration folder change, need to be all in a mir folder
    - where is the line between using code and a spec? maybe enforcing a spec is sufficient instead of creating a maze of code abstraction for it

### Dev setup Improvements

- fix my neovim
- single docker compose with all dependencies running
- tmux script file in repo
- think of nice setup for vscode setup or other ide.
- as the number of services becomes more complete, they will need to run same as dependency software
  - need to think about how to do it
  - tmux window with a go run main.go ?
  - container strategy for docker
