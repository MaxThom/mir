package core

import (
	"time"

	"github.com/maxthom/mir/api/gen/proto/v1alpha/core"
)

type Device struct {
	Id          string             `json:"__id"`
	DeviceId    string             `json:"__device_id"`
	System      system             `json:"__system"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Disabled    bool               `json:"disabled"`
	Labels      map[string]*string `json:"labels"`
	Annotations map[string]*string `json:"annotations"`
}

type system struct {
	Online         bool      `json:"online"`
	LastHearthbeat time.Time `json:"last_hearth_beath"`
}

func NewUpdateDeviceReqFromDevice(d Device) *core.UpdateDeviceRequest {
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

	return &core.UpdateDeviceRequest{
		Name:        &d.Name,
		Description: &d.Description,
		Disabled:    &d.Disabled,
		Labels:      toUpdateMap(d.Labels),
		Annotations: toUpdateMap(d.Annotations),
		Targets: &core.Targets{
			Ids: []string{d.DeviceId},
		},
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

func NewDeviceFromProtoDevice(d *core.Device) *Device {
	toPtrMap := func(m map[string]string) map[string]*string {
		mPtr := make(map[string]*string, len(m))
		for k, v := range m {
			mPtr[k] = &v
		}
		return mPtr
	}

	var lastHeartbeatTime time.Time
	if d.LastHearthbeat != nil {
		lastHeartbeatTime = AsGoTime(d.LastHearthbeat)
	}

	return &Device{
		Id:          d.Id,
		DeviceId:    d.DeviceId,
		Name:        d.Name,
		Description: d.Description,
		Disabled:    d.Disabled,
		Labels:      toPtrMap(d.Labels),
		Annotations: toPtrMap(d.Annotations),
		System: system{
			Online:         d.Online,
			LastHearthbeat: lastHeartbeatTime,
		},
	}
}

func AsGoTime(ts *core.Timestamp) time.Time {
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}
