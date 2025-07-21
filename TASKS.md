# Tasks

## Immediate

### Bug

- [ ] Add deviceid to badger

### Features

- [ ] Search by wildcard
- [ ] Grafana Dashboard for eventstore
- [x] Docker container
  - [x] Multibuild
  - [x] Simple
- [x] Pipeline for each
- [x] CLI Tool template
  - [x] new command to generate config of device
  - [x] serve up with Mir
- [ ] MCP Server for Mir
- [ ] autoreconnect
  - [ ] nats
    - [ ] Check if Mir is running with a command reply
      - [ ] Check for tlm
      - [ ] Check for core
      - [ ] Check for cfg
  - [x] on startup
    - [x] surreal
    - [x] influx
  - [ ] during
    - [ ] surreal
    - [ ] influx
      - [ ] already reconnect and has a buffer
            - can we increase it?
            - add better status log
### Refactoring

### Documentation

- [ ] ModuleSDK
- [x] DeviceSDK new features
- [x] DeviceSDK with buf
- [ ] Update MdBook to latest version
- [x] Mir Concepts

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
