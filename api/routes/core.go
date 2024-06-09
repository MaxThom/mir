package routes

import "fmt"

type Subject string

func (s Subject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

const (
	CreateDeviceStream     Subject = "client.%s.core.create.v1alpha"
	UpdateDeviceStream     Subject = "client.%s.core.update.v1alpha"
	DeleteDeviceStream     Subject = "client.%s.core.delete.v1alpha"
	ListDeviceStream       Subject = "client.%s.core.list.v1alpha"
	HearthbeatDeviceStream Subject = "device.%s.core.hearthbeat.v1alpha"
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
