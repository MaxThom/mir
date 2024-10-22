# TODO

## Bug

- [ ] add to tooltip for newly created device
- [ ] search bug

## Immediate

- [ ] integration test for command send
- [ ] integration test for command list
- [ ] sdk for send command
- [ ] proto to json template
- [x] send commands to device
- [x] device sdk command handler
- [ ] show template in command
- [ ] event on command sent

## Refactoring

- [ ] redo module sdk
- [ ] get rid of surrealdb
- [x] create external with interfaces
- [x] hide itos from domain objects in sdk
- [ ] look on how k8s sdk is made
- [ ] ask redit for my questioning
- [x] need a centralized place for errors, in models
- [x] surreal update api, maybe try to remove optional
      optional becomes deletable fields with NONE
      non-optional presence is tested with empty/default struct/value

## Protoflux

- [x] integrate database
- [x] read on surreal type
- [x] add unit/integration tests
- [x] schema annotation for ts field
- [x] schema annotation for labels of msgs
- [x] upload schema flow
- [ ] prometheus metrics (jitter, tlm_count, schema_req, schema_err, etc)
- [x] cli/tui to upload, list and explore the schema?
- [ ] work on protodash
- [x] unit test for schema
- [ ] cli for telemetry
- [ ] tui for telemetry
- [ ] schema force update or version or something

## Protocmd

- [x] Setup new app
- [x] Basic client send command + integration test
- [x] Add cmd annotations and labels to schema
- [x] Add cmd list to explore commands and cli
- [x] Select devices
- [x] Json Payload to proto
- [x] Send cmd to devices
- [ ] Device SDK handler
- [x] Response back to protocmd and then client
- [ ] Event on send cmds and resp
- [ ] Prometheus metrics

- [ ] Retweak client handler to return an error and that get handled by the sdk
- [ ] Send schema on bootup

## General Improvements

- [ ] need a way to know when a device gets updated to refresh caches
- [ ] add a self identifier for each mir sdk user which is the app-name +  generated guid.
      each request from the sdk should have this identifier. Subsequent request such as events
      should have this id. With this, a service can subcribe to a device update event as well
      as publishing device update events. This will prevent a loop of request/events.
