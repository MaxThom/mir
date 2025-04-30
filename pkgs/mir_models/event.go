package mir_models

import "time"

type EventType = string

const (
	EventTypeNormal  EventType = "normal"
	EventTypeWarning EventType = "warning"
)

func NewEvent() Event {
	return Event{
		Object: Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/events",
		},
	}
}

func NewEventWithMeta(m Meta) Event {
	return Event{
		Object: Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/events",
			Meta:       m,
		},
	}
}

type Event struct {
	Object             // Todo inline json
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
	Type          EventType      `json:"type,omitempty" yaml:"status"`
	Reason        string         `json:"reason,omitempty" yaml:"reason"`
	Message       string         `json:"message,omitempty" yaml:"message"`
	Payload       map[string]any `json:"payload,omitempty" yaml:"payload"`
	RelatedObject Object         `json:"relatedObject,omitempty" yaml:"relatedObject"`
}

type EventStatus struct {
	Count   int       `json:"count,omitempty" yaml:"count"`
	FirstAt time.Time `json:"firstAt,omitempty" yaml:"firstAt"`
	LastAt  time.Time `json:"lastAt,omitempty" yaml:"lastAt"`
}

type EventUpdate struct {
	Meta   *MetaUpdate        `json:"meta,omitempty" yaml:"meta"`
	Spec   *EventUpdateSpec   `json:"spec,omitempty" yaml:"spec"`
	Status *EventUpdateStatus `json:"status,omitempty" yaml:"spec"`
}

type EventUpdateSpec struct {
	Type          *EventType     `json:"type,omitempty" yaml:"type"`
	Reason        *string        `json:"reason,omitempty" yaml:"reason"`
	Message       *string        `json:"message,omitempty" yaml:"message"`
	Payload       map[string]any `json:"payload,omitempty" yaml:"payload"`
	RelatedObject *ObjectUpdate  `json:"relatedObject,omitempty" yaml:"relatedObject"`
}

type EventUpdateStatus struct {
	Count   *int       `json:"count,omitempty" yaml:"count"`
	FirstAt *time.Time `json:"firstAt,omitempty" yaml:"firstAt"`
	LastAt  *time.Time `json:"lastAt,omitempty" yaml:"lastAt"`
}
