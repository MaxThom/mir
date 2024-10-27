package mir_utils

import (
	"fmt"

	"github.com/maxthom/mir/internal/libs/proto/proto_mir"
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
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELECOMMAND {
				boiler, err := proto_mir.GetJsonBoilerTemplate(msgDesc)
				if err != nil {
					fmt.Println(err)
				}
				cmd := cmd_apiv1.CommandDescriptor{
					Name:        string(msgDesc.FullName()),
					Description: "",
					Labels:      proto_mir.RetrieveMessageTags(msgDesc),
					Arguments:   proto_mir.RetrieveMessageArguments(msgDesc),
					Boilerplate: string(boiler),
				}
				commands = append(commands, &cmd)
			}
		}
		return true
	})
	return commands, nil
}
