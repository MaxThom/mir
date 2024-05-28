package mir_module

import "strings"

type stream interface {
	Subject() string
}

// Builder
type streamBuilder struct {
	source  string
	id      string
	module  string
	fn      string
	version string
}

func Stream() streamBuilder {
	return streamBuilder{
		source:  "*",
		id:      "*",
		module:  "*",
		fn:      "*",
		version: "*",
	}
}

func (s streamBuilder) Subject() string {
	return strings.Join(
		[]string{
			s.source,
			s.id,
			s.module,
			s.fn,
			s.version,
		},
		".",
	)
}

func (s streamBuilder) Client() streamClientBuilder {
	s.source = "client"
	return streamClientBuilder{
		stream: s,
	}
}

func (s streamBuilder) Device() streamDeviceBuilder {
	s.source = "device"
	return streamDeviceBuilder{
		labels: map[string]string{},
		stream: s,
	}
}

// Device Builder
type streamDeviceBuilder struct {
	labels map[string]string
	stream streamBuilder
}

func (s streamDeviceBuilder) DeviceId(id string) streamDeviceBuilder {
	s.stream.id = id
	return s
}

func (s streamDeviceBuilder) Labels(lbl map[string]string) streamDeviceBuilder {
	s.labels = lbl
	return s
}

func (s streamDeviceBuilder) Core() coreDeviceStream {
	s.stream.module = "core"
	return coreDeviceStream{
		deviceStream: s,
	}
}

func (s streamDeviceBuilder) Subject() string {
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

// Client Builder
type streamClientBuilder struct {
	stream streamBuilder
}

func (s streamClientBuilder) UserId(id string) streamClientBuilder {
	s.stream.id = id
	return s
}

func (s streamClientBuilder) Core() coreClientStream {
	s.stream.module = "core"
	return coreClientStream{
		clientStream: s,
	}
}

func (s streamClientBuilder) Subject() string {
	return s.stream.Subject()
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

type event struct{}
