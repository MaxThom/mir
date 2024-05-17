.PHONY: api seed build

api:
	buf generate --config ./api/buf.gen.yaml

seed: build
	./scripts/seed.sh

build:
	go build -o bin/mir cmds/cli/main.go
	go build -o bin/tui cmds/tui/main.go
	go build -o bin/core cmds/core/main.go


tui-log:
	tail ~/.config/mir/cli.log -f
