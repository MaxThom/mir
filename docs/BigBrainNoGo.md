# BigBrainNoGo

## Ideas

## Dilema
- protobuf models and domain models
- route subjects and clients
  maybe device_route instead of device_client

IDEA
- change logger to standard logger
- all command go through command modules

INTEGRATION TEST
when using top level go test. each module spawn their
required services which can step on each other toes.
using queue group instead solve this problem. I can foresee
in the futur this problem coming back because maybe some services
will be fan out and not queue. Maybe we should stand
the infrascture including Mir services before instead of
the services coded
