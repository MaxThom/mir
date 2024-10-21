package test_utils

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func GetProtoBytes(m proto.Message) ([]byte, string, error) {
	b, err := proto.Marshal(m)
	if err != nil {
		return nil, "", err
	}

	return b, string(m.ProtoReflect().Descriptor().FullName()), nil
}

func GetJsonBytes(m proto.Message) ([]byte, string, error) {
	b, err := protojson.Marshal(m)
	if err != nil {
		return nil, "", err
	}

	return b, string(m.ProtoReflect().Descriptor().FullName()), nil
}
