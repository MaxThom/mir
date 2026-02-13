ld_flags := "-X 'github.com/maxthom/mir/internal/libs/build_meta.Version=$(git branch --show-current)-$(git rev-parse --short HEAD)' -X 'github.com/maxthom/mir/internal/libs/build_meta.User=$(id -u -n)' -X 'github.com/maxthom/mir/internal/libs/build_meta.Time=$(date -u)'"

# Display recipes
default:
    just -l --unsorted

# Build all
build:
    go build -ldflags="{{ ld_flags }}" -o bin/mir cmds/mir/main.go
    go build -ldflags="{{ ld_flags }}" -o bin/core cmds/core/main.go
    go build -ldflags="{{ ld_flags }}" -o bin/prototlm cmds/prototlm/main.go
    go build -ldflags="{{ ld_flags }}" -o bin/protocmd cmds/protocmd/main.go
    go build -ldflags="{{ ld_flags }}" -o bin/eventstore cmds/eventstore/main.go

# Build Mir binary
build-mir:
    go build -ldflags="{{ ld_flags }}" -o bin/mir cmds/mir/main.go

# Build a stripped Mir binary
build-mir-tiny:
    go build -ldflags="{{ ld_flags }} -s -w" -trimpath -o bin/mir cmds/mir/main.go

# Build cockpit web UI (Svelte)
build-cockpit-web:
    npm run build --prefix ./internal/ui/web

# Build cockpit Go server binary
build-cockpit-server:
    rsync -a --delete internal/ui/web/build/ cmds/cockpit/build/
    GOWORK=off go build -ldflags="{{ ld_flags }}" -o bin/cockpit ./cmds/cockpit

# Build complete cockpit (web UI + Go server)
build-cockpit: build-cockpit-web build-cockpit-server

# Run core module using Air
run-core:
    air -c .air/core.toml

# Run telemetry module using Air
run-prototlm:
    air -c .air/prototlm.toml

# Run command module using Air
run-protocmd:
    air -c .air/protocmd.toml

# Run cockpit web UI in dev mode (Svelte dev server)
run-cockpit-web:
    npm run dev --prefix ./internal/ui/web

# Run cockpit Go server (requires build-cockpit-web first)
run-cockpit-server:
    ./bin/cockpit

# Run cockpit Go server with Air for hot-reload
run-cockpit:
    air -c .air/cockpit.toml

# See list of direct dependencies
dep-list:
    go list -u -m -f '{{{{if not .Indirect}}{{{{.}}{{{{end}}' all

# Update all dependencies
dep-update:
    go get -u ./...

# Test with coverage
test:
    mkdir -p ./.tmp
    go test -coverprofile ./.tmp/coverage.out ./...
    go tool cover -html ./.tmp/coverage.out

# Start test infra
test-infra:
    ./scripts/integration_tests.sh

# Install Mir to path
install: build-mir-tiny
    sudo cp bin/mir /usr/local/bin/mir

# Compile Mir to ARCH (amd64|arm64|arm32), OS (linux|windows) and SCP to host (<usr>:<ip>)
install-scp host arch="arm64" os="linux":
    GOOS={{ os }} GOARCH={{ arch }} go build -ldflags="-s -w {{ ld_flags }}" -trimpath -o bin/mir_{{ os }}_{{ arch }} cmds/mir/main.go
    scp bin/mir_{{ os }}_{{ arch }} {{ host }}:mir_{{ os }}_{{ arch }}
    ssh {{ host }} "sudo cp mir_{{ os }}_{{ arch }} /usr/local/bin/mir && rm mir_{{ os }}_{{ arch }}"
    rm bin/mir_{{ os }}_{{ arch }}

# Start tmux layouts for local dev
tx:
    tmuxifier s ./.tmux/mir.session.sh

# Start tmux layouts with mir in docker and serve using CLI
tx-serve:
    tmuxifier s ./.tmux/mir-serve.session.sh

# Start tmux layouts with mir in docker
tx-full:
    tmuxifier s ./.tmux/mir-full.session.sh

# Start tmux layouts with local test setup
tx-test:
    tmuxifier s ./.tmux/mir-test.session.sh

# Run supporting infra with docker
infra:
    docker compose -f infra/compose/local_support/compose.yaml up --force-recreate

infra-down:
    docker compose -f infra/compose/local_support/compose.yaml down

# Run mir and supporting infra with docker
local:
    docker compose -f infra/compose/local_mir_support/compose.yaml up --force-recreate

local-down:
    docker compose -f infra/compose/local_mir_support/compose.yaml down

docker-kill:
    docker stop "$(docker ps -a -q)"
    docker rm "$(docker ps -a -q)"
    docker ps --all

# Build docker image
docker-build tag="latest" platform="linux/amd64" version="0.0.0" user="$(id -u -n)" time="$(date -u)":
    docker build -t ghcr.io/maxthom/mir:{{ tag }} --build-arg BUILDPLATFORM={{ platform }} --build-arg VERSION={{ version }} --build-arg USER="{{ user }}" --build-arg TIME="{{ time }}" .

# Run docker Mir
docker-run entry="serve":
    docker run --network host mir:latest {{ entry }}

# Run docker Mir with a config file
docker-run-config entry="serve":
    docker run -v $(pwd)/infra/compose/mir/local-config.yaml:/home/mir/.config/mir/mir.yaml --network host mir:latest {{ entry }}

# Run docker Mir and enter the container
docker-run-exec:
    docker run -v $(pwd)/infra/compose/mir/local-config.yaml:/home/mir/.config/mir/mir.yaml --network host -it --entrypoint /bin/sh mir:latest

# Start k3d cluster for local k8s dev
k3d-create:
    k3d cluster create mir-local-dev -c infra/k8s/k3d/scratch/k3d_config.yaml

# Delete k3d cluster
k3d-delete:
    k3d cluster delete mir-local-dev

# Recreate k3d cluster
k3d-recreate: k3d-delete k3d-create

# Deploy mir
k3d-mir instance="local":
    cd infra/k8s/charts/mir && helm install {{ instance }} . -f values-local-k3d.yaml

# Run Mir book for local documentation
book:
    cd book && mdbook serve -p 5001

# Follow logs written to file from Tui
log:
    tail ~/.config/mir/cli.log -f

# Seed the database with test data
seed:
    ./scripts/seed.sh

# Clean the database
cleandb:
    ./scripts/clean_db.sh

# Install development tools
tooling:
    ./scripts/tooling.sh

# Run protogen for all protobuf generation
protogen:
    ./scripts/protogen.sh

# Run the certificate manager script
certs *args:
    ./scripts/generate_certs.sh {{ args }}

# Go code line count
line-count:
    find . -name '*.go' | xargs -I {} cat {} | wc -l
    find . -name '*.go' ! -name '*.pb.go' | xargs -I {} cat {} | wc -l

# Go test count
test-count:
    find . -name '*_test.go' -exec grep -E '^func Test' {} \; | wc -l

# NSC Clean
nsc-clean:
    rm -rf ~/.nsc
    rm -rf ~/.local/share/nats/nsc
    rm -rf ~/.config/nats/nsc
