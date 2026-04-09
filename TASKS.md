# Tasks

## Immediate

- [x] Rust theme
- [x] multi tlm
  - [x] review
  - [x] bug table/qry horizontal scroll
- [x] config update, refresh device
- [ ] perf check
- [x] tutorial on each page
- [ ] widgets
  - [x] db, dashboard unit test
  - [x] tlm, save settings of selection
  - [x] tlm, make sure toolbar work
  - [x] tlm, tiny
  - [x] tlm, wizard and multischema and start size
  - [x] tlm, header pills
  - [x] dashboard toolbar, horizontal scroll
  - [x] tlm, autorefresh and timerange
  - [x] tlm, utc switch
  - [ ] tlm, last value widget chart
  - [x] store, validate CONTAINS with *
  - [ ] widget, dev list, grey if offline
  - [x] txt mardown widget
  - [x] txt, mardown from link
  - [x] txt, fetch
  - [ ] welcome page which is a custome dashboard (online, release, get started, event list, device list)
  - [x] dev spec
  - [x] dev props
  - [x] dev online/offline
  - [ ] dev list widget
  - [x] cmd
  - [x] cfg
  - [x] evt
  - [ ] tutorial panel
  - [ ] book link
  - [ ] mir book for Cockpit
  - [ ] VIM at the top

### Bug

- [ ] Add deviceid to badger
- [ ] boolean false not get written in db, properties default not getting written
- [ ] bytes in tlm

### Features

- [ ] MCP Server for Mir
  - [ ] https://www.utcp.io  
- [ ] Healthcare
  - [ ] Uniformised wrapper arround external to have proper subscrition for conn status
  - [ ] Have system health check for higher components
- [ ] DeviceSDK v2
  - [ ] TinyGo
  - [x] Refactor
  - [x] Store with prefix ID
  - [ ] Event (error) msg to evt mod becomes event eg: disk full
  - [ ] Metrics gopsutil
- [ ] Cockpit
  - [x] New cmd for refresh schema with button next to Delete
  - [x] cfg
  - [x] tlm
  - [x] tlm with charts
    - [x] fe
    - [x] be
  - [x] create device
  - [x] device list improvement
    - [x] select multiple device
  - [ ] dashboard creation
  - [ ] security
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
