#!/bin/sh

buf lint || true
# root
buf generate --clean --template buf.gen.api.yaml
buf generate --clean --template buf.gen.device.yaml
# examples
buf generate --clean --template examples/tutorials/device/buf.gen.yaml
buf generate --clean --template examples/example_device/buf.gen.yaml
buf generate --clean --template internal/ui/cli/buf.gen.yaml
# device
buf generate --clean --template pkgs/device/mir/proto_test/buf.gen.yaml
# tests
buf generate --clean --template internal/servers/core_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/prototlm_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/protocmd_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/protocfg_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/eventstore_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/services/schema_cache/proto_test/buf.gen.yaml
buf generate --clean --template internal/libs/proto/line_protocol/proto_test/buf.gen.yaml
buf generate --clean --template internal/libs/proto/json_template/proto_test/buf.gen.yaml
buf build --path internal/libs/proto/line_protocol/proto_test/lp_test/v1/marshal.proto -o internal/libs/proto/line_protocol/proto_test/gen/lp.binpb
