package core_ito

import (
	"time"

	common_ito "github.com/maxthom/mir/internal/ito/proto/v1alpha/common_ito"
	"github.com/maxthom/mir/pkgs/models"
)

func NewUpdateDeviceMetaReqFromDevice(d models.Device) *UpdateDeviceRequest {
	toUpdateMap := func(m map[string]*string) map[string]*UpdateDeviceRequest_OptString {
		opt := map[string]*UpdateDeviceRequest_OptString{}
		for k, v := range m {
			if v == nil {
				opt[k] = &UpdateDeviceRequest_OptString{
					Value: nil,
				}
			} else {
				opt[k] = &UpdateDeviceRequest_OptString{
					Value: v,
				}
			}
		}
		return opt
	}

	// IDEA maybe add the possibility to update device id
	return &UpdateDeviceRequest{
		Meta: &UpdateDeviceRequest_Meta{
			Name:        &d.Meta.Name,
			Namespace:   &d.Meta.Namespace,
			Labels:      toUpdateMap(d.Meta.Labels),
			Annotations: toUpdateMap(d.Meta.Annotations),
		},
		Spec: &UpdateDeviceRequest_Spec{
			Disabled: &d.Spec.Disabled,
		},
		Targets: &Targets{
			Ids: []string{d.Spec.DeviceId},
		},
	}
}

func NewCreateDeviceReqFromDevice(d models.Device) *CreateDeviceRequest {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &CreateDeviceRequest{
		DeviceId:    d.Spec.DeviceId,
		Disabled:    d.Spec.Disabled,
		Name:        d.Meta.Name,
		Namespace:   d.Meta.Namespace,
		Labels:      toValueMap(d.Meta.Labels),
		Annotations: toValueMap(d.Meta.Annotations),
	}
}

func NewProtoDeviceListFromDevicesWithId(d []models.DeviceWithId) []*Device {
	p := []*Device{}
	for _, v := range d {
		p = append(p, NewProtoDeviceFromDeviceWithId(v))
	}
	return p
}

func NewProtoDeviceFromDeviceWithId(d models.DeviceWithId) *Device {
	toValueMap := func(m map[string]*string) map[string]string {
		vm := map[string]string{}
		for k, v := range m {
			vm[k] = *v
		}
		return vm
	}

	return &Device{
		Id:         d.Id,
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: &Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      toValueMap(d.Meta.Labels),
			Annotations: toValueMap(d.Meta.Annotations),
		},
		Spec: &Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Properties: &Properties{},
		Status: &Status{
			Online:         d.Status.Online,
			LastHearthbeat: AsProtoTimestamp(d.Status.LastHearthbeat),
		},
	}
}

func NewDeviceFromProtoDevice(d *Device) *models.Device {
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

	return &models.Device{
		ApiVersion: d.ApiVersion,
		ApiName:    d.ApiName,
		Meta: models.Meta{
			Name:        d.Meta.Name,
			Namespace:   d.Meta.Namespace,
			Labels:      toPtrMap(d.Meta.Labels),
			Annotations: toPtrMap(d.Meta.Annotations),
		},
		Spec: models.Spec{
			DeviceId: d.Spec.DeviceId,
			Disabled: d.Spec.Disabled,
		},
		Properties: models.Properties{},
		Status: models.Status{
			Online:         d.Status.Online,
			LastHearthbeat: lastHeartbeatTime,
		},
	}
}

func NewDeviceFromCreateDeviceReq(c *CreateDeviceRequest) models.Device {
	toPtrMap := func(m map[string]string) map[string]*string {
		mPtr := make(map[string]*string, len(m))
		for k, v := range m {
			mPtr[k] = &v
		}
		return mPtr
	}
	return models.Device{
		ApiVersion: "v1alpha",
		ApiName:    "twin",
		Meta: models.Meta{
			Name:        c.Name,
			Namespace:   c.Namespace,
			Labels:      toPtrMap(c.Labels),
			Annotations: toPtrMap(c.Annotations),
		},
		Spec: models.Spec{
			DeviceId: c.DeviceId,
			Disabled: c.Disabled,
		},
		Status: models.Status{},
	}
}

func AsProtoTimestamp(t time.Time) *common_ito.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &common_ito.Timestamp{
		Seconds: int64(t.Unix()),
		Nanos:   int32(t.Nanosecond()),
	}
}

func AsGoTime(ts *common_ito.Timestamp) time.Time {
	return time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC()
}
