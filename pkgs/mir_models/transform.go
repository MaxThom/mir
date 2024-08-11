package mir_models

import (
	"time"

	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/common_api"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
)

func NewUpdateDeviceMetaReqFromDevice(d Device) *core_api.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]*string) map[string]*common_api.OptString {
		opt := map[string]*common_api.OptString{}
		for k, v := range m {
			if v == nil {
				opt[k] = &common_api.OptString{
					Value: nil,
				}
			} else {
				opt[k] = &common_api.OptString{
					Value: v,
				}
			}
		}
		return opt
	}

	// IDEA maybe add the possibility to update device id
	return &core_api.UpdateDeviceRequest{
		Meta: &core_api.UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &core_api.UpdateDeviceRequest_Spec{
			Disabled: &d.Spec.Disabled,
		},
		Targets: &core_api.Targets{
			Ids: []string{d.Spec.DeviceId},
		},
	}
}

func NewCreateDeviceReqFromDevice(d Device) *core_api.CreateDeviceRequest {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core_api.CreateDeviceRequest{
		DeviceId:    d.Spec.DeviceId,
		Disabled:    d.Spec.Disabled,
		Name:        d.Meta.Name,
		Namespace:   d.Meta.Namespace,
		Labels:      toValueMap(d.Meta.Labels),
		Annotations: toValueMap(d.Meta.Annotations),
	}
}

func NewProtoDeviceListFromDevicesWithId(d []DeviceWithId) []*core_api.Device {
	p := []*core_api.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDeviceWithId(v))
	}
	return p
}

func NewProtoDeviceFromDeviceWithId(d DeviceWithId) *core_api.Device {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core_api.Device{
		Id:         d.Id,
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: &core_api.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      toValueMap(d.Meta.Labels),
			Annotations: toValueMap(d.Meta.Annotations),
		},
		Spec: &core_api.Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Properties: &core_api.Properties{},
		Status: &core_api.Status{
			Online:         d.Status.Online,
			LastHearthbeat: AsProtoTimestamp(d.Status.LastHearthbeat),
		},
	}
}

func NewDeviceFromProtoDevice(d *core_api.Device) *Device {
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
		},
	}
}

func NewDeviceFromCreateDeviceReq(c *core_api.CreateDeviceRequest) Device {
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

func AsProtoTimestamp(t time.Time) *common_api.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &common_api.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsGoTime(ts *common_api.Timestamp) time.Time {
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}
