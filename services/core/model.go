package core

import (
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
)

type DeviceWithId struct {
	Id string `json:"id"`
	Device
}

type Device struct {
	ApiVersion string     `json:"apiVersion"`
	ApiName    string     `json:"apiName"`
	Meta       Meta       `json:"meta"`
	Properties Properties `json:"properties"`
	Status     Status     `json:"status"`
}

type Meta struct {
	DeviceId    string             `json:"deviceId"`
	Name        string             `json:"name"`
	Disabled    bool               `json:"disabled"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type Properties struct {
}

type Status struct {
	Online         bool      `json:"online"`
	LastHearthbeat time.Time `json:"lastHearthbeat"`
}

func NewUpdateDeviceMetaReqFromDevice(d Device) *core.UpdateDeviceRequest {
	toUpdateMap := func(m map[string]*string) map[string]*core.UpdateDeviceRequest_OptString {
		opt := map[string]*core.UpdateDeviceRequest_OptString{}
		for k, v := range m {
			if v == nil {
				opt[k] = &core.UpdateDeviceRequest_OptString{
					Value: nil,
				}
			} else {
				opt[k] = &core.UpdateDeviceRequest_OptString{
					Value: v,
				}
			}
		}
		return opt
	}

	// IDEA maybe add the possibility to update device id
	return &core.UpdateDeviceRequest{
		Request: &core.UpdateDeviceRequest_Meta_{
			Meta: &core.UpdateDeviceRequest_Meta{
				Name:        &d.Meta.Name,
				Disabled:    &d.Meta.Disabled,
				Labels:      toUpdateMap(d.Meta.Labels),
				Annotations: toUpdateMap(d.Meta.Annotations),
			},
		},
		Targets: &core.Targets{
			Ids: []string{d.Meta.DeviceId},
		},
	}
}

func NewCreateDeviceReqFromDevice(d Device) *core.CreateDeviceRequest {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core.CreateDeviceRequest{
		DeviceId:    d.Meta.DeviceId,
		Name:        d.Meta.Name,
		Disabled:    d.Meta.Disabled,
		Labels:      toValueMap(d.Meta.Labels),
		Annotations: toValueMap(d.Meta.Annotations),
	}
}

func NewProtoDeviceListFromDevicesWithId(d []*DeviceWithId) []*core.Device {
	p := []*core.Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDeviceWithId(v))
	}
	return p
}

func NewProtoDeviceFromDeviceWithId(d *DeviceWithId) *core.Device {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &core.Device{
		Id:         d.Id,
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: &core.Meta{
			DeviceId:    d.Meta.DeviceId,
			Name:        d.Meta.Name,
			Disabled:    d.Meta.Disabled,
			Labels:      toValueMap(d.Meta.Labels),
			Annotations: toValueMap(d.Meta.Annotations),
		},
		Properties: &core.Properties{},
		Status: &core.Status{
			Online:         d.Status.Online,
			LastHearthbeat: AsProtoTimestamp(d.Status.LastHearthbeat),
		},
	}
}

func NewDeviceFromProtoDevice(d *core.Device) *Device {
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
			DeviceId:    d.Meta.DeviceId,
			Name:        d.Meta.Name,
			Disabled:    d.Meta.Disabled,
			Labels:      toPtrMap(d.Meta.Labels),
			Annotations: toPtrMap(d.Meta.Annotations),
		},
		Properties: Properties{},
		Status: Status{
			Online:         d.Status.Online,
			LastHearthbeat: lastHeartbeatTime,
		},
	}
}

func NewDeviceFromCreateDeviceReq(c *core.CreateDeviceRequest) Device {
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
			DeviceId:    c.DeviceId,
			Name:        c.Name,
			Disabled:    c.Disabled,
			Labels:      toPtrMap(c.Labels),
			Annotations: toPtrMap(c.Annotations),
		},
		Status: Status{},
	}
}

func AsProtoTimestamp(t time.Time) *core.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &core.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsGoTime(ts *core.Timestamp) time.Time {
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}
