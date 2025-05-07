package mir_models

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	event_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/event_api"
	"google.golang.org/protobuf/types/known/structpb"
)

// Devices

func NewDeviceListFromProtoDevices(d []*core_apiv1.Device) []Device {
	p := []Device{}
	for _, v := range d {
		dev := NewDeviceFromProtoDevice(v)
		p = append(p, dev)
	}
	return p
}

func NewDeviceFromProtoDevice(d *core_apiv1.Device) Device {
	dev := NewDevice()
	if d == nil {
		return dev
	}
	if d.Meta != nil {
		dev.Meta = Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		}
	}
	if d.Spec != nil {
		dev.Spec = DeviceSpec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		}
	}
	if d.Properties != nil {
		dev.Properties.Desired = d.Properties.Desired.AsMap()
		dev.Properties.Reported = d.Properties.Reported.AsMap()
	}
	if d.Status != nil {
		var lastHeartbeatTime time.Time
		if d.Status.LastHearthbeat != nil {
			lastHeartbeatTime = AsGoTime(d.Status.LastHearthbeat)
		}
		dev.Status = DeviceStatus{
			Online:         d.Status.Online,
			LastHearthbeat: lastHeartbeatTime,
		}
		if d.Status.Schema != nil {
			var lastSchemaFetch time.Time
			if d.Status.Schema.LastSchemaFetch != nil {
				lastSchemaFetch = AsGoTime(d.Status.Schema.LastSchemaFetch)
			}
			dev.Status.Schema = Schema{
				CompressedSchema: d.Status.Schema.CompressedSchema,
				PackageNames:     d.Status.Schema.PackageNames,
				LastSchemaFetch:  lastSchemaFetch,
			}
		}
		if d.Status.Properties != nil {
			dev.Status.Properties = PropertiesTime{}
			if d.Status.Properties.Desired != nil {
				dev.Status.Properties.Desired = map[string]time.Time{}
				for k, v := range d.Status.Properties.Desired {
					dev.Status.Properties.Desired[k] = AsGoTime(v)
				}
			}
			if d.Status.Properties.Reported != nil {
				dev.Status.Properties.Reported = map[string]time.Time{}
				for k, v := range d.Status.Properties.Reported {
					dev.Status.Properties.Reported[k] = AsGoTime(v)
				}
			}
		}
	}

	return dev
}

func NewProtoDeviceListFromDevices(d []Device) []*core_apiv1.Device {
	p := []*core_apiv1.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDevice(v))
	}
	return p
}

func NewProtoDeviceFromDevice(d Device) *core_apiv1.Device {
	des, _ := structpb.NewStruct(d.Properties.Desired)
	rep, _ := structpb.NewStruct(d.Properties.Reported)
	return &core_apiv1.Device{
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: &core_apiv1.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		},
		Spec: &core_apiv1.Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Properties: &core_apiv1.Properties{
			Desired:  des,
			Reported: rep,
		},
		Status: &core_apiv1.Status{
			Online:         d.Status.Online,
			LastHearthbeat: AsProtoTimestamp(d.Status.LastHearthbeat),
			Schema: &core_apiv1.Schema{
				CompressedSchema: d.Status.Schema.CompressedSchema,
				PackageNames:     d.Status.Schema.PackageNames,
				LastSchemaFetch:  AsProtoTimestamp(d.Status.Schema.LastSchemaFetch),
			},
			Properties: &core_apiv1.PropertiesTime{
				Desired:  mapToProtoTs(d.Status.Properties.Desired),
				Reported: mapToProtoTs(d.Status.Properties.Reported),
			},
		},
	}
}

func NewUpdateDeviceReqFromDeviceWithTarget(t Targets, d Device) *core_apiv1.UpdateDeviceRequest {
	dev := NewUpdateDeviceReqFromDevice(d)
	dev.Targets = &core_apiv1.Targets{
		Ids:        t.Ids,
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
	}
	return dev
}
func NewUpdateDeviceReqFromDeviceWithNameNs(n NameNs, d Device) *core_apiv1.UpdateDeviceRequest {
	dev := NewUpdateDeviceReqFromDevice(d)
	dev.Targets = &core_apiv1.Targets{
		Names:      []string{n.Name},
		Namespaces: []string{n.Namespace},
	}
	return dev
}

func NewUpdateDeviceReqFromDevice(d Device) *core_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]string) map[string]*common_apiv1.OptString {
		opt := map[string]*common_apiv1.OptString{}
		for k, v := range m {
			if strings.ToLower(v) == "null" {
				opt[k] = &common_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &common_apiv1.OptString{
					Value: &v,
				}
			}
		}
		return opt
	}
	des, _ := structpb.NewStruct(d.Properties.Desired)

	devUpd := &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &core_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: &d.Spec.DeviceId,
			Disabled: &d.Spec.Disabled,
		},
		Props: &core_apiv1.UpdateDeviceRequest_Properties{
			Desired: des,
		},
		Targets: &core_apiv1.Targets{
			Names:      []string{d.Meta.Name},
			Namespaces: []string{d.Meta.Namespace},
		},
	}
	if d.Spec.DeviceId == "" {
		devUpd.Spec.DeviceId = nil
	}
	if d.Properties.Desired == nil && d.Properties.Reported == nil {
		devUpd.Props = nil
	}

	return devUpd
}

func NewUpdateDeviceReqFromProtoDevice(t *core_apiv1.Targets, d *core_apiv1.Device) *core_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]string) map[string]*common_apiv1.OptString {
		opt := map[string]*common_apiv1.OptString{}
		for k, v := range m {
			if v == "none" {
				opt[k] = &common_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &common_apiv1.OptString{
					Value: &v,
				}
			}
		}
		return opt
	}
	devUpd := &core_apiv1.UpdateDeviceRequest{
		Targets: t,
	}
	if d.Meta != nil {
		devUpd.Meta = &core_apiv1.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		}
		if d.Meta.Name == "" {
			devUpd.Meta.Name = nil
		}
		if d.Meta.Namespace == "" {
			devUpd.Meta.Namespace = nil
		}
	}
	if d.Spec != nil {
		devUpd.Spec = &core_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: &d.Spec.DeviceId,
			Disabled: &d.Spec.Disabled,
		}
		if d.Spec.DeviceId == "" {
			devUpd.Spec.DeviceId = nil
		}
	}
	if d.Properties != nil {
		devUpd.Props = &core_apiv1.UpdateDeviceRequest_Properties{
			Desired: d.Properties.Desired,
		}
	}

	return devUpd
}

func NewCreateDeviceReqFromDevice(d Device) *core_apiv1.CreateDeviceRequest {
	return &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		},
		Spec: &core_apiv1.Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Properties: &core_apiv1.Properties{},
	}
}

func NewCreateDeviceReqFromDeviceUpdateRequest(d *core_apiv1.UpdateDeviceRequest) *core_apiv1.CreateDeviceRequest {
	toMap := func(m map[string]*common_apiv1.OptString) map[string]string {
		opt := map[string]string{}
		for k, v := range m {
			if v != nil && v.Value != nil {
				opt[k] = *v.Value
			}
		}
		return opt
	}
	dev := &core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Labels:      toMap(d.Meta.Labels),
			Annotations: toMap(d.Meta.Annotations),
		},
		Spec:       &core_apiv1.Spec{},
		Properties: &core_apiv1.Properties{},
	}
	if d.Meta != nil {
		if d.Meta.Name != nil {
			dev.Meta.Name = *d.Meta.Name
		}
		if d.Meta.Namespace != nil {
			dev.Meta.Namespace = *d.Meta.Namespace
		}
	}
	if d.Spec != nil {
		if d.Spec.DeviceId != nil {
			dev.Spec.DeviceId = *d.Spec.DeviceId
		}
		if d.Spec.Disabled != nil {
			dev.Spec.Disabled = *d.Spec.Disabled
		}
	}
	return dev
}

func NewDeviceFromCreateDeviceReq(d *core_apiv1.CreateDeviceRequest) Device {
	return Device{
		Object: Object{
			ApiVersion: "v1alpha",
			ApiName:    "device",
			Meta: Meta{
				Name:        d.Meta.Name,
				Namespace:   d.Meta.Namespace,
				Labels:      d.Meta.Labels,
				Annotations: d.Meta.Annotations,
			},
		},
		Spec: DeviceSpec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Status: DeviceStatus{},
	}
}

/// Objects

func MirObjectTargetToProtoObjectTarget(t ObjectTarget) *common_apiv1.Targets {
	return &common_apiv1.Targets{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
	}
}

func ProtoObjectTargetToMirObjectTarget(t *common_apiv1.Targets) ObjectTarget {
	if t == nil {
		return ObjectTarget{}
	}
	return ObjectTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
	}
}

func ProtoObjectToMirObject(o *common_apiv1.Object) Object {
	if o == nil {
		return Object{}
	}
	return Object{
		ApiVersion: o.ApiVersion,
		ApiName:    o.ApiName,
		Meta: Meta{
			Name:        o.Meta.Name,
			Namespace:   o.Meta.Namespace,
			Labels:      o.Meta.Labels,
			Annotations: o.Meta.Annotations,
		},
	}
}

func MirObjectToProtoObject(o Object) *common_apiv1.Object {
	return &common_apiv1.Object{
		ApiVersion: o.ApiVersion,
		ApiName:    o.ApiName,
		Meta: &common_apiv1.Meta{
			Name:        o.Meta.Name,
			Namespace:   o.Meta.Namespace,
			Labels:      o.Meta.Labels,
			Annotations: o.Meta.Annotations,
		},
	}
}

/// Events

func MirEventToProtoEvent(e Event) *event_apiv1.Event {
	pb, err := structpb.NewStruct(e.Spec.Payload)
	if err != nil {
		return nil
	}
	return &event_apiv1.Event{
		Object: MirObjectToProtoObject(e.Object),
		Spec: &event_apiv1.EventSpec{
			Type:          e.Spec.Type,
			Reason:        e.Spec.Reason,
			Message:       e.Spec.Message,
			Payload:       pb,
			RelatedObject: MirObjectToProtoObject(e.Spec.RelatedObject),
		},
		Status: &event_apiv1.EventStatus{
			Count:   int32(e.Status.Count),
			FirstAt: AsProtoTimestamp(e.Status.FirstAt),
			LastAt:  AsProtoTimestamp(e.Status.LastAt),
		},
	}
}
func MirEventSpecToProtoCreateEvent(e EventSpec) *event_apiv1.CreateEventRequest {
	pb, err := structpb.NewStruct(e.Payload)
	if err != nil {
		return nil
	}
	return &event_apiv1.CreateEventRequest{
		Spec: &event_apiv1.EventSpec{
			Type:          e.Type,
			Reason:        e.Reason,
			Message:       e.Message,
			Payload:       pb,
			RelatedObject: MirObjectToProtoObject(e.RelatedObject),
		},
	}
}
func ProtoEventsToMirEvents(events []*event_apiv1.Event) []Event {
	var mirEvents []Event
	for _, event := range events {
		mirEvents = append(mirEvents, ProtoEventToMirEvent(event))
	}
	return mirEvents
}

func MirEventsToProtoEvents(events []Event) []*event_apiv1.Event {
	var protoEvents []*event_apiv1.Event
	for _, event := range events {
		protoEvents = append(protoEvents, MirEventToProtoEvent(event))
	}
	return protoEvents
}

func ProtoEventToMirEvent(e *event_apiv1.Event) Event {
	if e == nil {
		return Event{}
	}
	o := ProtoObjectToMirObject(e.Spec.RelatedObject)
	return Event{
		Object: Object{
			ApiVersion: "v1alpha",
			ApiName:    "event",
			Meta: Meta{
				Name:        e.Object.Meta.Name,
				Namespace:   e.Object.Meta.Namespace,
				Labels:      e.Object.Meta.Labels,
				Annotations: e.Object.Meta.Annotations,
			},
		},
		Spec: EventSpec{
			Type:          e.Spec.Type,
			Reason:        e.Spec.Reason,
			Message:       e.Spec.Message,
			Payload:       e.Spec.Payload.AsMap(),
			RelatedObject: o,
		},
		Status: EventStatus{
			Count:   int(e.Status.Count),
			FirstAt: AsGoTime(e.Status.FirstAt),
			LastAt:  AsGoTime(e.Status.LastAt),
		},
	}
}

func ProtoCreateEventReqToMirEventSpec(e *event_apiv1.CreateEventRequest) EventSpec {
	if e == nil || e.Spec == nil {
		return EventSpec{}
	}
	o := ProtoObjectToMirObject(e.Spec.RelatedObject)
	return EventSpec{
		Type:          e.Spec.Type,
		Reason:        e.Spec.Reason,
		Message:       e.Spec.Message,
		Payload:       e.Spec.Payload.AsMap(),
		RelatedObject: o,
	}
}

func ProtoEventTargetToMirEventTarget(t *event_apiv1.ListEventsRequest) EventTarget {
	if t == nil {
		return EventTarget{}
	}
	return EventTarget{
		ObjectTarget: ProtoObjectTargetToMirObjectTarget(t.Targets),
		DateFilter:   ProtoTimeFilterToMirTimeFilter(t.FilterDate),
		Limit:        int(t.FilterLimit),
	}
}

func MirEventTargetToProtoEventTarget(t EventTarget) *event_apiv1.ListEventsRequest {
	return &event_apiv1.ListEventsRequest{
		Targets:     MirObjectTargetToProtoObjectTarget(t.ObjectTarget),
		FilterDate:  MirTimeFilterToProtoTimeFilter(t.DateFilter),
		FilterLimit: int32(t.Limit),
	}
}

/// Utils

func AsProtoTimestamp(t time.Time) *common_apiv1.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &common_apiv1.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsGoTime(ts *common_apiv1.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}

func mapToProtoTs(m map[string]time.Time) map[string]*common_apiv1.Timestamp {
	ts := map[string]*common_apiv1.Timestamp{}
	for k, v := range m {
		ts[k] = AsProtoTimestamp(v)
	}
	return ts
}

func StructToMapAny(s interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result, errors.New("argument is not a struct")
	}
	data, err := json.Marshal(s)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func ProtoTimeFilterToMirTimeFilter(f *common_apiv1.DateFilter) DateFilter {
	if f == nil {
		return DateFilter{}
	}
	return DateFilter{
		To:   AsGoTime(f.To),
		From: AsGoTime(f.From),
	}
}

func MirTimeFilterToProtoTimeFilter(f DateFilter) *common_apiv1.DateFilter {
	return &common_apiv1.DateFilter{
		To:   AsProtoTimestamp(f.To),
		From: AsProtoTimestamp(f.From),
	}
}
