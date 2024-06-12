package routes

import (
	"fmt"
	"strings"
)

type Subject string

func (s Subject) WithId(id string) string {
	return fmt.Sprintf(string(s), id)
}

func (s Subject) GetSource() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[0]
}

func (s Subject) GetId() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[1]
}

func (s Subject) GetModule() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[2]
}

func (s Subject) GetVersion() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[3]
}

func (s Subject) GetFunction() string {
	parts := strings.Split(string(s), ".")
	if len(parts) != 5 {
		return ""
	}
	return parts[4]
}

const (
	CreateDeviceStream     Subject = "client.%s.core.v1alpha.create"
	UpdateDeviceStream     Subject = "client.%s.core.v1alpha.update"
	DeleteDeviceStream     Subject = "client.%s.core.v1alpha.delete"
	ListDeviceStream       Subject = "client.%s.core.v1alpha.list"
	HearthbeatDeviceStream Subject = "device.%s.core.v1alpha.hearthbeat"
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
