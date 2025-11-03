#!/bin/bash

bin="./bin/mir"

while true; do
    namespaces=$($bin device list -o json -e | sed -n 's/.*"namespace": "\([^"]*\)".*/\1/p' | sort -u)
    if [ -z "$namespaces" ]; then
        break
    fi
    for namespace in $namespaces; do
        echo "$namespace"
        $bin device delete --target.namespaces "$namespace"
    done
done

while true; do
    namespaces=$($bin event list -o json --limit 1000 | sed -n 's/.*"namespace": "\([^"]*\)".*/\1/p' | sort -u)
    if [ -z "$namespaces" ]; then
        break
    fi
    for namespace in $namespaces; do
        echo "$namespace"
        $bin event delete --target.namespaces "$namespace"
    done
done
