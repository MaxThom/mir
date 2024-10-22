package mir_utils

import (
	"fmt"

	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"

	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MirProtoSchema struct {
	*protoregistry.Files
}

func (m *MirProtoSchema) GetCommandsList() ([]*cmd_apiv1.CommandDescriptor, error) {
	commands := []*cmd_apiv1.CommandDescriptor{}
	// 1. Add labels
	// 2. Add arguments
	// 3. Add json template
	// 4. Add description metadata?
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Messages().Len(); i++ {
			msgDesc := fd.Messages().Get(i)
			opts, ok := msgDesc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				continue
			}
			//import "google.golang.org/protobuf/encoding/protojson"
			//m := protojson.MarshalOptions{EmitUnpopulated: true}
			//resp, err := m.Marshal(w)
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELECOMMAND {
				boiler, err := proto_lineprotocol.GetJsonBoilerTemplate(msgDesc)
				if err != nil {
					fmt.Println(err)
				}
				cmd := cmd_apiv1.CommandDescriptor{
					Name:        string(msgDesc.FullName()),
					Description: "",
					Labels:      proto_lineprotocol.RetrieveMessageTags(msgDesc),
					Arguments:   proto_lineprotocol.RetrieveMessageArguments(msgDesc),
					Boilerplate: string(boiler),
				}
				commands = append(commands, &cmd)
			}
		}
		return true
	})
	return commands, nil
}
