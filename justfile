# Display recipes
default:
    just -l --unsorted

# Build all
build:
	go build -o bin/mir cmds/mir/main.go
	go build -o bin/core cmds/core/main.go
	go build -o bin/prototlm cmds/prototlm/main.go
	go build -o bin/protocmd cmds/protocmd/main.go

# Build Mir binary
build-mir:
	go build -o bin/mir cmds/mir/main.go

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
install: build
    sudo cp bin/mir /usr/local/bin/mir

# Start tmux layouts for local dev
tx:
	tmuxifier s ./.tmux/mir.session.sh

# Run supporting infra with docker
infra:
	docker compose -f infra/local/compose.yaml up --force-recreate

# Run Mir book for local documentation
book:
	cd book && mdbook serve -p 5001

# Follow logs written to file from Tui
log:
    tail ~/.config/mir/mir.log -f

# Seed the database with test data
seed: build
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
	find . -name '*.go' | xargs -I {} cat {} | wc -l
