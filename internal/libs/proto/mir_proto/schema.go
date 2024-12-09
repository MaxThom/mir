package mir_proto

import (
	"slices"

	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/internal/libs/proto/json_template"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
	"gopkg.in/yaml.v3"

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

func NewMirProtoSchema(s ...protoreflect.FileDescriptor) (*MirProtoSchema, error) {
	pbSet := &descriptorpb.FileDescriptorSet{}
	for _, f := range s {
		pbSet.File = append(pbSet.File,
			protodesc.ToFileDescriptorProto(f))
	}

	reg, err := protodesc.NewFiles(pbSet)
	return &MirProtoSchema{Files: reg}, err
}

func (m *MirProtoSchema) GetTelemetryList(filterMeasurements []string, filterLabels map[string]string) ([]*tlm_apiv1.TelemetryDescriptor, error) {
	telemetry := []*tlm_apiv1.TelemetryDescriptor{}
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Messages().Len(); i++ {
			msgDesc := fd.Messages().Get(i)
			opts, ok := msgDesc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				continue
			}
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELEMETRY {
				lbls := RetrieveMessageTags(msgDesc)
				if !isSubsetContainedInSet(filterLabels, lbls) ||
					(len(filterMeasurements) > 0 && !slices.Contains(filterMeasurements, string(msgDesc.FullName()))) {
					continue
				}
				tlm := tlm_apiv1.TelemetryDescriptor{
					Name:   string(msgDesc.FullName()),
					Labels: lbls,
				}
				telemetry = append(telemetry, &tlm)
			}
		}
		return true
	})
	return telemetry, nil
}

func (m *MirProtoSchema) GetCommandsList(filterLabels map[string]string) ([]*cmd_apiv1.CommandDescriptor, error) {
	commands := []*cmd_apiv1.CommandDescriptor{}
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

func (m *MirProtoSchema) GetConfigList(filterLabels map[string]string) ([]*cfg_apiv1.ConfigDescriptor, error) {
	cfgs := []*cfg_apiv1.ConfigDescriptor{}
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Messages().Len(); i++ {
			msgDesc := fd.Messages().Get(i)
			opts, ok := msgDesc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				continue
			}
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELECONFIG {
				lbls := RetrieveMessageTags(msgDesc)
				if !isSubsetContainedInSet(filterLabels, lbls) {
					continue
				}
				boiler, err := json_template.GenerateTemplate(msgDesc)
				cfg := cfg_apiv1.ConfigDescriptor{
					Name:     string(msgDesc.FullName()),
					Labels:   lbls,
					Template: string(boiler),
				}
				if err != nil {
					cfg.Error = err.Error()
				}
				cfgs = append(cfgs, &cfg)
			}
		}
		return true
	})
	return cfgs, nil
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

func (m *MirProtoSchema) ToYaml() ([]byte, error) {
	pbset := &descriptorpb.FileDescriptorSet{}
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		pbset.File = append(pbset.File, protodesc.ToFileDescriptorProto(fd))
		return true
	})

	bytes, err := yaml.Marshal(pbset)
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
	map1 := make(map[string]protoreflect.FileDescriptor)
	map2 := make(map[string]protoreflect.FileDescriptor)

	sch1.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		map1[fd.Path()] = fd
		return true
	})

	sch2.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		map2[fd.Path()] = fd
		return true
	})

	if len(map1) != len(map2) {
		return false
	}

	// Compare each file descriptor
	for path, fd1 := range map1 {
		fd2, exists := map2[path]
		if !exists {
			return false
		}

		// Compare file descriptor properties
		if fd1.Path() != fd2.Path() ||
			fd1.Package() != fd2.Package() ||
			fd1.Messages().Len() != fd2.Messages().Len() ||
			fd1.Enums().Len() != fd2.Enums().Len() ||
			fd1.Extensions().Len() != fd2.Extensions().Len() ||
			fd1.Services().Len() != fd2.Services().Len() {
			return false
		}

		// Compare actual proto content
		proto1 := protodesc.ToFileDescriptorProto(fd1)
		proto2 := protodesc.ToFileDescriptorProto(fd2)
		if !proto.Equal(proto1, proto2) {
			return false
		}
	}
	return true
}
