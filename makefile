.PHONY: api seed build

# scripts
seed: build
	./scripts/seed.sh

clean_db:
	./scripts/clean_db.sh

tooling:
	./scripts/tooling.sh

protogen:
	./scripts/protogen.sh

# builds
build:
	go build -o bin/tui cmds/tui/main.go
	go build -o bin/mir cmds/mir/main.go
	go build -o bin/core cmds/core/main.go
	go build -o bin/protoflux cmds/protoflux/main.go
	go build -o bin/protocmd cmds/protocmd/main.go

build-mir:
	go build -o bin/mir cmds/mir/main.go

# run
ex-module:
	go run ./examples/hearthbeat_module

ex-device:
	go run ./examples/hearthbeat_device

book:
	cd ../mir.wiki && mdbook serve -p 5001

# install
install-mir: build-mir
	sudo cp bin/mir /usr/local/bin/mir

# utils
mir-log:
	tail ~/.config/mir/mir.log -f

make dev:
	tmuxifier s ./.tmux/mir.session.sh

# air
air-core:
	air -c .air/core.toml

air-protoflux:
	air -c .air/protoflux.toml

air-protocmd:
	air -c .air/protocmd.toml
