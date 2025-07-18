
ld_flags := "-X 'github.com/maxthom/mir/internal/libs/build_meta.Version=0.0.0' -X 'github.com/maxthom/mir/internal/libs/build_meta.User=$(id -u -n)' -X 'github.com/maxthom/mir/internal/libs/build_meta.Time=$(date -u)'"

# Display recipes
default:
    just -l --unsorted

# Build all
build:
	go build -ldflags="{{ld_flags}}" -o bin/mir cmds/mir/main.go
	go build -ldflags="{{ld_flags}}" -o bin/core cmds/core/main.go
	go build -ldflags="{{ld_flags}}" -o bin/prototlm cmds/prototlm/main.go
	go build -ldflags="{{ld_flags}}" -o bin/protocmd cmds/protocmd/main.go
	go build -ldflags="{{ld_flags}}" -o bin/eventstore cmds/eventstore/main.go

# Build Mir binary
build-mir:
	go build -ldflags="{{ld_flags}}" -o bin/mir cmds/mir/main.go

# Run core module using Air
run-core:
	air -c .air/core.toml

# Run telemetry module using Air
run-prototlm:
	air -c .air/prototlm.toml

# Run command module using Air
run-protocmd:
	air -c .air/protocmd.toml

# Test with coverage
test:
	mkdir -p ./.tmp
	go test -coverprofile ./.tmp/coverage.out ./...
	go tool cover -html ./.tmp/coverage.out

# Install Mir to path
install: build-mir
    sudo cp bin/mir /usr/local/bin/mir

# Compile Mir to ARCH (amd64|arm64|arm32), OS (linux|windows) and SCP to host (<usr>:<ip>)
install-scp host arch="arm64" os="linux":
    GOOS={{os}} GOARCH={{arch}} go build -ldflags="-s -w {{ld_flags}}" -o bin/mir_{{os}}_{{arch}} cmds/mir/main.go
    scp bin/mir_{{os}}_{{arch}} {{host}}:mir_{{os}}_{{arch}}
    ssh {{host}} "sudo cp mir_{{os}}_{{arch}} /usr/local/bin/mir && rm mir_{{os}}_{{arch}}"
    rm bin/mir_{{os}}_{{arch}}

# Start tmux layouts for local dev
tx:
	tmuxifier s ./.tmux/mir.session.sh

# Run supporting infra with docker
infra:
	docker compose -f infra/local/compose.yaml up --force-recreate

# Build docker image
docker-build tag="latest" version="0.0.0" user="$(id -u -n)" time="$(date -u)":
    docker build -t mir:{{tag}} --build-arg VERSION={{version}} --build-arg USER="{{user}}" --build-arg TIME="{{time}}" .

# Run docker Mir
docker-run entry="serve":
    docker run --network host mir:latest {{entry}}

# Run docker Mir with a config file
docker-run-config entry="serve":
    docker run -v $(pwd)/infra/mir/local-config.yaml:/home/mir/.config/mir/mir.yaml --network host mir:latest {{entry}}

# Run docker Mir and enter the container
docker-run-exec:
    docker run -v $(pwd)/infra/mir/local-config.yaml:/home/mir/.config/mir/mir.yaml --network host -it --entrypoint /bin/sh mir:latest


# Run Mir book for local documentation
book:
	cd book && mdbook serve -p 5001

# Follow logs written to file from Tui
log:
    tail ~/.config/mir/mir.log -f

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

# Go code line count
line-count:
	find . -name '*.go' ! -name '*.pb.go' | xargs -I {} cat {} | wc -l

# Go test count
test-count:
    find . -name '*_test.go' -exec grep -E '^func Test' {} \; | wc -l
