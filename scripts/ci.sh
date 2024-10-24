#!/bin/bash

bin="./bin/mir"

go build -o bin/core cmds/core/main.go
go build -o bin/protoflux cmds/protoflux/main.go
go build -o bin/protocmd cmds/protocmd/main.go

mkdir -p .tmp
for binary in $(ls "./bin"); do
    "./bin/$binary" > "./.tmp/$binary.log" 2>&1 &
done
