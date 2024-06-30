.PHONY: api seed build

# code generation
api:
	buf generate --config ./api/buf.gen.yaml

# scripts
seed: build
	./scripts/seed.sh

clean_db:
	./scripts/clean_db.sh

# builds
build:
	go build -o bin/tui cmds/tui/main.go
	go build -o bin/mir cmds/mir/main.go
	go build -o bin/core cmds/core/main.go

build-mir:
	go build -o bin/mir cmds/mir/main.go
# run
ex-module:
	go run ./examples/hearthbeat_module

ex-device:
	go run ./examples/hearthbeat_device

# install
install-mir: build-mir
	sudo cp bin/mir /usr/local/bin/mir

# utils
mir-log:
	tail ~/.config/mir/mir.log -f

# air
air-core:
	air -c .air/core.toml
