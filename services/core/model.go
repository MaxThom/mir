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
	DeviceId string `json:"device_id"`
	Spec     Spec   `json:"spec"`
	Status   Status `json:"status"`
}

type Spec struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Disabled    bool               `json:"disabled"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type Status struct {
	Online         bool      `json:"online"`
	LastHearthbeat time.Time `json:"last_hearthbeat"`
}

func NewUpdateDeviceSpecReqFromDevice(d Device) *core.UpdateDeviceRequest {
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
		Request: &core.UpdateDeviceRequest_Spec_{
			Spec: &core.UpdateDeviceRequest_Spec{
				Name:        &d.Spec.Name,
				Description: &d.Spec.Description,
				Disabled:    &d.Spec.Disabled,
				Labels:      toUpdateMap(d.Spec.Labels),
				Annotations: toUpdateMap(d.Spec.Annotations),
			},
		},
		Targets: &core.Targets{
			Ids: []string{d.DeviceId},
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
		DeviceId:    d.DeviceId,
		Name:        d.Spec.Name,
		Description: d.Spec.Description,
		Disabled:    d.Spec.Disabled,
		Labels:      toValueMap(d.Spec.Labels),
		Annotations: toValueMap(d.Spec.Annotations),
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
		Id:       d.Id,
		DeviceId: d.DeviceId,
		Spec: &core.Spec{
			Name:        d.Spec.Name,
			Description: d.Spec.Description,
			Disabled:    d.Spec.Disabled,
			Labels:      toValueMap(d.Spec.Labels),
			Annotations: toValueMap(d.Spec.Annotations),
		},
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
		DeviceId: d.DeviceId,
		Spec: Spec{
			Name:        d.Spec.Name,
			Description: d.Spec.Description,
			Disabled:    d.Spec.Disabled,
			Labels:      toPtrMap(d.Spec.Labels),
			Annotations: toPtrMap(d.Spec.Annotations),
		},
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
		DeviceId: c.DeviceId,
		Spec: Spec{
			Name:        c.Name,
			Description: c.Description,
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
