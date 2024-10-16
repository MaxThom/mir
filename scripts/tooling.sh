#!/bin/sh

# TODO verify before dl
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install github.com/air-verse/air@latest
go install github.com/bufbuild/buf/cmd/buf@latest
cargo install mdbook
echo '--== Post Install ==--'
echo '  ? Don''t forget to append go binaries path to path if not set (export PATH=$PATH:$(go env GOPATH)/bin)'
echo '  ? Don''t forget to append rust binaries path to path if not set (export PATH=$PATH:$HOME/.cargo/bin)'
echo '--== Happy Mir Coding 🛰️ ==--'
