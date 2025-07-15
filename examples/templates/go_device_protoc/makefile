.PHONY: proto build run

proto:
	protoc \
	--proto_path=proto/ \
	--go_out=proto \
	--go_opt=paths=source_relative \
	proto/schema/v1/schema.proto

build:
	go build -o bin/device cmd/main.go

run:
	go run cmd/main.go
