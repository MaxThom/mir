package mir_models

import (
	"strings"

	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type Device struct {
	ApiVersion string     `json:"apiVersion,omitempty" yaml:"apiVersion"`
	ApiName    string     `json:"apiName,omitempty" yaml:"apiName"`
	Meta       Meta       `json:"meta,omitempty" yaml:"meta"`
	Spec       Spec       `json:"spec,omitempty" yaml:"spec"`
	Properties Properties `json:"properties,omitempty" yaml:"properties"`
	Status     Status     `json:"status,omitempty" yaml:"status"`
}

func NewDevice() Device {
	return Device{
		ApiVersion: "v1alpha",
		ApiName:    "device",
		// Properties: Properties{
		// 	Desired:  make(map[string]interface{}),
		// 	Reported: make(map[string]interface{}),
		// },
	}
}

func (d Device) GetNameNamespace() string {
	return d.Meta.Name + "/" + d.Meta.Namespace
}

func (d Device) GetNameNs() NameNs {
	return NewNameNs(d.Meta.Name, d.Meta.Namespace)
}

type Targets struct {
	Ids        []string
	Names      []string
	Namespaces []string
	Labels     map[string]string
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
	Name        string            `json:"name,omitempty" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`
}

type Spec struct {
	DeviceId string `json:"deviceId,omitempty" yaml:"deviceId"`
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled"`
}

type Properties struct {
	Desired  map[string]interface{} `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]interface{} `json:"reported,omitempty" yaml:"reported"`
}

type PropertiesTime struct {
	Desired  map[string]surrealdbModels.CustomDateTime `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]surrealdbModels.CustomDateTime `json:"reported,omitempty" yaml:"reported"`
}

type Status struct {
	Online         bool                           `json:"online,omitempty" yaml:"online"`
	LastHearthbeat surrealdbModels.CustomDateTime `json:"lastHearthbeat,omitempty" yaml:"lastHearthbeat"`
	Schema         Schema                         `json:"schema,omitempty" yaml:"schema"`
	Properties     PropertiesTime                 `json:"properties,omitempty" yaml:"properties"`
}

type Schema struct {
	// Compressed with ZSTD
	CompressedSchema []byte                         `json:"compressedSchema,omitempty" yaml:"-"`
	PackageNames     []string                       `json:"packageNames,omitempty" yaml:"packageNames"`
	LastSchemaFetch  surrealdbModels.CustomDateTime `json:"lastSchemaFetch,omitempty" yaml:"lastSchemaFetch"`
}

func (s Schema) GetProtoFiles() (*mir_proto.MirProtoSchema, error) {
	return mir_proto.DecompressSchema(s.CompressedSchema)
}

func (s *Schema) SetProtoSchema(m *mir_proto.MirProtoSchema) error {
	b, err := m.CompressSchema()
	if err != nil {
		return err
	}
	s.CompressedSchema = b
	s.PackageNames = m.GetPackageList()
	return nil
}
