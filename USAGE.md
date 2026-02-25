# Usage

## pprof

set -gx GO_PPROF on
pprof -seconds 10 -http=localhost:8181 http://localhost:3015/debug/pprof/profile

## Running Integration Tests Locally

```bash
# tx
just tx-test
# on terminal
just test
```

1. Start the supporting infrastructure:

```bash
# manually
docker compose -f infra/compose/local_support/compose.yaml up --force-recreate
# just
just infra
```

2. Start all the services

```bash
# manually
./scripts/integration_tests.sh
# just
just test-infra
```

3. Run all the tests

```bash
# manually
mkdir -p ./.tmp
go test -coverprofile ./.tmp/coverage.out ./...
go tool cover -html ./.tmp/coverage.out
# just
just test
```
