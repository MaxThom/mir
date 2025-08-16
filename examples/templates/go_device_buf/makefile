.PHONY: proto build run docker-build docker-run docker-run-config docker-run-exec

proto:
	buf generate
	buf lint

build:
	go build -o bin/mir.dev cmd/main.go

build-tiny:
	go build -ldflags="-s -w" -trimpath -o bin/mir.dev cmd/main.go

run:
	go run cmd/main.go

GITHUB_TOKEN := $(shell gh auth token)
docker-build:
	docker build --build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) -t mir.dev:latest .

docker-run:
	docker run --network host mir.dev:latest

docker-run-config:
	docker run -v $(pwd)/device.yaml:/home/mir/.config/mir/device.yaml --network host mir.dev:latest

docker-run-exec:
	docker run -v $(pwd)/device.yaml:home/mir/.config/mir/device.yaml --network host -it --entrypoint /bin/sh mir.dev:latest
