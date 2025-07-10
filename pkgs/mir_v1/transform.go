package mir_v1

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"google.golang.org/protobuf/types/known/structpb"

	surrealdbModels "github.com/maxthom/surrealdb.go/pkg/models"
)

// Devices

func NewDeviceListFromProtoDevices(d []*mir_apiv1.Device) []Device {
	p := []Device{}
	for _, v := range d {
		dev := NewDeviceFromProtoDevice(v)
		p = append(p, dev)
	}
	return p
}

func NewDeviceFromProtoDevice(d *mir_apiv1.Device) Device {
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
			Disabled: &d.Spec.Disabled,
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
			Online:         &d.Status.Online,
			LastHearthbeat: &surrealdbModels.CustomDateTime{Time: lastHeartbeatTime},
		}
		if d.Status.Schema != nil {
			var lastSchemaFetch time.Time
			if d.Status.Schema.LastSchemaFetch != nil {
				lastSchemaFetch = AsGoTime(d.Status.Schema.LastSchemaFetch)
			}
			dev.Status.Schema = Schema{
				CompressedSchema: d.Status.Schema.CompressedSchema,
				PackageNames:     d.Status.Schema.PackageNames,
				LastSchemaFetch:  &surrealdbModels.CustomDateTime{Time: lastSchemaFetch},
			}
		}
		if d.Status.Properties != nil {
			dev.Status.Properties = PropertiesTime{}
			if d.Status.Properties.Desired != nil {
				dev.Status.Properties.Desired = map[string]surrealdbModels.CustomDateTime{}
				for k, v := range d.Status.Properties.Desired {
					dev.Status.Properties.Desired[k] = surrealdbModels.CustomDateTime{Time: AsGoTime(v)}
				}
			}
			if d.Status.Properties.Reported != nil {
				dev.Status.Properties.Reported = map[string]surrealdbModels.CustomDateTime{}
				for k, v := range d.Status.Properties.Reported {
					dev.Status.Properties.Reported[k] = surrealdbModels.CustomDateTime{Time: AsGoTime(v)}
				}
			}
		}
		if len(d.Status.Events) > 0 {
			dev.Status.Events = []DeviceStatusEvent{}
			for _, e := range d.Status.Events {
				if e != nil {
					dev.Status.Events = append(dev.Status.Events, DeviceStatusEvent{
						Type:    e.Type,
						Message: e.Message,
						Reason:  e.Reason,
						FirstAt: surrealdbModels.CustomDateTime{Time: AsGoTime(e.FirstAt)},
					})
				}
			}
		}
	}

	return dev
}

func NewProtoDeviceListFromDevices(d []Device) []*mir_apiv1.Device {
	p := []*mir_apiv1.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDevice(v))
	}
	return p
}

func NewProtoDeviceFromDevice(d Device) *mir_apiv1.Device {
	des, _ := structpb.NewStruct(d.Properties.Desired)
	rep, _ := structpb.NewStruct(d.Properties.Reported)

	evts := []*mir_apiv1.DeviceStatusEvent{}
	for _, e := range d.Status.Events {
		evts = append(evts, &mir_apiv1.DeviceStatusEvent{
			Type:    e.Type,
			Message: e.Message,
			Reason:  e.Reason,
			FirstAt: AsProtoTimestamp(e.FirstAt.Time),
		})
	}

	return &mir_apiv1.Device{
		ApiVersion: d.ApiVersion,
		Kind:       d.Kind,
		Meta: &mir_apiv1.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: d.Spec.DeviceId,
			Disabled: asUnRefBool(d.Spec.Disabled),
		},
		Properties: &mir_apiv1.DeviceProperties{
			Desired:  des,
			Reported: rep,
		},
		Status: &mir_apiv1.DeviceStatus{
			Online:         asUnRefBool(d.Status.Online),
			LastHearthbeat: AsRefProtoTimestampFromSurreal(d.Status.LastHearthbeat),
			Schema: &mir_apiv1.Schema{
				CompressedSchema: d.Status.Schema.CompressedSchema,
				PackageNames:     d.Status.Schema.PackageNames,
				LastSchemaFetch:  AsRefProtoTimestampFromSurreal(d.Status.Schema.LastSchemaFetch),
			},
			Properties: &mir_apiv1.PropertiesTime{
				Desired:  mapToProtoTs(d.Status.Properties.Desired),
				Reported: mapToProtoTs(d.Status.Properties.Reported),
			},
			Events: evts,
		},
	}
}

func NewUpdateDeviceReqFromDeviceWithTarget(t DeviceTarget, d Device) *mir_apiv1.UpdateDeviceRequest {
	dev := NewUpdateDeviceReqFromDevice(d)
	dev.Targets = MirDeviceTargetToProtoDeviceTarget(t)
	return dev
}

func NewUpdateDeviceReqFromDeviceWithNameNs(n NameNs, d Device) *mir_apiv1.UpdateDeviceRequest {
	dev := NewUpdateDeviceReqFromDevice(d)
	dev.Targets = &mir_apiv1.DeviceTarget{
		Names:      []string{n.Name},
		Namespaces: []string{n.Namespace},
	}
	return dev
}

func NewUpdateDeviceReqFromDevice(d Device) *mir_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]string) map[string]*mir_apiv1.OptString {
		opt := map[string]*mir_apiv1.OptString{}
		for k, v := range m {
			if strings.ToLower(v) == "null" {
				opt[k] = &mir_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &mir_apiv1.OptString{
					Value: &v,
				}
			}
		}
		return opt
	}
	des, _ := structpb.NewStruct(d.Properties.Desired)

	devUpd := &mir_apiv1.UpdateDeviceRequest{
		Meta: &mir_apiv1.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &mir_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: &d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Props: &mir_apiv1.UpdateDeviceRequest_Properties{
			Desired: des,
		},
		Targets: &mir_apiv1.DeviceTarget{
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

func NewUpdateDeviceReqFromProtoDevice(t *mir_apiv1.DeviceTarget, d *mir_apiv1.Device) *mir_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]string) map[string]*mir_apiv1.OptString {
		opt := map[string]*mir_apiv1.OptString{}
		for k, v := range m {
			if v == "none" {
				opt[k] = &mir_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &mir_apiv1.OptString{
					Value: &v,
				}
			}
		}
		return opt
	}
	devUpd := &mir_apiv1.UpdateDeviceRequest{
		Targets: t,
	}
	if d.Meta != nil {
		devUpd.Meta = &mir_apiv1.UpdateDeviceRequest_Meta{
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
		devUpd.Spec = &mir_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: &d.Spec.DeviceId,
			Disabled: &d.Spec.Disabled,
		}
		if d.Spec.DeviceId == "" {
			devUpd.Spec.DeviceId = nil
		}
	}
	if d.Properties != nil {
		devUpd.Props = &mir_apiv1.UpdateDeviceRequest_Properties{
			Desired: d.Properties.Desired,
		}
	}

	return devUpd
}

func NewDeviceFromUpdateDeviceReq(d *mir_apiv1.UpdateDeviceRequest) Device {
	dev := NewDevice()
	if d == nil {
		return dev
	}
	if d.Meta != nil {
		if d.Meta.Name != nil {
			dev.Meta.Name = *d.Meta.Name
		}
		if d.Meta.Namespace != nil {
			dev.Meta.Namespace = *d.Meta.Namespace
		}
		if d.Meta.Labels != nil {
			dev.Meta.Labels = make(map[string]string)
			for k, v := range d.Meta.Labels {
				if v != nil && v.Value != nil {
					dev.Meta.Labels[k] = *v.Value
				}
			}
		}
		if d.Meta.Annotations != nil {
			dev.Meta.Annotations = make(map[string]string)
			for k, v := range d.Meta.Annotations {
				if v != nil && v.Value != nil {
					dev.Meta.Annotations[k] = *v.Value
				}
			}
		}
	}
	if d.Spec != nil {
		if d.Spec.DeviceId != nil {
			dev.Spec.DeviceId = *d.Spec.DeviceId
		}
		if d.Spec.Disabled != nil {
			dev.Spec.Disabled = d.Spec.Disabled
		}
	}
	if d.Props != nil && d.Props.Desired != nil {
		dev.Properties.Desired = d.Props.Desired.AsMap()
	}

	if d.Status != nil {
		if d.Status.Online != nil {
			dev.Status.Online = d.Status.Online
		}
		if d.Status.LastHearthbeat != nil {
			heartbeat := AsGoTime(d.Status.LastHearthbeat)
			dev.Status.LastHearthbeat = &surrealdbModels.CustomDateTime{Time: heartbeat}
		}
		if d.Status.Schema != nil {
			if d.Status.Schema.CompressedSchema != nil && len(d.Status.Schema.CompressedSchema) > 0 {
				dev.Status.Schema.CompressedSchema = d.Status.Schema.CompressedSchema
			}
			if d.Status.Schema.PackageNames != nil && len(d.Status.Schema.PackageNames) > 0 {
				dev.Status.Schema.PackageNames = d.Status.Schema.PackageNames
			}
			if d.Status.Schema.LastSchemaFetch != nil {
				lastFetch := AsGoTime(d.Status.Schema.LastSchemaFetch)
				dev.Status.Schema.LastSchemaFetch = &surrealdbModels.CustomDateTime{Time: lastFetch}
			}
		}
	}
	return dev
}

func NewCreateDeviceReqFromDevice(d Device) *mir_apiv1.CreateDeviceRequest {
	return &mir_apiv1.CreateDeviceRequest{
		Meta: &mir_apiv1.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: d.Spec.DeviceId,
			Disabled: asUnRefBool(d.Spec.Disabled),
		},
		Properties: &mir_apiv1.DeviceProperties{},
	}
}

func DeviceToUpdateDeviceRequest(d Device) *mir_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]string) map[string]*mir_apiv1.OptString {
		opt := map[string]*mir_apiv1.OptString{}
		for k, v := range m {
			if strings.ToLower(v) == "null" {
				opt[k] = &mir_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &mir_apiv1.OptString{
					Value: &v,
				}
			}
		}
		return opt
	}

	des, _ := structpb.NewStruct(d.Properties.Desired)

	devUpd := &mir_apiv1.UpdateDeviceRequest{
		Meta: &mir_apiv1.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &mir_apiv1.UpdateDeviceRequest_Spec{
			DeviceId: &d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Props: &mir_apiv1.UpdateDeviceRequest_Properties{
			Desired: des,
		},
	}

	if d.Spec.DeviceId == "" {
		devUpd.Spec.DeviceId = nil
	}

	if d.Properties.Desired == nil && d.Properties.Reported == nil {
		devUpd.Props = nil
	}

	devUpd.Status = &mir_apiv1.UpdateDeviceRequest_Status{
		Online:         d.Status.Online,
		LastHearthbeat: AsRefProtoTimestampFromSurreal(d.Status.LastHearthbeat),
	}

	if d.Status.Schema.CompressedSchema != nil || d.Status.Schema.PackageNames != nil || d.Status.Schema.LastSchemaFetch != nil {
		devUpd.Status.Schema = &mir_apiv1.UpdateDeviceRequest_Schema{
			CompressedSchema: d.Status.Schema.CompressedSchema,
			PackageNames:     d.Status.Schema.PackageNames,
			LastSchemaFetch:  AsRefProtoTimestampFromSurreal(d.Status.Schema.LastSchemaFetch),
		}
	}

	return devUpd
}

func NewCreateDeviceReqFromDeviceUpdateRequest(d *mir_apiv1.UpdateDeviceRequest) *mir_apiv1.CreateDeviceRequest {
	toMap := func(m map[string]*mir_apiv1.OptString) map[string]string {
		opt := map[string]string{}
		for k, v := range m {
			if v != nil && v.Value != nil {
				opt[k] = *v.Value
			}
		}
		return opt
	}
	dev := &mir_apiv1.CreateDeviceRequest{
		Meta: &mir_apiv1.Meta{
			Labels:      toMap(d.Meta.Labels),
			Annotations: toMap(d.Meta.Annotations),
		},
		Spec:       &mir_apiv1.DeviceSpec{},
		Properties: &mir_apiv1.DeviceProperties{},
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

func NewDeviceFromCreateDeviceReq(d *mir_apiv1.CreateDeviceRequest) Device {
	return Device{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: Meta{
				Name:        d.Meta.Name,
				Namespace:   d.Meta.Namespace,
				Labels:      d.Meta.Labels,
				Annotations: d.Meta.Annotations,
			},
		},
		Spec: DeviceSpec{
			DeviceId: d.Spec.DeviceId,
			Disabled: &d.Spec.Disabled,
		},
		Status: DeviceStatus{},
	}
}

func ProtoDeviceTargetToMirDeviceTarget(t *mir_apiv1.DeviceTarget) DeviceTarget {
	if t == nil {
		return DeviceTarget{}
	}
	return DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
	}
}

func MirDeviceTargetToProtoDeviceTarget(t DeviceTarget) *mir_apiv1.DeviceTarget {
	return &mir_apiv1.DeviceTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
		Ids:        t.Ids,
	}
}

/// Objects

func MirObjectTargetToProtoObjectTarget(t ObjectTarget) *mir_apiv1.Targets {
	return &mir_apiv1.Targets{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
	}
}

func ProtoObjectTargetToMirObjectTarget(t *mir_apiv1.Targets) ObjectTarget {
	if t == nil {
		return ObjectTarget{}
	}
	return ObjectTarget{
		Names:      t.Names,
		Namespaces: t.Namespaces,
		Labels:     t.Labels,
	}
}

func ProtoObjectToMirObject(o *mir_apiv1.Object) Object {
	if o == nil {
		return Object{}
	}
	return Object{
		ApiVersion: o.ApiVersion,
		Kind:       o.Kind,
		Meta: Meta{
			Name:        o.Meta.Name,
			Namespace:   o.Meta.Namespace,
			Labels:      o.Meta.Labels,
			Annotations: o.Meta.Annotations,
		},
	}
}

func MirObjectToProtoObject(o Object) *mir_apiv1.Object {
	return &mir_apiv1.Object{
		ApiVersion: o.ApiVersion,
		Kind:       o.Kind,
		Meta: &mir_apiv1.Meta{
			Name:        o.Meta.Name,
			Namespace:   o.Meta.Namespace,
			Labels:      o.Meta.Labels,
			Annotations: o.Meta.Annotations,
		},
	}
}

/// Events

func MirEventToProtoEvent(e Event) *mir_apiv1.Event {
	return &mir_apiv1.Event{
		Object: MirObjectToProtoObject(e.Object),
		Spec: &mir_apiv1.EventSpec{
			Type:          e.Spec.Type,
			Reason:        e.Spec.Reason,
			Message:       e.Spec.Message,
			JsonPayload:   e.Spec.Payload,
			RelatedObject: MirObjectToProtoObject(e.Spec.RelatedObject),
		},
		Status: &mir_apiv1.EventStatus{
			Count:   int32(e.Status.Count),
			FirstAt: AsProtoTimestampFromSurreal(&e.Status.FirstAt),
			LastAt:  AsProtoTimestampFromSurreal(&e.Status.LastAt),
		},
	}
}

func MirEventSpecToProtoCreateEvent(e EventSpec) *mir_apiv1.CreateEventRequest {
	return &mir_apiv1.CreateEventRequest{
		Spec: &mir_apiv1.EventSpec{
			Type:          e.Type,
			Reason:        e.Reason,
			Message:       e.Message,
			JsonPayload:   e.Payload,
			RelatedObject: MirObjectToProtoObject(e.RelatedObject),
		},
	}
}

func ProtoEventsToMirEvents(events []*mir_apiv1.Event) []Event {
	var mirEvents []Event
	for _, event := range events {
		mirEvents = append(mirEvents, ProtoEventToMirEvent(event))
	}
	return mirEvents
}

func MirEventsToProtoEvents(events []Event) []*mir_apiv1.Event {
	var protoEvents []*mir_apiv1.Event
	for _, event := range events {
		protoEvents = append(protoEvents, MirEventToProtoEvent(event))
	}
	return protoEvents
}

func ProtoEventToMirEvent(e *mir_apiv1.Event) Event {
	if e == nil {
		return Event{}
	}
	o := ProtoObjectToMirObject(e.Spec.RelatedObject)
	return Event{
		Object: Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "event",
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
			Payload:       e.Spec.JsonPayload,
			RelatedObject: o,
		},
		Status: EventStatus{
			Count:   int(e.Status.Count),
			FirstAt: AsSurrealTime(e.Status.FirstAt),
			LastAt:  AsSurrealTime(e.Status.LastAt),
		},
	}
}

func ProtoCreateEventReqToMirEventSpec(e *mir_apiv1.CreateEventRequest) EventSpec {
	if e == nil || e.Spec == nil {
		return EventSpec{}
	}
	o := ProtoObjectToMirObject(e.Spec.RelatedObject)
	return EventSpec{
		Type:          e.Spec.Type,
		Reason:        e.Spec.Reason,
		Message:       e.Spec.Message,
		Payload:       e.Spec.JsonPayload,
		RelatedObject: o,
	}
}

func ProtoEventTargetToMirEventTarget(t *mir_apiv1.EventTarget) EventTarget {
	if t == nil {
		return EventTarget{}
	}
	return EventTarget{
		ObjectTarget: ProtoObjectTargetToMirObjectTarget(t.Targets),
		DateFilter:   ProtoTimeFilterToMirTimeFilter(t.FilterDate),
		Limit:        int(t.FilterLimit),
	}
}

func MirEventTargetToProtoEventTarget(t EventTarget) *mir_apiv1.EventTarget {
	return &mir_apiv1.EventTarget{
		Targets:     MirObjectTargetToProtoObjectTarget(t.ObjectTarget),
		FilterDate:  MirTimeFilterToProtoTimeFilter(t.DateFilter),
		FilterLimit: int32(t.Limit),
	}
}

/// Utils

func asUnRefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func AsProtoTimestamp(t time.Time) *mir_apiv1.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &mir_apiv1.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsProtoTimestampFromSurreal(t *surrealdbModels.CustomDateTime) *mir_apiv1.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return &mir_apiv1.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsRefProtoTimestampFromSurreal(t *surrealdbModels.CustomDateTime) *mir_apiv1.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return &mir_apiv1.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsGoTime(ts *mir_apiv1.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}

func AsSurrealTime(ts *mir_apiv1.Timestamp) surrealdbModels.CustomDateTime {
	if ts == nil {
		return surrealdbModels.CustomDateTime{}
	}
	return surrealdbModels.CustomDateTime{Time: time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()}
}

func mapToProtoTs(m map[string]surrealdbModels.CustomDateTime) map[string]*mir_apiv1.Timestamp {
	ts := map[string]*mir_apiv1.Timestamp{}
	for k, v := range m {
		ts[k] = AsProtoTimestamp(v.Time)
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

func ProtoTimeFilterToMirTimeFilter(f *mir_apiv1.DateFilter) DateFilter {
	if f == nil {
		return DateFilter{}
	}
	return DateFilter{
		To:   AsGoTime(f.To),
		From: AsGoTime(f.From),
	}
}

func MirTimeFilterToProtoTimeFilter(f DateFilter) *mir_apiv1.DateFilter {
	return &mir_apiv1.DateFilter{
		To:   AsProtoTimestamp(f.To),
		From: AsProtoTimestamp(f.From),
	}
}
