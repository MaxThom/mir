package proto_lineprotocol

import (
	"time"

	devicev1 "github.com/maxthom/mir/internal/libs/proto/line_protocol/proto_test/gen/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func findTimestampField(desc protoreflect.MessageDescriptor) {

}

func generateTimeFn(desc protoreflect.MessageDescriptor) (func(protoreflect.Message) int64, int) {
	tsFieldIndex := -1
	tsType := devicev1.TimestampType_TIMESTAMP_TYPE_UNSPECIFIED
	opts, ok := desc.Options().(*descriptorpb.MessageOptions)
	msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
	fieldsDesc := desc.Fields()
	if ok {
		if msgType == devicev1.MessageType_MESSAGE_TYPE_TELEMETRY {
			for i := 0; i < desc.Fields().Len(); i++ {
				fieldOpts, ok := desc.Fields().Get(i).Options().(*descriptorpb.FieldOptions)
				if ok {
					tsType, ok = proto.GetExtension(fieldOpts, devicev1.E_Timestamp).(devicev1.TimestampType)
					if ok && tsType != devicev1.TimestampType_TIMESTAMP_TYPE_UNSPECIFIED {
						tsFieldIndex = i
						break
					}
				}
			}
		}
	}

	if tsFieldIndex == -1 || tsType == devicev1.TimestampType_TIMESTAMP_TYPE_UNSPECIFIED {
		return func(_ protoreflect.Message) int64 {
			return time.Now().UTC().UnixNano()
		}, tsFieldIndex
	}
	switch tsType {
	case devicev1.TimestampType_TIMESTAMP_TYPE_SEC:
		return func(m protoreflect.Message) int64 {
			v := m.Get(fieldsDesc.Get(tsFieldIndex))
			return v.Int() * 1e9
		}, tsFieldIndex
	case devicev1.TimestampType_TIMESTAMP_TYPE_MILLI:
		return func(m protoreflect.Message) int64 {
			v := m.Get(fieldsDesc.Get(tsFieldIndex))
			return v.Int() * 1e6
		}, tsFieldIndex
	case devicev1.TimestampType_TIMESTAMP_TYPE_MICRO:
		return func(m protoreflect.Message) int64 {
			v := m.Get(fieldsDesc.Get(tsFieldIndex))
			return v.Int() * 1e3
		}, tsFieldIndex
	case devicev1.TimestampType_TIMESTAMP_TYPE_NANO:
		return func(m protoreflect.Message) int64 {
			v := m.Get(fieldsDesc.Get(tsFieldIndex))
			return v.Int()
		}, tsFieldIndex
	case devicev1.TimestampType_TIMESTAMP_TYPE_FRACTION:
		return func(m protoreflect.Message) int64 {
			v := m.Get(fieldsDesc.Get(tsFieldIndex))
			mrNested := v.Message()
			secondsField := mrNested.Descriptor().Fields().ByName("seconds")
			nanosField := mrNested.Descriptor().Fields().ByName("nanos")
			seconds := mrNested.Get(secondsField).Int()
			nanos := mrNested.Get(nanosField).Int()
			return seconds*1e9 + nanos
		}, tsFieldIndex
	default:
		return func(_ protoreflect.Message) int64 {
			return time.Now().UTC().UnixNano()
		}, tsFieldIndex
	}
}

func retrieveMessageTags(desc protoreflect.MessageDescriptor) map[string]string {
	tags := make(map[string]string)
	retrieveTagsFromDescriptor(desc, tags)
	return tags
}

func retrieveTagsFromDescriptor(desc protoreflect.MessageDescriptor, tags map[string]string) {
	// Get meta opts
	opts, ok := desc.Options().(*descriptorpb.MessageOptions)
	if ok {
		meta := proto.GetExtension(opts, devicev1.E_Meta).(*devicev1.Meta)
		if meta != nil {
			for k, v := range meta.Tags {
				if v != "" {
					tags[k] = v
				}
			}
		}
	}

	// Get tags of each field and walk through nested messages
	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)
		fieldOpts, ok := fd.Options().(*descriptorpb.FieldOptions)
		if ok {
			fieldMeta := proto.GetExtension(fieldOpts, devicev1.E_FieldMeta).(*devicev1.FieldMeta)
			if fieldMeta != nil {
				for k, v := range fieldMeta.Tags {
					if v != "" {
						tags[k] = v
					}
				}
			}
		}
		if fd.Kind() == protoreflect.MessageKind {
			retrieveTagsFromDescriptor(fd.Message(), tags)
		}
	}
}
