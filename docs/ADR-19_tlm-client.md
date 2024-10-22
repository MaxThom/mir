# Architecture Design Record

## ADR-19, Telemetry subscription client

### CLI Design

To see telemetry, create an influx query

```sh
mir telemetry play --start <ts> --end <ts> --since <timespan> --refresh <timespan>|3sec --output [csv|pretty] --target.id <ids...> --target.namespace <namespace...> --target.labels <kv...> --target.annotations <kv..> --target.measurements <table> --target.fields <columns> --target.tag <tags...>
```

To explore the telemetry schema

```sh
mir telemetry list --target... [all target types]
```

### Architecture

1. subscription client listen to request to subsribe
2. send live data or query database

### Preconditions

- Either do with influx query directly

- or wait for tlmv2
