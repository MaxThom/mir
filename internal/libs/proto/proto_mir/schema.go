package proto_mir

import (
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/internal/libs/proto/json_template"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"

	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MirProtoSchema struct {
	*protoregistry.Files
}

func (m *MirProtoSchema) GetCommandsList(filterLabels map[string]string) ([]*cmd_apiv1.CommandDescriptor, error) {
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
				lbls := RetrieveMessageTags(msgDesc)
				if !isSubsetContainedInSet(filterLabels, lbls) {
					continue
				}
				boiler, err := json_template.GenerateTemplate(msgDesc)
				cmd := cmd_apiv1.CommandDescriptor{
					Name:     string(msgDesc.FullName()),
					Labels:   lbls,
					Template: string(boiler),
				}
				if err != nil {
					cmd.Error = err.Error()
				}
				commands = append(commands, &cmd)
			}
		}
		return true
	})
	return commands, nil
}

func (m *MirProtoSchema) GetPackageList() []string {
	packages := []string{}
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		packages = append(packages, string(fd.Package()))
		return true
	})
	return packages
}

func (m *MirProtoSchema) MarshalSchema() ([]byte, error) {
	pbset := &descriptorpb.FileDescriptorSet{}
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		pbset.File = append(pbset.File, protodesc.ToFileDescriptorProto(fd))
		return true
	})

	bytes, err := proto.Marshal(pbset)
	if err != nil {
		return []byte{}, err
	}
	return bytes, nil
}

func (m *MirProtoSchema) CompressSchema() ([]byte, error) {
	bytes, err := m.MarshalSchema()
	if err != nil {
		return []byte{}, err
	}

	b, err := zstd.CompressData(bytes)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func UnmarshalSchema(b []byte) (*MirProtoSchema, error) {
	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(b, pbSet); err != nil {
		return nil, err
	}

	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		return nil, err
	}

	return &MirProtoSchema{reg}, nil
}

func DecompressSchema(b []byte) (*MirProtoSchema, error) {
	bDecomp, err := zstd.DecompressData(b)
	if err != nil {
		return nil, err
	}

	return UnmarshalSchema(bDecomp)
}

func isSubsetContainedInSet(subset, set map[string]string) bool {
	if len(subset) > len(set) {
		return false
	}

	for key, val := range subset {
		setVal, ok := set[key]
		if !ok || val != setVal {
			return false
		}
	}

	return true
}

func AreSchemaEqual(sch1, sch2 *MirProtoSchema) bool {
	b1, err := sch1.MarshalSchema()
	if err != nil {
		return false
	}
	b2, err := sch2.MarshalSchema()
	if err != nil {
		return false
	}

	if len(b1) != len(b2) {
		return false
	}
	if string(b1) != string(b2) {
		return false
	}
	return true
}
