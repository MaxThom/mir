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
	go build -o bin/protoflux cmds/protoflux/main.go

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

protogen:
	buf lint || true
	buf generate --template buf.gen.api.yaml
	buf generate --template buf.gen.device.yaml
	buf generate --template examples/telemetry_device/buf.gen.yaml
	buf generate --template pkgs/device/mir/proto_test/buf.gen.yaml
	buf generate --template pkgs/module/mir/proto_test/buf.gen.yaml
	buf generate --template internal/services/protoflux_srv/proto_test/buf.gen.yaml
	buf generate --template internal/libs/proto/line_protocol/proto_test/buf.gen.yaml
	buf build --path internal/libs/proto/line_protocol/proto_test/lp_test/v1/marshal.proto -o internal/libs/proto/line_protocol/proto_test/gen/lp.binpb

# air
air-core:
	air -c .air/core.toml

air-protoflux:
	air -c .air/protoflux.toml
