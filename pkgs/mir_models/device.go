package mir_models

import (
	"time"

	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Device struct {
	ApiVersion string     `json:"apiVersion"`
	ApiName    string     `json:"apiName"`
	Meta       Meta       `json:"meta"`
	Spec       Spec       `json:"spec"`
	Properties Properties `json:"properties"`
	Status     Status     `json:"status"`
}

func (d Device) GetNameNamespace() string {
	return d.Meta.Name + "/" + d.Meta.Namespace
}

type Meta struct {
	Name        string             `json:"name"`
	Namespace   string             `json:"namespace"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type Spec struct {
	DeviceId string `json:"deviceId"`
	Disabled bool   `json:"disabled"`
}

type Properties struct {
}

type Status struct {
	Online         bool      `json:"online"`
	LastHearthbeat time.Time `json:"lastHearthbeat"`
	Schema         Schema    `json:"schema"`
}

type Schema struct {
	// Compressed with ZSTD
	CompressedSchema []byte    `json:"compressedSchema"`
	PackageNames     []string  `json:"packageNames"`
	LastSchemaFetch  time.Time `json:"lastSchemaFetch"`
}

func (s Schema) GetProtoFiles() (*protoregistry.Files, error) {
	_, reg, err := DecompressFileDescriptorSet(s.CompressedSchema)
	return reg, err
}

func (s *Schema) SetProtoSchema(desc *descriptorpb.FileDescriptorSet) error {
	b, err := CompressFileDescriptorSet(desc)
	if err != nil {
		return err
	}
	s.CompressedSchema = b
	return nil
}

func MarshalProtoFiles(s ...protoreflect.FileDescriptor) ([]byte, error) {
	pbSet := &descriptorpb.FileDescriptorSet{}
	for _, f := range s {
		pbSet.File = append(pbSet.File,
			protodesc.ToFileDescriptorProto(f))
	}

	bytes, err := proto.Marshal(pbSet)
	if err != nil {
		return []byte{}, err
	}
	return bytes, nil
}

func CompressProtoFiles(s ...protoreflect.FileDescriptor) ([]byte, error) {
	pbSet := new(descriptorpb.FileDescriptorSet)
	for _, f := range s {
		pbSet.File = append(pbSet.File,
			protodesc.ToFileDescriptorProto(f))
	}
	return CompressFileDescriptorSet(pbSet)
}

func CompressFileDescriptorSet(desc *descriptorpb.FileDescriptorSet) ([]byte, error) {
	bytes, err := proto.Marshal(desc)
	if err != nil {
		return []byte{}, err
	}
	b, err := zstd.CompressData(bytes)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func DecompressFileDescriptorSet(b []byte) (*descriptorpb.FileDescriptorSet, *protoregistry.Files, error) {
	bDecomp, err := zstd.DecompressData(b)
	if err != nil {
		return nil, nil, err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(bDecomp, pbSet); err != nil {
		return nil, nil, err
	}

	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		return nil, nil, err
	}
	return pbSet, reg, nil
}
