#!/bin/sh

check_go() {
    if ! command -v go >/dev/null 2>&1; then
        echo "❌ Go is not installed and required"
        exit 1
    fi
    echo "✅ Go is installed"
    return 0
}

check_rust() {
    if ! command -v rustc >/dev/null 2>&1; then
        echo "❌ Rust is not installed and required"
        exit 1
    fi
    echo "✅ Rust is installed"
    return 0
}

check_npm() {
    if ! command -v npm >/dev/null 2>&1; then
        echo "❌ Npm is not installed and required"
        exit 1
    fi
    echo "✅ Npm is installed"
    return 0
}

echo '-- Pre Install --'
check_go
check_rust
check_npm

# TODO verify before dl
echo -e '\n-- Install Go Tools --'
echo "- protoc-gen-go"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
echo "- air"
go install github.com/air-verse/air@latest
echo "- buf"
go install github.com/bufbuild/buf/cmd/buf@latest
echo "- badger"
go install github.com/dgraph-io/badger/v4/badger@latest
echo "- natscli"
go install github.com/nats-io/natscli/nats@latest

echo -e '\n-- Install Rust Tools --'
echo "- mdbook"
cargo install mdbook@0.5.2
echo "- just"
cargo install just

echo -e '\n-- Install Npm Tools --'
echo "- vite"
npm install -D vite
echo "- svelte"
npm install -D svelte
echo "- protoc-gen-es"
npm install -g @bufbuild/protoc-gen-es

echo -e '\n-- Install Other Tools --'
echo "- surreal"
curl -sSf https://install.surrealdb.com | sh
echo "- nsc"
curl -L https://raw.githubusercontent.com/nats-io/nsc/master/install.py | python
sudo mv ~/.surrealdb/surreal /usr/local/bin

echo -e '\n-- Post Install --'
echo '  ? Don''t forget to append go binaries path to path if not set (export PATH=$PATH:$(go env GOPATH)/bin)'
echo '  ? Don''t forget to append rust binaries path to path if not set (export PATH=$PATH:$HOME/.cargo/bin)'
echo '-- Happy Mir Coding 🛰️ --'
