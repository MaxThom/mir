package mir_models

import (
	"time"

	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
)

func NewUpdateDeviceReqFromDevice(d Device) *core_apiv1.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]*string) map[string]*common_apiv1.OptString {
		opt := map[string]*common_apiv1.OptString{}
		for k, v := range m {
			if v == nil {
				opt[k] = &common_apiv1.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &common_apiv1.OptString{
					Value: v,
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
			Ids: []string{d.Spec.DeviceId},
		},
	}
}

func NewCreateDeviceReqFromDevice(d Device) *core_apiv1.CreateDeviceRequest {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core_apiv1.CreateDeviceRequest{
		DeviceId:    d.Spec.DeviceId,
		Disabled:    d.Spec.Disabled,
		Name:        d.Meta.Name,
		Namespace:   d.Meta.Namespace,
		Labels:      toValueMap(d.Meta.Labels),
		Annotations: toValueMap(d.Meta.Annotations),
	}
}

func NewProtoDeviceListFromDevicesWithId(d []DeviceWithId) []*core_apiv1.Device {
	p := []*core_apiv1.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDeviceWithId(v))
	}
	return p
}

func NewProtoDeviceFromDeviceWithId(d DeviceWithId) *core_apiv1.Device {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core_apiv1.Device{
		Id:         d.Id,
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: &core_apiv1.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      toValueMap(d.Meta.Labels),
			Annotations: toValueMap(d.Meta.Annotations),
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

func NewDeviceFromProtoDevice(d *core_apiv1.Device) *Device {
	toPtrMap := func(m map[string]string) map[string]*string {
		mPtr := make(map[string]*string, len(m))
		for k, v := range m {
			mPtr[k] = &v
		}
		return mPtr
	}

	var lastHeartbeatTime time.Time
	if d.Status.LastHearthbeat != nil {
		lastHeartbeatTime = AsGoTime(d.Status.LastHearthbeat)
	}
	var lastSchemaFetch time.Time
	if d.Status.Schema.LastSchemaFetch != nil {
		lastSchemaFetch = AsGoTime(d.Status.Schema.LastSchemaFetch)
	}

	return &Device{
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      toPtrMap(d.Meta.Labels),
			Annotations: toPtrMap(d.Meta.Annotations),
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

func NewDeviceFromCreateDeviceReq(c *core_apiv1.CreateDeviceRequest) Device {
	toPtrMap := func(m map[string]string) map[string]*string {
		mPtr := make(map[string]*string, len(m))
		for k, v := range m {
			mPtr[k] = &v
		}
		return mPtr
	}
	return Device{
		ApiVersion: "v1alpha",
		ApiName:    "twin",
		Meta: Meta{
			Name:        c.Name,
			Namespace:   c.Namespace,
			Labels:      toPtrMap(c.Labels),
			Annotations: toPtrMap(c.Annotations),
		},
		Spec: Spec{
			DeviceId: c.DeviceId,
			Disabled: c.Disabled,
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
