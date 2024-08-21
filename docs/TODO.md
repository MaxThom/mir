# TODO
## Immediate

- [x] fix cycle of clients and mir module

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
