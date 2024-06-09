package routes

import (
	"strings"
)

// Builder
type streamBuilder struct {
	source  string
	id      string
	module  string
	fn      string
	version string
}

func newStreamBuilder() streamBuilder {
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

func Client() streamClientBuilder {
	s := newStreamBuilder()
	s.source = "client"
	return streamClientBuilder{
		stream: s,
	}
}

func Device() streamDeviceBuilder {
	s := newStreamBuilder()
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

// Client Builder
type streamClientBuilder struct {
	stream streamBuilder
}

func (s streamClientBuilder) UserId(id string) streamClientBuilder {
	s.stream.id = id
	return s
}
