# Tasks

## Immediate

### Bug

### Features

- [ ] Search by wildcard
- [ ] Grafana Dashboard for eventstore
- [ ] CLI Tool template
  - [ ] new command to generate config of device
- [ ] MCP Server for Mir

### Refactoring

### Documentation

- [ ] ModuleSDK
- [ ] DeviceSDK new features
- [ ] DeviceSDK with buf
- [ ] Update MdBook to latest version
- [ ] Mir Concepts

### Ergonomics

- [ ] Tool to see data in badger

## Roadmap

- [ ] Monitoring
  - [x] Metrics
  - [x] Dashboards
  - [ ] Alerts & Alarm with Grafana and Influx
  - [x] Dashboards with influx/surreal/grafana data
- [ ] Productions
  - [ ] Docker
  - [ ] Template container for device sdk
  - [ ] Kubernetes/Helm, helm in the code and pushing chart to a registry
  - [ ] One deployment in private cluster
  - [ ] Performance analysis
- [x] Event Module
  - [x] Code
  - [x] Tests
  - [x] CLI
  - [x] watch events
  - [ ] dashboard
- [ ] DeviceSDK
  - [x] Msg store
  - [ ] Host metrics https://github.com/shirou/gopsutil
  - [ ] Buf documentation/template
  - [x] DeviceID (MAC, random, etc [save to kv store])
  - [x] Custom Routes
- [ ] ModuleSDK
  - [ ] Documentation
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
