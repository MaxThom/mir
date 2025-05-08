package mir_models

import (
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
)

type Device struct {
	Object     ``               // TODO check line
	Spec       DeviceSpec       `json:"spec,omitempty" yaml:"spec"`
	Properties DeviceProperties `json:"properties,omitempty" yaml:"properties"`
	Status     DeviceStatus     `json:"status,omitempty" yaml:"status"`
}

func NewDevice() Device {
	return Device{
		Object: Object{
			ApiVersion: "v1alpha",
			ApiName:    "device",
		},
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

type DeviceSpec struct {
	DeviceId string `json:"deviceId,omitempty" yaml:"deviceId"`
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled"`
}

type DeviceProperties struct {
	Desired  map[string]interface{} `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]interface{} `json:"reported,omitempty" yaml:"reported"`
}

type PropertiesTime struct {
	Desired  map[string]time.Time `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]time.Time `json:"reported,omitempty" yaml:"reported"`
}

type DeviceStatus struct {
	Online         bool                `json:"online,omitempty" yaml:"online"`
	LastHearthbeat time.Time           `json:"lastHearthbeat,omitempty" yaml:"lastHearthbeat"`
	Schema         Schema              `json:"schema,omitempty" yaml:"schema"`
	Properties     PropertiesTime      `json:"properties,omitempty" yaml:"properties"`
	Events         []DeviceStatusEvent `json:"events,omitempty" yaml:"events"`
}

type DeviceStatusEvent struct {
	Type    EventType `json:"type,omitempty" yaml:"type"`
	Reason  string    `json:"reason,omitempty" yaml:"reason"`
	Message string    `json:"message,omitempty" yaml:"message"`
	FirstAt time.Time `json:"firstAt,omitempty" yaml:"firstAt"`
}

type Schema struct {
	// Compressed with ZSTD
	CompressedSchema []byte    `json:"compressedSchema,omitempty" yaml:"-"`
	PackageNames     []string  `json:"packageNames,omitempty" yaml:"packageNames"`
	LastSchemaFetch  time.Time `json:"lastSchemaFetch,omitempty" yaml:"lastSchemaFetch"`
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
