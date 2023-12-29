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

// Syntax
// <measurement>[,<tag_key>=<tag_value>[,<tag_key>=<tag_value>]] <field_key>=<field_value>[,<field_key>=<field_value>] [<timestamp>]
//myMeasurement,tag1=value1,tag2=value2 fieldKey="fieldValue" 1556813561098000000
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
func Marshal(pinnedTags map[string]string, desc protoreflect.MessageDescriptor) func([]byte, map[string]string) (string, error) {

	// Order the fields by the number
	// Have a special field name that is server for the timestamp
	// If not present, add current time
	// See if we have a virtual timestamp field that is always the time.now

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

	// TODO make sure the order is correct, might not be possible to determine before hand
	orderedFieldDesc := make([]protoreflect.FieldDescriptor, desc.Fields().Len())
	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)
		orderedFieldDesc[fd.Number()-1] = fd
	}

	// Create a set of function that will be called for each field
	fieldFns := make([]func(value protoreflect.Value) string, len(orderedFieldDesc))
	for i := 0; i < len(orderedFieldDesc); i++ {
		fd := orderedFieldDesc[i]
		fieldLp := ","
		if i == 0 {
			fieldLp = ""
		}
		fieldLp += string(fd.Name()) + "=" + formatProtoTypeToLineProtocol(fd.Kind())

		fieldFns[i] = func(value protoreflect.Value) string {
			return fmt.Sprintf(fieldLp, value.Interface())
		}
	}

	m := dynamicpb.NewMessage(desc)
	return func(in []byte, tags map[string]string) (string, error) {
		// Tags
		// TODO assume tags are sorted, should be done at the source
		for k, v := range tags {
			lp.WriteString(fmt.Sprintf(",%s=%s", k, v))
		}

		// Fields
		if err := proto.Unmarshal(in, m); err != nil {
			return "", err
		}
		lp.WriteByte(' ')
		for i, fn := range fieldFns {
			lp.WriteString(fn(m.ProtoReflect().Get(orderedFieldDesc[i])))
		}

		// Timestamp
		lp.WriteString(fmt.Sprint(" ", time.Now().UTC().Unix()))

		return lp.String(), nil
	}
}

func formatProtoTypeToLineProtocol(k protoreflect.Kind) string {
	switch k {
	case protoreflect.BoolKind:
		return "%t"
	case protoreflect.StringKind:
		return "%q"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "%d"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "%d"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "%d"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "%d"
	case protoreflect.FloatKind:
		return "%f"
	case protoreflect.DoubleKind:
		return "%f"
	default:
		return "%s"
	}

}
