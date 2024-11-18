
.PHONY: proto build run

proto:
	protoc \
		--proto_path=schemav1/ \
        --go_out=schemav1 \
        --go_opt=paths=source_relative \
        schemav1/schema.proto

build:
	go build -o bin/device cmd/main.go

run: build
	./bin/device
