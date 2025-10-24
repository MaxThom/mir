package mir_proto

import (
	"fmt"

	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewFileDescriptor(packageName, fileName string) *descriptorpb.FileDescriptorProto {
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    proto.String(fmt.Sprintf("%s/%s.proto", packageName, fileName)),
		Package: proto.String(packageName),
		Syntax:  proto.String("proto3"),
		Dependency: []string{
			"mir/device/v1/mir.proto",
		},
	}
	return fileDesc
}

func NewTelemetryDescriptor(name string, meta *devicev1.Meta, tst devicev1.TimestampType) *descriptorpb.DescriptorProto {
	// Create message descriptor
	msgDesc := &descriptorpb.DescriptorProto{
		Name:    proto.String(name),
		Options: &descriptorpb.MessageOptions{},
	}

	// Add Mir message type extension
	proto.SetExtension(msgDesc.Options, devicev1.E_MessageType, devicev1.MessageType_MESSAGE_TYPE_TELEMETRY)

	// Add message-level meta (Mir metadata)
	if meta != nil {
		proto.SetExtension(msgDesc.Options, devicev1.E_Meta, meta)
	}

	// Add TS field
	if tst == devicev1.TimestampType_TIMESTAMP_TYPE_FRACTION {
		tsField := &descriptorpb.FieldDescriptorProto{
			Name:     proto.String("ts"),
			Number:   proto.Int32(1),
			Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			TypeName: proto.String("mir.device.v1.Timestamp"),
			Options:  &descriptorpb.FieldOptions{},
		}
		proto.SetExtension(tsField.Options, devicev1.E_Timestamp, tst)
		msgDesc.Field = append(msgDesc.Field, tsField)
	} else {
		tsField := &descriptorpb.FieldDescriptorProto{
			Name:    proto.String("ts"),
			Number:  proto.Int32(1),
			Type:    descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
			Options: &descriptorpb.FieldOptions{},
		}
		proto.SetExtension(tsField.Options, devicev1.E_Timestamp, tst)
		msgDesc.Field = append(msgDesc.Field, tsField)
	}

	return msgDesc
}

func NewCommandDescriptor(name string, meta *devicev1.Meta) *descriptorpb.DescriptorProto {
	// Create message descriptor
	msgDesc := &descriptorpb.DescriptorProto{
		Name:    proto.String(name),
		Options: &descriptorpb.MessageOptions{},
	}

	// Add Mir message type extension
	proto.SetExtension(msgDesc.Options, devicev1.E_MessageType,
		devicev1.MessageType_MESSAGE_TYPE_TELECOMMAND)

	// Add message-level meta (Mir metadata)
	if meta != nil {
		proto.SetExtension(msgDesc.Options, devicev1.E_Meta, meta)
	}

	return msgDesc
}

func NewConfigDescriptor(name string, meta *devicev1.Meta) *descriptorpb.DescriptorProto {
	// Create message descriptor
	msgDesc := &descriptorpb.DescriptorProto{
		Name:    proto.String(name),
		Options: &descriptorpb.MessageOptions{},
	}

	// Add Mir message type extension
	proto.SetExtension(msgDesc.Options, devicev1.E_MessageType,
		devicev1.MessageType_MESSAGE_TYPE_TELECONFIG)

	// Add message-level meta (Mir metadata)
	if meta != nil {
		proto.SetExtension(msgDesc.Options, devicev1.E_Meta, meta)
	}

	return msgDesc
}

func NewFractionTimestampDescriptor() *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String("Timestamp"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("seconds"),
				Number: proto.Int32(1),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
			},
			{
				Name:   proto.String("nanos"),
				Number: proto.Int32(2),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
			},
		},
	}
}
