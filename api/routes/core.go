package routes

const (
	CreateDeviceRequest Subject = "client.%s.core.v1alpha.create"
	UpdateDeviceRequest Subject = "client.%s.core.v1alpha.update"
	DeleteDeviceRequest Subject = "client.%s.core.v1alpha.delete"
	ListDeviceRequest   Subject = "client.%s.core.v1alpha.list"

	DeviceOnlineEvent  Subject = "event.%s.core.v1alpha.deviceonline"
	DeviceOfflineEvent Subject = "event.%s.core.v1alpha.deviceoffline"
	DeviceCreatedEvent Subject = "event.%s.core.v1alpha.devicecreated"
	DeviceDeletedEvent Subject = "event.%s.core.v1alpha.devicedeleted"
	DeviceUpdatedEvent Subject = "event.%s.core.v1alpha.deviceupdated"

	HearthbeatDeviceStream Subject = "device.%s.core.v1alpha.hearthbeat"
	TelemetryDeviceStream  Subject = "device.%s.telemetry.v1alpha.proto"
)

// Core Builder
func (s streamClientBuilder) Core() coreClientStream {
	s.stream.module = "core"
	return coreClientStream{
		clientStream: s,
	}
}

func (s streamClientBuilder) Subject() string {
	return s.stream.Subject()
}

// Core Device Builder
type coreDeviceStream struct {
	deviceStream streamDeviceBuilder
}

func (s coreDeviceStream) V1Alpha() coreDeviceStream {
	s.deviceStream.stream.version = "v1alpha"
	return s
}

func (s coreDeviceStream) Hearthbeat() coreDeviceStream {
	s.deviceStream.stream.fn = "hearthbeat"
	return s
}

func (s coreDeviceStream) Subject() string {
	return s.deviceStream.Subject()
}

// Core Client Builder
type coreClientStream struct {
	clientStream streamClientBuilder
}

func (s coreClientStream) V1Alpha() coreClientStream {
	s.clientStream.stream.version = "v1alpha"
	return s
}

func (s coreClientStream) Create() coreClientStream {
	s.clientStream.stream.fn = "create"
	return s
}

func (s coreClientStream) Subject() string {
	return s.clientStream.Subject()
}
