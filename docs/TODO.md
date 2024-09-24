# TODO

## Bug

- [ ] add to tooltip for newly created device
- [ ] search bug

## Immediate

- [ ] redo module sdk
- [ ] proto to json template
- [ ] testing for list commands
- [ ] send commands to device
- [ ] device sdk command handler

## Refactoring

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
- [ ] Add cmd annotations and labels to schema
- [ ] Add cmd list to explore commands and cli
- [ ] Select devices
- [ ] Json Payload to proto
- [ ] Send cmd to devices
- [ ] Device SDK handler
- [ ] Response back to protocmd and then client
- [ ] Event on send cmds and resp
- [ ] Prometheus metrics

- [ ] Retweak client handler to return an error and that get handled by the sdk
- [ ] Send schema on bootup
