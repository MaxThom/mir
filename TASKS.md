# Tasks

## Immediate

### Bug

- [ ] Add deviceid to badger
- [ ] core bug with reconnect of loading devices

### Features

- [ ] MCP Server for Mir
- [ ] TinyGo
- [ ] Road to Production
  - [ ] update all dependencies
  - [x] docs
  - [ ] alert & alarms
  - [ ] perf test
  - [x] swarm
  - [ ] autoreconnect
    - [ ] if modules are down
      - [ ] switch to jetstream for tlm, could be in memory TLM
        - must add jetstream functions to ModuleSDK
        - update routes to use jetstream, hearthbeat, tlm, reported properties

### Refactoring

### Documentation

- [x] ModuleSDK
- [ ] Update MdBook to latest version
- [x] DeviceConfiguration and ServerConfiguration options
- [x] Talk about reconnection and network loss
- [x] Update CLI usage, mainly ctx and under device subcommand
- [x] Security
- [x] DeviceStorage

### Ergonomics

- [ ] Tool to see data in badger

## Roadmap

- [ ] Monitoring
  - [x] Metrics
  - [x] Dashboards
  - [ ] Alerts & Alarm with Grafana and Influx
  - [x] Dashboards with influx/surreal/grafana data
  - [x] Grafana Loki for logs
- [ ] Productions
  - [x] Docker
  - [x] Template container for device sdk
  - [x] Kubernetes/Helm, helm in the code and pushing chart to a registry
  - [x] One deployment in private cluster
  - [ ] Performance analysis
- [x] Event Module
  - [x] Code
  - [x] Tests
  - [x] CLI
  - [x] watch events
  - [x] dashboard
- [ ] DeviceSDK
  - [x] Msg store
  - [ ] Host metrics https://github.com/shirou/gopsutil
  - [x] Buf documentation/template
  - [x] DeviceID (MAC, random, etc [save to kv store])
  - [x] Custom Routes
- [ ] ModuleSDK
  - [x] Documentation
  - [x] Improvements
  - [x] Custom Routes
  - [x] Better reconnections
- [ ] Tui
  - [ ] Cfg
  - [ ] Cmd
  - [ ] Tlm

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
