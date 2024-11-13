package mir_models

import (
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Device struct {
	ApiVersion string     `json:"apiVersion" yaml:"apiVersion"`
	ApiName    string     `json:"apiName" yaml:"apiName"`
	Meta       Meta       `json:"meta" yaml:"meta"`
	Spec       Spec       `json:"spec" yaml:"spec"`
	Properties Properties `json:"properties" yaml:"properties"`
	Status     Status     `json:"status,omitempty" yaml:"status,omitempty"`
}

func NewDevice() Device {
	return Device{
		ApiVersion: "v1alpha",
		ApiName:    "device",
	}
}

func (d Device) GetNameNamespace() string {
	return d.Meta.Name + "/" + d.Meta.Namespace
}

func (d Device) GetNameNs() NameNs {
	return NewNameNs(d.Meta.Name, d.Meta.Namespace)
}

type NameNs struct {
	Name      string
	Namespace string
}

func NewNameNs(name, namespace string) NameNs {
	return NameNs{
		Name:      name,
		Namespace: namespace,
	}
}

func FromNameNsString(nameNs string) NameNs {
	s := strings.Split(nameNs, "/")
	if len(s) == 1 {
		return NameNs{
			Name:      s[0],
			Namespace: "default",
		}
	}
	return NameNs{
		Name:      s[0],
		Namespace: s[1],
	}
}
func (d NameNs) GetNameNamespace() string {
	return d.Name + "/" + d.Namespace
}

type Meta struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace" yaml:"namespace"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
}

type Spec struct {
	DeviceId string `json:"deviceId" yaml:"deviceId"`
	Disabled bool   `json:"disabled" yaml:"disabled"`
}

type Properties struct {
}

type Status struct {
	Online         bool      `json:"online" yaml:"online"`
	LastHearthbeat time.Time `json:"lastHearthbeat" yaml:"lastHearthbeat"`
	Schema         Schema    `json:"schema" yaml:"schema"`
}

type Schema struct {
	// Compressed with ZSTD
	CompressedSchema []byte    `json:"compressedSchema" yaml:"-"`
	PackageNames     []string  `json:"packageNames" yaml:"packageNames"`
	LastSchemaFetch  time.Time `json:"lastSchemaFetch" yaml:"lastSchemaFetch"`
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
