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
  - [ ] Add unit test with sub test and better handle db close
  - [x] Add search by annotations
  - [ ] Add search by any json fields
  - [x] Add custom set of Mir errors for nice and consistent error handling
  - [x] Comment the protofile
  - [ ] Added hearthbeat functionality
  - [ ] Add env var for integration test if run

- TwinManager, the digital twin manager

- MirTUI, the Terminal User Interface with bubble tea

  - [x] Learn BubbleTea
  - [x] Create the general parent layout
  - [ ] Create basic components like tooltip and toast
  - [ ] Create main page layout
  - [ ] Create the list device layouts
  - [ ] Create the next device page layout
  - [ ] Create the edit device page layout
  - [ ] Delete a device function

- MirWebUI, the Web User Interface with htmx and templ

