#!/bin/sh

buf lint || true
buf generate --clean --template buf.gen.api.yaml
buf generate --clean --template buf.gen.device.yaml
buf generate --clean --template examples/telemetry_device/buf.gen.yaml
buf generate --clean --template examples/command_device/buf.gen.yaml
buf generate --clean --template examples/config_device/buf.gen.yaml
buf generate --clean --template pkgs/device/mir/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/prototlm_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/protocmd_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/servers/protocfg_srv/proto_test/buf.gen.yaml
buf generate --clean --template internal/services/schema_cache/proto_test/buf.gen.yaml
buf generate --clean --template internal/libs/proto/line_protocol/proto_test/buf.gen.yaml
buf generate --clean --template internal/libs/proto/json_template/proto_test/buf.gen.yaml
buf generate --clean --template internal/ui/cli/buf.gen.yaml
buf build --path internal/libs/proto/line_protocol/proto_test/lp_test/v1/marshal.proto -o internal/libs/proto/line_protocol/proto_test/gen/lp.binpb
