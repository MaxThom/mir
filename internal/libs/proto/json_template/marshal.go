package json_template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func GenerateTemplate(desc protoreflect.MessageDescriptor) (json.RawMessage, error) {
	var sb strings.Builder
	err := formatMessageToJson(&sb, desc)
	return json.RawMessage(sb.String()), err
}

func GeneratePrettyTemplate(desc protoreflect.MessageDescriptor) (json.RawMessage, error) {
	tpl, errs := GenerateTemplate(desc)
	var prettyBuf bytes.Buffer
	errs = errors.Join(json.Indent(&prettyBuf, []byte(tpl), "", "  "))
	return json.RawMessage(prettyBuf.String()), errs
}

func formatMessageToJson(sb *strings.Builder, desc protoreflect.MessageDescriptor) error {
	var errs error
	sb.WriteString("{")

	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if i > 0 {
			// We add comma after the first field so each addition
			// brings the previous comma
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%q:", fd.JSONName()))
		err := getJsonTemplateValue(sb, fd)
		errs = errors.Join(err)
	}

	sb.WriteString("}")
	return errs
}

func getJsonTemplateValue(sb *strings.Builder, fd protoreflect.FieldDescriptor) error {
	switch {
	case fd.IsList():
		// For array, we want to give one example of the type
		// so like an array of one element
		sb.WriteString("[")
		err := getJsonTemplateValue_SingleField(sb, fd)
		if err != nil {
			return err
		}
		sb.WriteString("]")
		return nil
	case fd.IsMap():
		keyType := "\"string\":"
		// Map keys are integers and strings
		if fd.MapKey().Kind() != protoreflect.StringKind {
			keyType = "\"0\":"
		}
		sb.WriteString("{")
		sb.WriteString(keyType)
		err := getJsonTemplateValue_SingleField(sb, fd.MapValue())
		if err != nil {
			return err
		}
		sb.WriteString("}")
		return nil

	default:
		return getJsonTemplateValue_SingleField(sb, fd)
	}
}

func getJsonTemplateValue_SingleField(sb *strings.Builder, fd protoreflect.FieldDescriptor) error {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		sb.WriteString("false")
	case protoreflect.StringKind:
		sb.WriteString("\"string\"")
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		sb.WriteString("0")
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		sb.WriteString("0.0")
	case protoreflect.EnumKind:
		// Use the first enum value as default
		if fd.Enum().Values().Len() > 0 {
			sb.WriteString(fmt.Sprintf("%q", fd.Enum().Values().Get(0).Name()))
		} else {
			sb.WriteString("\"UNKNOWN\"")
		}
	case protoreflect.MessageKind:
		return formatMessageToJson(sb, fd.Message())
	case protoreflect.BytesKind:
		// TODO check how to represent bytes
		sb.WriteString("0x12")
	default:
		return fmt.Errorf("unsupported field kind: %v", fd.Kind())
	}
	return nil
}
