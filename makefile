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
	go build -o bin/mir cmds/mir/main.go
	go build -o bin/core cmds/core/main.go
	go build -o bin/prototlm cmds/prototlm/main.go
	go build -o bin/protocmd cmds/protocmd/main.go

build-mir:
	go build -o bin/mir cmds/mir/main.go

# test
test:
	mkdir -p ./.tmp
	go test -coverprofile ./.tmp/coverage.out ./...
	go tool cover -html ./.tmp/coverage.out

ci-test:
	docker compose -f infra/ci/compose.yaml up -d
	go test -coverprofile coverage.out -count 1 ./...
	docker compose -f infra/ci/compose.yaml down

# run
ex-module:
	go run ./examples/hearthbeat_module

ex-device:
	go run ./examples/hearthbeat_device

mir-book:
	cd book && mdbook serve -p 5001

# install
mir-install: build-mir
	sudo cp bin/mir /usr/local/bin/mir

# utils
mir-log:
	tail ~/.config/mir/mir.log -f

# local dev with tmuxifier
dev-tx:
	tmuxifier s ./.tmux/mir.session.sh

# docker
docker-infra:
	docker compose -f infra/local/compose.yaml up --force-recreate

# air
air-core:
	air -c .air/core.toml

air-prototlm:
	air -c .air/prototlm.toml

air-protocmd:
	air -c .air/protocmd.toml

# random
line-count:
	find . -name '*.go' | xargs -I {} cat {} | wc -l
