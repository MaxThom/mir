# Tasks

## Immediate

### Bug

- [ ] Add deviceid to badger

### Features

- [ ] MCP Server for Mir
- [ ] TinyGo
- [ ] Road to Production
  - [ ] device security with nats
    - [ ] nkeys/jwt
      - [x] modify sdks
        - [x] add device,default credential location
      - [x] learn how it work
      - [x] integrate nsc to cli with default security for device and clients
        - [x] add the nsc generator
        - [x] what permission does a device need
        - [x] what permission does a client need
          - [x] read only
          - [x] read and write
      - [ ] work on compose release, maybe its just a readme
        - [ ] default config has resolver.conf include commented out
      - [ ] adjust helm release
    - [ ] tls
  - [ ] core bug with reconnect of loading devices
  - [ ] autoreconnect
    - [ ] nats
      - [ ] switch to jetstream for tlm
    - [ ] during
      - [ ] surreal
        - [ ] accumulate events in a buffer, event from cmd
            - [ ] Add that events can be sent without related object
            - [ ] Have a event buffer
            - [ ] events
              - [ ] online
              - [ ] offline
              - [ ] cmd send


### Refactoring

### Documentation

- [ ] ModuleSDK
- [ ] Update MdBook to latest version
- [ ] DeviceConfiguration and ServerConfiguration options
- [ ] Talk about reconnection and network loss
- [ ] Update CLI usage, mainly ctx and under device subcommand

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
  - [ ] Template container for device sdk
  - [ ] Kubernetes/Helm, helm in the code and pushing chart to a registry
  - [ ] One deployment in private cluster
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
