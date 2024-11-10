package mir_models

import (
	"time"

	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
)

func NewDeviceListFromProtoDevices(d []*core_apiv1.Device) []*Device {
	p := []*Device{}
	for _, v := range d {
		dev := NewDeviceFromProtoDevice(v)
		p = append(p, &dev)
	}
	return p
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

	// IDEA maybe add the possibility to update device id
	return &core_apiv1.UpdateDeviceRequest{
		Meta: &core_apiv1.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &core_apiv1.UpdateDeviceRequest_Spec{
			Disabled: &d.Spec.Disabled,
		},
		Targets: &core_apiv1.Targets{
			Names:      []string{d.Meta.Name},
			Namespaces: []string{d.Meta.Namespace},
		},
	}
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

func NewProtoDeviceListFromDevices(d []Device) []*core_apiv1.Device {
	p := []*core_apiv1.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDevice(v))
	}
	return p
}

func NewProtoDeviceFromDevice(d Device) *core_apiv1.Device {
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
		Properties: &core_apiv1.Properties{},
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

func NewDeviceFromProtoDevice(d *core_apiv1.Device) Device {
	var lastHeartbeatTime time.Time
	if d.Status.LastHearthbeat != nil {
		lastHeartbeatTime = AsGoTime(d.Status.LastHearthbeat)
	}
	var lastSchemaFetch time.Time
	if d.Status.Schema.LastSchemaFetch != nil {
		lastSchemaFetch = AsGoTime(d.Status.Schema.LastSchemaFetch)
	}

	return Device{
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
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
		Properties: Properties{},
		Status: Status{
			Online:         d.Status.Online,
			LastHearthbeat: lastHeartbeatTime,
			Schema: Schema{
				CompressedSchema: d.Status.Schema.CompressedSchema,
				PackageNames:     d.Status.Schema.PackageNames,
				LastSchemaFetch:  lastSchemaFetch,
			},
		},
	}
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
