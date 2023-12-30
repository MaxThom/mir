package proto_lineprotocol

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoBytesToLineFn = func(in []byte, tags map[string]string) (string, error)
type WriteValueToLpFn = func(value protoreflect.Value) string

const (
	fieldArrayTpl string = "%s_%d=%s"
	nestingChar   string = "."
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

// Marhsal from proto encoding to influx line protocol
// MessageDescriptor represent the proto descripor
// We create a proto.Message from it and the add the data to it from the in byte using the unmarshall
// Now with both data and descriptor, we can walk the fields
//
// TODO
// - enum kind
// - group kind
// - repeated, cardinality is repeated and IsList is true
// - nested message
func GenerateMarshalFn(pinnedTags map[string]string, desc protoreflect.MessageDescriptor) ProtoBytesToLineFn {

	// Order the fields by the number
	// Have a special field name that is server for the timestamp
	// If not present, add current time

	var lp strings.Builder
	lp.WriteString(string(desc.FullName()))

	// Pinned tags
	// Good practices to sort them else its done by the db
	pinnedTagKeys := make([]string, 0, len(pinnedTags))
	for k := range pinnedTags {
		pinnedTagKeys = append(pinnedTagKeys, k)
	}
	sort.Strings(pinnedTagKeys)
	for _, k := range pinnedTagKeys {
		lp.WriteString(fmt.Sprintf(",%s=%s", k, pinnedTags[k]))
	}

	orderedFieldDesc, fieldFns := formatProtoMessageToLineProtocol("", desc)

	m := dynamicpb.NewMessage(desc)
	mr := m.ProtoReflect()
	return func(in []byte, tags map[string]string) (string, error) {
		// Tags
		// TODO assume tags are sorted, should be done at the source
		for k, v := range tags {
			lp.WriteString(fmt.Sprintf(",%s=%s", k, v))
		}

		// Put the data in the proto message
		if err := proto.Unmarshal(in, m); err != nil {
			return "", err
		}
		lp.WriteByte(' ')
		for i, fn := range fieldFns {
			// I wonder if I should pass the string builder around for performance
			lp.WriteString(fn(mr.Get(orderedFieldDesc[i])) + ",")
		}
		lpStr := strings.TrimSuffix(lp.String(), ",")

		// Timestamp
		//lp.WriteString(fmt.Sprint(" ", time.Now().UTC().UnixNano()))
		lpStr += fmt.Sprint(" ", time.Now().UTC().UnixNano())

		return lpStr, nil
	}
}

func Marhsal(in []byte, tags map[string]string, fn ProtoBytesToLineFn) (string, error) {
	return fn(in, tags)
}

func formatProtoMessageToLineProtocol(prefix string, desc protoreflect.MessageDescriptor) ([]protoreflect.FieldDescriptor, []WriteValueToLpFn) {
	// TODO make sure the order is correct, might not be possible to determine before hand
	// TODO better sort with Number
	orderedFieldDesc := make([]protoreflect.FieldDescriptor, desc.Fields().Len())
	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)
		orderedFieldDesc[fd.Number()-1] = fd
	}

	// Create a set of function that will be called for each field
	fieldFns := make([]WriteValueToLpFn, len(orderedFieldDesc))
	for i := 0; i < len(orderedFieldDesc); i++ {
		fd := orderedFieldDesc[i]

		// Apply the right generation function for the field
		// according to its type
		switch fd.Cardinality() {
		case protoreflect.Required:
		case protoreflect.Optional:
			// Primitive types and nested messages
			// Proto3 removed required
			// TODO
			// - if optional add the check in the func (if mr.Has(orderedFieldDesc[i]) { ... })
			// - bytes array
			// - map

			var lp strings.Builder
			switch fd.Kind() {
			case protoreflect.MessageKind:
				fds, fns := formatProtoMessageToLineProtocol(prefix+fd.TextName()+nestingChar, fd.Message())
				fieldFns[i] = func(value protoreflect.Value) string {
					// The value is a proto.Message thus we move in it
					mr := value.Message()
					for i, fn := range fns {
						lp.WriteString(fn(mr.Get(fds[i])) + ",")
					}
					return strings.TrimSuffix(lp.String(), ",")
				}
			case protoreflect.GroupKind:
				panic("GroupKind is a depraecated feature")
			default:
				fieldFns[i] = formatProtoPrimitiveToLineProtocol(prefix, fd)
			}

		case protoreflect.Repeated:
			// Complex objects such as list and map
			if fd.IsList() {
				fieldTpl := string(fd.Name()) + "_%d=" + formatProtoPrimitiveToSymbol(fd.Kind())
				fieldFns[i] = func(value protoreflect.Value) string {
					fieldStr := ""
					l := value.List()
					fieldStr += fmt.Sprintf(fieldTpl, i, l.Get(0).Interface())
					for i := 1; i < l.Len(); i++ {
						fieldStr += fmt.Sprintf(","+fieldTpl, i, l.Get(i).Interface())
					}
					return fieldStr
				}
				//} else if fd.IsMap() {
				// TODO after nesting https://protobuf.dev/programming-guides/encoding/#maps
				//}
			}
		}
	}
	return orderedFieldDesc, fieldFns
}

func formatProtoPrimitiveToLineProtocol(prefix string, fd protoreflect.FieldDescriptor) WriteValueToLpFn {
	fieldLp := prefix + fd.TextName() + "="

	switch fd.Kind() {
	case protoreflect.BoolKind:
		fieldLp += "%t"
	case protoreflect.StringKind:
		fieldLp += "%q"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		fieldLp += "%di"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		fieldLp += "%di"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		fieldLp += "%du"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		fieldLp += "%du"
	case protoreflect.FloatKind:
		fieldLp += "%f"
	case protoreflect.DoubleKind:
		fieldLp += "%f"
	case protoreflect.EnumKind:
		fieldLp += "%du"
	//case protoreflect.BytesKind:
	//TODO array of byte, could be store as string base64
	default:
		fieldLp += "%q"
	}

	return func(value protoreflect.Value) string {
		return fmt.Sprintf(fieldLp, value.Interface())
	}
}

func formatProtoPrimitiveToSymbol(k protoreflect.Kind) string {
	switch k {
	case protoreflect.BoolKind:
		return "%t"
	case protoreflect.StringKind:
		return "%q"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "%di"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "%di"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "%du"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "%du"
	case protoreflect.FloatKind:
		return "%f"
	case protoreflect.DoubleKind:
		return "%f"
	case protoreflect.EnumKind:
		return "%du"
	default:
		return "%q"
	}
}
