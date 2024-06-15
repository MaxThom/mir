#!/bin/bash

bin="./bin/mir"

namespaces=$($bin device list -o json | sed -n 's/.*"namespace": "\([^"]*\)".*/\1/p')
for namespace in $namespaces; do
    echo "$namespace"
    $bin device delete --target.namespaces "$namespace" -o json
done
