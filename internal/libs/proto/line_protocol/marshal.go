package proto_lineprotocol

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/proto/proto_mir"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type (
	ProtoBytesToLpFn      = func(in []byte, tags map[string]string) string
	WriteValueToLpFn      = func(value protoreflect.Value, mr protoreflect.Message, args []any) string
	WriteArrayValueToLpFn = func(value protoreflect.Value, args []any) string
)

const (
	nestedChar     string = "."
	arrayIndexChar string = "_"
)

// Syntax
// <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
// myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000
//
// measurement = proto name
// timestamp = time.Now or special field
// tag set = map of string string
// field set = map of string float|integer|uinteger|bool|string
// field of type string must be quoted
// fields are not indexed, while tags are indexed
// tags could be used for device id, schema id, etc
// tags could be passed in function or special field in schema

// Marshal from proto encoding to influx line protocol
// MessageDescriptor represent the proto descripor
// We create a proto.Message from it and the add the data to it from the in byte using the unmarshall
// Now with both data and descriptor, we can walk the fields
//
// TODO
// - [x] enum kind
// - [x] primitives fields
// - [x] return error and error join
// - [x] repeated, cardinality is repeated and IsList is true
// - [x] nested message
// - [x] repeated message
// - [x] map field
// - [x] optional field
// - [x] imports
// - [x] enum list
// - [x] fix number array mistmatch
// - [x] more unit test
// - [ ] one of field
// - [ ] enum options
// - [x] timestamp field
// - [x] labels field
func GenerateMarshalFn(pinnedTags map[string]string, desc protoreflect.MessageDescriptor) (ProtoBytesToLpFn, error) {
	var lpHead strings.Builder
	lpHead.WriteString(string(desc.FullName()))

	// Pinned tags
	// Get meta opts
	lbl := proto_mir.RetrieveMessageTags(desc)
	for k, v := range lbl {
		if v != "" {
			pinnedTags[k] = v
		}
	}

	// Good practices to sort them else its done by the db
	pinnedTagKeys := make([]string, 0, len(pinnedTags))
	for k := range pinnedTags {
		pinnedTagKeys = append(pinnedTagKeys, k)
	}
	sort.Strings(pinnedTagKeys)
	for _, k := range pinnedTagKeys {
		if pinnedTags[k] != "" {
			lpHead.WriteString(fmt.Sprintf(",%s=%s", k, pinnedTags[k]))
		}
	}

	// Timestamp special field
	skippedFields := []int{}
	timeFn, tsFieldIndex := generateTimeFn(desc)
	if tsFieldIndex != -1 {
		skippedFields = append(skippedFields, tsFieldIndex)
	}

	// Generate the field functions
	orderedFieldDesc, fieldFns, err := formatProtoMessageToLineProtocol("", desc, skippedFields)
	m := dynamicpb.NewMessage(desc)
	mr := m.ProtoReflect()

	return func(in []byte, tags map[string]string) string {
		var lp strings.Builder
		lp.WriteString(lpHead.String())
		// Tags
		// TODO assume tags are sorted, should be done at the source
		// TODO could be extra fields instead of tags
		for k, v := range tags {
			if v != "" {
				lp.WriteString(fmt.Sprintf(",%s=%s", k, v))
			}
		}

		// Put the data in the proto message
		if err := proto.Unmarshal(in, m); err != nil {
			// TODO update function to return error
			return ""
		}
		lp.WriteByte(' ')
		for i, fn := range fieldFns {
			// I wonder if I should pass the string builder around for performance
			lp.WriteString(fn(mr.Get(orderedFieldDesc[i]), mr, []any{}))
		}
		lpStr := strings.TrimSuffix(lp.String(), ",")

		// Timestamp
		lpStr += fmt.Sprint(" ", timeFn(mr))

		return lpStr
	}, err
}

func Marshal(in []byte, tags map[string]string, fn ProtoBytesToLpFn) string {
	return fn(in, tags)
}

func formatProtoMessageToLineProtocol(prefix string, desc protoreflect.MessageDescriptor, fieldIndexToSkip []int) ([]protoreflect.FieldDescriptor, []WriteValueToLpFn, error) {
	var errs error
	var err error

	orderedFieldDesc := make([]protoreflect.FieldDescriptor, desc.Fields().Len()-len(fieldIndexToSkip))
	skippedCounter := 0
	for i := 0; i < desc.Fields().Len(); i++ {
		if slices.Contains(fieldIndexToSkip, i) {
			skippedCounter += 1
			continue
		}
		fd := desc.Fields().Get(i)
		orderedFieldDesc[i-skippedCounter] = fd
	}

	// Create a set of function that will be called for each field
	fieldFns := make([]WriteValueToLpFn, len(orderedFieldDesc))
	for i := 0; i < len(orderedFieldDesc); i++ {
		fieldFns[i], err = formatProtoFieldToLineProtocol(prefix, orderedFieldDesc[i])
		errs = errors.Join(errs, err)
	}
	return orderedFieldDesc, fieldFns, errs
}

func formatProtoFieldToLineProtocol(prefix string, fd protoreflect.FieldDescriptor) (WriteValueToLpFn, error) {
	var errs error
	// Edge case with map, this allows to remove the 'value' field name
	fieldName := fd.TextName()
	if fd.Parent().(protoreflect.MessageDescriptor).IsMapEntry() {
		fieldName = ""
	}

	// Apply the right generation function for the field
	// according to its type
	switch fd.Cardinality() {
	case protoreflect.Required:
	case protoreflect.Optional:
		// Primitive types and nested messages
		switch fd.Kind() {
		case protoreflect.MessageKind:
			fds, fns, err := formatProtoMessageToLineProtocol(prefix+fieldName+nestedChar, fd.Message(), []int{})
			errs = errors.Join(errs, err)

			return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
				var lp strings.Builder
				// The value is a proto.Message thus we move in it
				mrNested := value.Message()
				// TODO does this need a hasPresence check and inner has
				// using this mr for has and fd.HasPresence outside
				for i, fn := range fns {
					lp.WriteString(fn(mrNested.Get(fds[i]), mrNested, args))
				}
				return lp.String()
			}, errs
		case protoreflect.GroupKind:
			return nil, fmt.Errorf("GroupKind for %q is a deprecated feature", fd.FullName())
		default:
			return formatProtoPrimitiveToLineProtocol(prefix+fieldName, fd)
		}

	case protoreflect.Repeated:
		// Complex objects such as list and map
		if fd.IsList() {
			switch fd.Kind() {
			case protoreflect.MessageKind:
				fds, fns, err := formatProtoMessageToLineProtocol(prefix+fieldName+arrayIndexChar+"%d"+nestedChar, fd.Message(), []int{})
				errs = errors.Join(errs, err)
				return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
					var lp strings.Builder
					// The value is a proto.List thus we move in it
					l := value.List()
					for i := 0; i < l.Len(); i++ {
						mrNested := l.Get(i).Message()
						for j, fn := range fns {
							lp.WriteString(fn(mrNested.Get(fds[j]), mrNested, append(args, i)))
						}
					}
					return lp.String()
				}, errs
			case protoreflect.GroupKind:
				return nil, fmt.Errorf("GroupKind for %q is a deprecated feature", fd.FullName())
			default:
				fn, err := formatProtoListToLineProtocol(prefix, fd)
				errs = errors.Join(errs, err)

				return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
					var lp strings.Builder
					l := value.List()
					for i := 0; i < l.Len(); i++ {
						lp.WriteString(fn(l.Get(i), append(args, i)))
					}
					return lp.String()
				}, errs

			}
		} else if fd.IsMap() {
			// Key can only be primitives except floating point and bytes
			keyPlaceholder := "%d"
			if fd.MapKey().Kind() == protoreflect.StringKind {
				keyPlaceholder = "%s"
			}
			fn, err := formatProtoFieldToLineProtocol(prefix+fieldName+arrayIndexChar+keyPlaceholder, fd.MapValue())
			errs = errors.Join(errs, err)
			return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
				// The value is a proto.List thus we move in it
				var lp strings.Builder
				value.Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					lp.WriteString(fn(value, mr, append(args, key.Interface())))
					return true
				})
				return lp.String()
			}, errs
		}
	}

	return nil, errs
}

func formatProtoPrimitiveToLineProtocol(prefix string, fd protoreflect.FieldDescriptor) (WriteValueToLpFn, error) {
	fieldLp := prefix + "="
	valueLp, err := formatProtoPrimitiveToSymbol(fd)
	fieldLp += valueLp

	if fd.HasPresence() {
		return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
			if mr.Has(fd) {
				return fmt.Sprintf(fieldLp, append(args, value.Interface())...) + ","
			}
			return ""
		}, err
	}

	return func(value protoreflect.Value, mr protoreflect.Message, args []any) string {
		return fmt.Sprintf(fieldLp, append(args, value.Interface())...) + ","
	}, err
}

func formatProtoListToLineProtocol(prefix string, fd protoreflect.FieldDescriptor) (WriteArrayValueToLpFn, error) {
	fieldLp := prefix + fd.TextName() + arrayIndexChar + "%d" + "="
	valueLp, err := formatProtoPrimitiveToSymbol(fd)
	fieldLp += valueLp

	return func(value protoreflect.Value, args []any) string {
		return fmt.Sprintf(fieldLp, append(args, value.Interface())...) + ","
	}, err
}

func formatProtoPrimitiveToSymbol(fd protoreflect.FieldDescriptor) (string, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return "%t", nil
	case protoreflect.StringKind:
		return "%q", nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "%di", nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "%di", nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "%du", nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "%du", nil
	case protoreflect.FloatKind:
		return "%f", nil
	case protoreflect.DoubleKind:
		return "%f", nil
	case protoreflect.EnumKind:
		return "%du", nil
	// case protoreflect.BytesKind:
	// TODO array of byte, could be store as string base64
	default:
		return "%q", fmt.Errorf("unknown field kind %q for %q", fd.Kind(), fd.FullName())
	}
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
