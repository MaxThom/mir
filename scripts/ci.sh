#!/bin/bash

bin="./bin/mir"

go build -o bin/core cmds/core/main.go
go build -o bin/prototlm cmds/prototlm/main.go
go build -o bin/protocmd cmds/protocmd/main.go
go build -o bin/protocfg cmds/protocfg/main.go
go build -o bin/eventstore cmds/eventstore/main.go

mkdir -p .tmp
for binary in $(ls "./bin"); do
    "./bin/$binary" > "./.tmp/$binary.log" 2>&1 &
done
