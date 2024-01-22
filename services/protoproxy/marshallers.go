package protoproxy

import (
	"fmt"

	proto_lineprotocol "github.com/maxthom/mir/libs/proto/line_protocol"
	protostore "github.com/maxthom/mir/libs/proto/store"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Marhshallers struct {
	marshallers map[MarshallerKey]proto_lineprotocol.ProtoBytesToLpFn
	Registry    *protostore.Registry
}

type MarshallerKey struct {
	messageName string
	deviceId    string
}

var lr zerolog.Logger

func NewMarshallers(registry *protostore.Registry) *Marhshallers {
	return &Marhshallers{
		marshallers: map[MarshallerKey]proto_lineprotocol.ProtoBytesToLpFn{},
		Registry:    registry,
	}
}

// Deserialize from bytes to line protocol using a cache
// messageName represent the name of the package.message eg: swarm.Telemetry
func (m *Marhshallers) Deserialize(proto []byte, key MarshallerKey) (string, error) {
	if fn, ok := m.marshallers[key]; ok {
		return fn(proto, map[string]string{}), nil
	}

	// Create fn for cache from regitry
	desc, err := m.Registry.FindDescriptorByName(protoreflect.FullName(key.messageName))
	if err != nil {
		return "", fmt.Errorf("could not find descriptor with name %s", key.messageName)
	}

	// Create set of pinned tags
	pinnedTags := map[string]string{}
	if key.deviceId != "" {
		pinnedTags["deviceId"] = key.deviceId
	}

	fn, err := proto_lineprotocol.GenerateMarshalFn(pinnedTags, desc.(protoreflect.MessageDescriptor))
	if err != nil {
		l.Error().Err(err).Msg("error while loading descriptor")
	}
	m.marshallers[key] = fn

	return fn(proto, map[string]string{}), nil
}
