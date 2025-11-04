package mir_v1

import (
	"strings"

	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type DeviceId string

type Device struct {
	Object     `json:",inline" yaml:",inline"`
	Spec       DeviceSpec       `json:"spec,omitempty" yaml:"spec"`
	Properties DeviceProperties `json:"properties,omitempty" yaml:"properties"`
	Status     DeviceStatus     `json:"status,omitempty" yaml:"status"`
}

func NewDevice() Device {
	return Device{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
		},
	}
}

func ToTargets(d ...Device) DeviceTarget {
	t := DeviceTarget{
		Ids: []string{},
	}
	for _, i := range d {
		t.Ids = append(t.Ids, i.Spec.DeviceId)
	}
	return t
}

func (d Device) ToTarget() DeviceTarget {
	return DeviceTarget{
		Ids:        []string{d.Spec.DeviceId},
		Names:      []string{d.Meta.Name},
		Namespaces: []string{d.Meta.Namespace},
	}
}

func (d Device) WithMeta(m Meta) Device {
	d.Meta = m
	return d
}

func (d Device) WithSpec(s DeviceSpec) Device {
	d.Spec = s
	return d
}

func (d Device) WithId(id string) Device {
	d.Spec.DeviceId = id
	return d
}

func (d Device) WithProps(p DeviceProperties) Device {
	d.Properties = p
	return d
}

func (d Device) WithStatus(s DeviceStatus) Device {
	d.Status = s
	return d
}

func (d Device) GetNameNamespace() string {
	return d.Meta.Name + "/" + d.Meta.Namespace
}

func (d Device) GetNameNs() NameNs {
	return NewNameNs(d.Meta.Name, d.Meta.Namespace)
}

type DeviceTarget struct {
	Ids        []string
	Names      []string
	Namespaces []string
	Labels     map[string]string
}

func (o DeviceTarget) HasNoTarget() bool {
	return len(o.Names) == 0 &&
		len(o.Namespaces) == 0 &&
		len(o.Labels) == 0 &&
		len(o.Ids) == 0
}

func (o DeviceTarget) HasOnlyIdsTarget() bool {
	if len(o.Names) > 0 || len(o.Namespaces) > 0 || len(o.Labels) > 0 {
		return false
	}
	if len(o.Ids) == 0 {
		return false
	}
	return true
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
	Disabled *bool  `json:"disabled,omitempty" yaml:"disabled"`
}

type DeviceProperties struct {
	Desired  map[string]any `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]any `json:"reported,omitempty" yaml:"reported"`
}

type PropertiesTime struct {
	Desired  map[string]surrealdbModels.CustomDateTime `json:"desired,omitempty" yaml:"desired"`
	Reported map[string]surrealdbModels.CustomDateTime `json:"reported,omitempty" yaml:"reported"`
}

type DeviceStatus struct {
	Online         *bool                           `json:"online,omitempty" yaml:"online"`
	LastHearthbeat *surrealdbModels.CustomDateTime `json:"lastHearthbeat,omitempty" yaml:"lastHearthbeat"`
	Schema         Schema                          `json:"schema,omitempty" yaml:"schema"`
	Properties     PropertiesTime                  `json:"properties,omitempty" yaml:"properties"`
	Events         []DeviceStatusEvent             `json:"events,omitempty" yaml:"events"`
}

type DeviceStatusEvent struct {
	Type    EventType                       `json:"type,omitempty" yaml:"type"`
	Reason  string                          `json:"reason,omitempty" yaml:"reason"`
	Message string                          `json:"message,omitempty" yaml:"message"`
	FirstAt *surrealdbModels.CustomDateTime `json:"firstAt,omitempty" yaml:"firstAt"`
}

type Schema struct {
	// Compressed with ZSTD
	CompressedSchema []byte                          `json:"compressedSchema,omitempty" yaml:"-"`
	PackageNames     []string                        `json:"packageNames,omitempty" yaml:"packageNames"`
	LastSchemaFetch  *surrealdbModels.CustomDateTime `json:"lastSchemaFetch,omitempty" yaml:"lastSchemaFetch"`
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
