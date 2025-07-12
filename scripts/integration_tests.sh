#!/bin/bash

# Parse command line arguments
TEST_PATH="${1:-./...}"

# In go test ./..., the ./... means:
#  - ./ - Start from the current directory
#  - ... - Recursively include all subdirectories
# Examples:
#  - go test ./internal/... - Tests in internal/ and all subdirs
#  - go test ./cmd/... - Tests in cmd/ and all subdirs
#  - go test ./... - Tests in current directory and all subdirs

set -e
# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Log functions
log_info() {
    echo -e "${BLUE}$1${NC}"
}

log_success() {
    echo -e "${GREEN}$1${NC}"
}

log_error() {
    echo -e "${RED}$1${NC}"
}

log_warn() {
    echo -e "${YELLOW}$1${NC}"
}

log_debug() {
    echo -e "${PURPLE}$1${NC}"
}

log_build() {
    echo -e "${CYAN}$1${NC}"
}

# Function to start core service
start_core() {
    go build -o bin/core cmds/core/main.go

    cat > .tmp/core_config.yaml << 'EOF'
logLevel: "debug"
httpServer:
  port: 3026
dataBusServer:
  url: "nats://127.0.0.1:4222"
databaseServer:
  url: "ws://127.0.0.1:8000/rpc"
  user: "root"
  password: "root"
  namespace: "global"
  database: "mir_testing"
EOF

    ./bin/core -config .tmp/core_config.yaml > .tmp/core.log 2>&1 &
    CORE_PID=$!
    log_info "- Started core service (.tmp/core.log) with PID: $CORE_PID"
}

# Function to start eventstore service
start_eventstore() {
    go build -o bin/eventstore cmds/eventstore/main.go

    cat > .tmp/eventstore_config.yaml << 'EOF'
logLevel: "debug"
httpServer:
  port: 3030
dataBusServer:
  url: "nats://127.0.0.1:4222"
databaseServer:
  url: "ws://127.0.0.1:8000/rpc"
  user: "root"
  password: "root"
  namespace: "global"
  database: "mir_testing"
EOF

    ./bin/eventstore -config .tmp/eventstore_config.yaml > .tmp/eventstore.log 2>&1 &
    EVENTSTORE_PID=$!
    log_info "- Started eventstore service (.tmp/eventstore.log) with PID: $EVENTSTORE_PID"
}

# Function to start protocfg service
start_protocfg() {
    go build -o bin/protocfg cmds/protocfg/main.go

    cat > .tmp/protocfg_config.yaml << 'EOF'
logLevel: "debug"
httpServer:
  port: 3029
dataBusServer:
  url: "nats://127.0.0.1:4222"
databaseServer:
  url: "ws://127.0.0.1:8000/rpc"
  user: "root"
  password: "root"
  namespace: "global"
  database: "mir_testing"
EOF

    ./bin/protocfg -config .tmp/protocfg_config.yaml > .tmp/protocfg.log 2>&1 &
    PROTOCFG_PID=$!
    log_info "- Started protocfg service (.tmp/protocfg.log) with PID: $PROTOCFG_PID"
}

# Function to start protocmd service
start_protocmd() {
    go build -o bin/protocmd cmds/protocmd/main.go

    cat > .tmp/protocmd_config.yaml << 'EOF'
logLevel: "debug"
httpServer:
  port: 3028
dataBusServer:
  url: "nats://127.0.0.1:4222"
databaseServer:
  url: "ws://127.0.0.1:8000/rpc"
  user: "root"
  password: "root"
  namespace: "global"
  database: "mir_testing"
EOF

    ./bin/protocmd -config .tmp/protocmd_config.yaml > .tmp/protocmd.log 2>&1 &
    PROTOCMD_PID=$!
    log_info "- Started protocmd service (.tmp/protocmd.log) with PID: $PROTOCMD_PID"
}

# Function to start prototlm service
start_prototlm() {
    go build -o bin/prototlm cmds/prototlm/main.go

    cat > .tmp/prototlm_config.yaml << 'EOF'
logLevel: "debug"
httpServer:
  port: 3027
dataBusServer:
  url: "nats://127.0.0.1:4222"
databaseServer:
  url: "ws://127.0.0.1:8000/rpc"
  user: "root"
  password: "root"
  namespace: "global"
  database: "mir_testing"
telemetryServer:
  url: "http://127.0.0.1:8086"
  token: "mir-operator-token"
  org: "mir"
  bucket: "mir_testing"
EOF

    ./bin/prototlm -config .tmp/prototlm_config.yaml > .tmp/prototlm.log 2>&1 &
    PROTOTLM_PID=$!
    log_info "- Started prototlm service (.tmp/prototlm.log) with PID: $PROTOTLM_PID"
}

# Check if services are still running
check_services() {
    local failed=false

    if ! kill -0 $CORE_PID 2>/dev/null; then
        log_error "- Core service (PID: $CORE_PID) has died!"
        log_error "- Core service logs:"
        cat .tmp/core.log
        failed=true
    fi

    if ! kill -0 $EVENTSTORE_PID 2>/dev/null; then
        log_error "- EventStore service (PID: $EVENTSTORE_PID) has died!"
        log_error "- EventStore service logs:"
        cat .tmp/eventstore.log
        failed=true
    fi

    if ! kill -0 $PROTOCFG_PID 2>/dev/null; then
        log_error "- ProtoCfg service (PID: $PROTOCFG_PID) has died!"
        log_error "- ProtoCfg service logs:"
        cat .tmp/protocfg.log
        failed=true
    fi

    if ! kill -0 $PROTOCMD_PID 2>/dev/null; then
        log_error "- ProtoCmd service (PID: $PROTOCMD_PID) has died!"
        log_error "- ProtoCmd service logs:"
        cat .tmp/protocmd.log
        failed=true
    fi

    if ! kill -0 $PROTOTLM_PID 2>/dev/null; then
        log_error "- ProtoTlm service (PID: $PROTOTLM_PID) has died!"
        log_error "- ProtoTlm service logs:"
        cat .tmp/prototlm.log
        failed=true
    fi

    if [ "$failed" = true ]; then
        cleanup
        exit 1
    fi

    log_info "- All services are running successfully!"
}

cleanup() {
    if [ "$CLEANUP_DONE" = "true" ]; then
        return
    fi
    CLEANUP_DONE=true

    # Kill all services
    log_warn "\nStopping all Mir services..."
    kill $CORE_PID 2>/dev/null || true
    log_info "- Stopped core with PID: $CORE_PID"
    kill $EVENTSTORE_PID 2>/dev/null || true
    log_info "- Stopped eventstore with PID: $EVENTSTORE_PID"
    kill $PROTOCFG_PID 2>/dev/null || true
    log_info "- Stopped protocfg with PID: $PROTOCFG_PID"
    kill $PROTOCMD_PID 2>/dev/null || true
    log_info "- Stopped protocmd with PID: $PROTOCMD_PID"
    kill $PROTOTLM_PID 2>/dev/null || true
    log_info "- Stopped prototlm with PID: $PROTOTLM_PID"
}

# Check if surreal CLI exists
log_success "Cleaning databases data..."
if command -v surreal &> /dev/null; then
    log_info "- Surreal"
    echo "DELETE devices;DELETE events;" | surreal sql -u root -p root --ns global --db mir_testing --hide-welcome
else
    log_warn "- Surreal CLI not found"
fi

# Start all services
log_info "- Influx"

log_success "Starting all Mir services for testing..."
mkdir -p .tmp
start_core
start_eventstore
start_protocfg
start_protocmd
start_prototlm

# Wait 5 seconds and check if services are still running
log_success "Waiting 5 seconds for services to stabilize..."
sleep 5
check_services

# Wait for OS signal to shutdown
log_success "All services started and verified. Press Ctrl+C to shutdown..."

# Wait for exit
trap cleanup INT TERM EXIT
while true; do
    sleep 1
done
