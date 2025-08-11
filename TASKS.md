# Tasks

## Immediate

### Bug

- [ ] Add deviceid to badger

### Features

- [ ] Search by wildcard
- [ ] MCP Server for Mir
- [ ] Road to Production
  - [ ] device security with nats
  - [ ] CLI
    - [ ] config edit and view command
    - [ ] log view command
    - [ ] cli context for easy cluster switch
    - [ ] tlm ls print query option
  - [ ] core bug with reconnect of loading devices
  - [x] Grafana Dashboard for eventstore
  - [ ] HelmChart
    - [x] Mir
    - [x] Nats
    - [x] Surreal
      - [x] Test with CLI, cant change url
    - [x] Influx
    - [ ] PromStack
  - [x] Compose release
  - [x] GrafanaLoki
  - [x] Docker container
    - [ ] Simple/Multi for device
    - [x] Multibuild
    - [x] Simple
  - [ ] Pipeline for each
    - [ ] HelmChart
    - [ ] For device
    - [x] For server
  - [x] CLI Tool template
    - [x] new command to generate config of device
    - [x] serve up with Mir
  - [ ] autoreconnect
    - [x] nats
      - [x] nats
      - [ ] switch to jetstream for tlm
    - [x] on startup
      - [x] surreal
      - [x] influx
    - [ ] during
      - [ ] surreal
        - [x] running in degraded state
        - [x] core, doesnt work anymore
        - [x] cfg, list can work
        - [x] cmd, can work if same schema
        - [x] tlm, can work if same schema
        - [ ] accumulate events in a buffer, event from cmd
            - [ ] Add that events can be sent without related object
            - [ ] Have a event buffer
            - [ ] events
              - [ ] online
              - [ ] offline
              - [ ] cmd send
       - [x] influx
        - [x] recreate org/bucket
        - [x] already reconnect and has a buffer
              - can we increase it?
              - add better status log
### Refactoring

### Documentation

- [ ] ModuleSDK
- [x] DeviceSDK new features
- [x] DeviceSDK with buf
- [ ] Update MdBook to latest version
- [x] Mir Concepts
- [ ] DeviceConfiguration
- [ ] Talk about reconnection and network loss

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
