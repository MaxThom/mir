# Usage

## pprof

set -gx GO_PPROF on
pprof -seconds 10 -http=localhost:8181 http://localhost:3015/debug/pprof/profile
