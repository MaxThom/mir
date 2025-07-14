package mir_v1

import (
	"github.com/maxthom/mir/internal/libs/jsonyaml"
	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
)

type EventType = string

const (
	EventTypeNormal  EventType = "normal"
	EventTypeWarning EventType = "warning"
)

type EventTarget struct {
	ObjectTarget
	DateFilter DateFilter
	Limit      int
}

func (o EventTarget) HasNoTarget() bool {
	return len(o.Names) == 0 &&
		len(o.Namespaces) == 0 &&
		len(o.Labels) == 0 &&
		o.DateFilter.From.IsZero() &&
		o.DateFilter.To.IsZero()
}

func NewEvent() Event {
	return Event{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "events",
		},
	}
}

func (e Event) WithMeta(m Meta) Event {
	e.Meta = m
	return e
}

type Event struct {
	Object `json:",inline" yaml:",inline"`
	Spec   EventSpec   `json:"spec,omitempty" yaml:"spec"`
	Status EventStatus `json:"status,omitempty" yaml:"status"`
}

func (e Event) WithSpec(spec EventSpec) Event {
	e.Spec = spec
	return e
}

func (e Event) WithStatus(status EventStatus) Event {
	e.Status = status
	return e
}

type EventSpec struct {
	Type          EventType           `json:"type,omitempty" yaml:"type"`
	Reason        string              `json:"reason,omitempty" yaml:"reason"`
	Message       string              `json:"message,omitempty" yaml:"message"`
	Payload       jsonyaml.RawMessage `json:"payload,omitempty" yaml:"-"`
	RelatedObject Object              `json:"relatedObject,omitempty" yaml:"relatedObject"`
}

type EventStatus struct {
	Count   int                             `json:"count,omitempty" yaml:"count"`
	FirstAt *surrealdbModels.CustomDateTime `json:"firstAt,omitempty" yaml:"firstAt"`
	LastAt  *surrealdbModels.CustomDateTime `json:"lastAt,omitempty" yaml:"lastAt"`
}

type EventUpdate struct {
	Meta   *MetaUpdate        `json:"meta,omitempty" yaml:"meta"`
	Spec   *EventUpdateSpec   `json:"spec,omitempty" yaml:"spec"`
	Status *EventUpdateStatus `json:"status,omitempty" yaml:"spec"`
}

type EventUpdateSpec struct {
	Type          *EventType           `json:"type,omitempty" yaml:"type"`
	Reason        *string              `json:"reason,omitempty" yaml:"reason"`
	Message       *string              `json:"message,omitempty" yaml:"message"`
	Payload       *jsonyaml.RawMessage `json:"payload,omitempty" yaml:"payload"`
	RelatedObject *ObjectUpdate        `json:"relatedObject,omitempty" yaml:"relatedObject"`
}

type EventUpdateStatus struct {
	Count   *int                            `json:"count,omitempty" yaml:"count"`
	FirstAt *surrealdbModels.CustomDateTime `json:"firstAt,omitempty" yaml:"firstAt"`
	LastAt  *surrealdbModels.CustomDateTime `json:"lastAt,omitempty" yaml:"lastAt"`
}
