package routes

const (
	CreateDeviceStream     Subject = "client.%s.core.v1alpha.create"
	UpdateDeviceStream     Subject = "client.%s.core.v1alpha.update"
	DeleteDeviceStream     Subject = "client.%s.core.v1alpha.delete"
	ListDeviceStream       Subject = "client.%s.core.v1alpha.list"
	HearthbeatDeviceStream Subject = "device.%s.core.v1alpha.hearthbeat"
	DeviceOnlineEvent      Subject = "event.%s.core.v1alpha.deviceonline"
	DeviceOfflineEvent     Subject = "event.%s.core.v1alpha.deviceoffline"
	DeviceCreatedEvent     Subject = "event.%s.core.v1alpha.devicecreated"
	DeviceDeletedEvent     Subject = "event.%s.core.v1alpha.devicedeleted"
	DeviceUpdatedEvent     Subject = "event.%s.core.v1alpha.deviceupdated"
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
