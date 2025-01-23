package mir_models

import (
	"time"

	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
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
		dev.Spec = Spec{
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
		dev.Status = Status{
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
		},
	}
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
		ApiVersion: "v1alpha",
		ApiName:    "device",
		Meta: Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      d.Meta.Labels,
			Annotations: d.Meta.Annotations,
		},
		Spec: Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Status: Status{},
	}
}

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
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}
