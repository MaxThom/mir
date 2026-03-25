package mir_v1

import "time"

type WidgetType string

const (
	WidgetTypeTelemetry WidgetType = "telemetry"
	WidgetTypeCommand   WidgetType = "command"
	WidgetTypeConfig    WidgetType = "config"
	WidgetTypeEvents    WidgetType = "events"
)

type Dashboard struct {
	Object `json:",inline" yaml:",inline"`
	Spec   DashboardSpec   `json:"spec,omitempty"`
	Status DashboardStatus `json:"status,omitempty"`
}

type DashboardSpec struct {
	Description     string            `json:"description,omitempty"`
	RefreshInterval *int              `json:"refreshInterval,omitempty"`
	TimeMinutes     *int              `json:"timeMinutes,omitempty"`
	Widgets         []DashboardWidget `json:"widgets,omitempty"`
}

type DashboardStatus struct {
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

func NewDashboard() Dashboard {
	return Dashboard{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "dashboard",
		},
		Spec: DashboardSpec{
			Widgets: []DashboardWidget{},
		},
		Status: DashboardStatus{
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}
}

func (d Dashboard) WithMeta(m Meta) Dashboard {
	d.Meta = m
	return d
}

func (d Dashboard) WithSpec(s DashboardSpec) Dashboard {
	d.Spec = s
	return d
}

type DashboardUpdate struct {
	Meta   *MetaUpdate            `json:"meta,omitempty" yaml:"meta"`
	Spec   *DashboardUpdateSpec   `json:"spec,omitempty" yaml:"spec"`
	Status *DashboardUpdateStatus `json:"status,omitempty" yaml:"spec"`
}

type DashboardUpdateSpec struct {
	Description     *string           `json:"description,omitempty"`
	RefreshInterval *int              `json:"refreshInterval,omitempty"`
	TimeMinutes     *int              `json:"timeMinutes,omitempty"`
	Widgets         []DashboardWidget `json:"widgets,omitempty"`
}

type DashboardUpdateStatus struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type TelemetryWidgetConfig struct {
	Target      ObjectTarget `json:"target"`
	Measurement string       `json:"measurement"`
	Fields      []string     `json:"fields"`
	TimeRange   string       `json:"timeRange"`
}

type CommandWidgetConfig struct {
	Target ObjectTarget `json:"target"`
}

type ConfigWidgetConfig struct {
	Target ObjectTarget `json:"target"`
}

type EventsWidgetConfig struct {
	Target ObjectTarget `json:"target"`
	Limit  int          `json:"limit"`
}

type DashboardWidget struct {
	Id     string     `json:"id"`
	Type   WidgetType `json:"type"`
	Title  string     `json:"title"`
	X      int        `json:"x"`
	Y      int        `json:"y"`
	W      int        `json:"w"`
	H      int        `json:"h"`
	Config any        `json:"config"`
}

type DashboardWidgetUpdate struct {
	Id     *string     `json:"id"`
	Type   *WidgetType `json:"type"`
	Title  *string     `json:"title"`
	X      *int        `json:"x"`
	Y      *int        `json:"y"`
	W      *int        `json:"w"`
	H      *int        `json:"h"`
	Config any         `json:"config"`
}
