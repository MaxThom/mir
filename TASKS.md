# Tasks

## Immediate

- [x] Rust theme
- [x] multi tlm
  - [ ] review
  - [ ] bug table/qry horizontal scroll
- [ ] perf check
- [ ] tutorial on each page
- [ ] schema page

### Bug

- [ ] Add deviceid to badger
- [ ] boolean false not get written in db, properties default not getting written
- [ ] bytes in tlm

### Features

- [ ] MCP Server for Mir
  - [ ] https://www.utcp.io  
- [ ] DeviceSDK v2
  - [ ] TinyGo
  - [x] Refactor
  - [ ] Store with prefix ID
  - [ ] Metrics gopsutil
  - [ ] Bbolt integration (choose between that and badger)
  - [ ] Tool to see data in badger
  - [ ] Disk size limits as well as TTL
  - [ ] Error msg to core eg: disk full
- [ ] Cockpit
  - [x] New cmd for refresh schema with button next to Delete
  - [x] cfg
  - [x] tlm
  - [x] tlm with charts
    - [x] fe
    - [x] be
  - [ ] device list improvement
    - [ ] select multiple device
    - [ ] collapsible with device view
- [ ] Road to Production
  - [x] update all dependencies
  - [ ] alert & alarms
  - [ ] autoreconnect
    - [ ] if modules are down
      - [ ] switch to jetstream for tlm, could be in memory TLM
        - must add jetstream functions to ModuleSDK
        - update routes to use jetstream, hearthbeat, tlm, reported properties

### Refactoring

### Documentation

- [ ] Cockpit

### Ergonomics

- [ ] Tool to see data in badger

## Roadmap

- [ ] Monitoring
  - [ ] Alerts & Alarm with Grafana and Influx
- [ ] DeviceSDK
  - [ ] Host metrics https://github.com/shirou/gopsutil
  - [ ] Rewrite
- [ ] Tui
  - [ ] Tlm display

## Far Future

- [ ] DeviceSDK
  - [ ] Bbolt integration (choose between that and badger)
  - [ ] Tool to see data in badger
  - [ ] Disk size limits as well as TTL
- [ ] ModuleSDK
  - [ ] Handle protocache registry and can use device schema
  - [ ] Should the schema be in the nats kv?
- [ ] Scope in namespace
- [ ] Service in trigger chain
- [ ] Autotwin template
- [ ] nats
  - [ ] Check if Mir is running with a command reply
    - [ ] Check for tlm, if down set to storage
    - [ ] Check for core
    - [ ] Check for cfg, if down
    - [ ] Check for cmd
  - [ ] Solution
    - [ ] part of hearthbeat as reply/request
    - [ ] core keep status of running services and can return system status
    - [ ] if core, cfg or tlm down, set local persistence
    - [ ] if cfg degraded, set local persistence
    - [ ] each service, need a health route, part of sdk
