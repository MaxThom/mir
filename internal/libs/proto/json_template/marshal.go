package json_template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type Options struct {
	WithoutArrayExample bool
	WithoutMapExample   bool
}

func GenerateTemplate(desc protoreflect.MessageDescriptor, opts Options) (json.RawMessage, error) {
	var sb strings.Builder
	err := formatMessageToJson(&sb, desc, opts)
	return json.RawMessage(sb.String()), err
}

func GeneratePrettyTemplate(desc protoreflect.MessageDescriptor, opts Options) (json.RawMessage, error) {
	tpl, errs := GenerateTemplate(desc, opts)
	var prettyBuf bytes.Buffer
	errs = errors.Join(json.Indent(&prettyBuf, []byte(tpl), "", "  "))
	return json.RawMessage(prettyBuf.String()), errs
}

func formatMessageToJson(sb *strings.Builder, desc protoreflect.MessageDescriptor, opts Options) error {
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
		err := getJsonTemplateValue(sb, fd, opts)
		errs = errors.Join(err)
	}

	sb.WriteString("}")
	return errs
}

func getJsonTemplateValue(sb *strings.Builder, fd protoreflect.FieldDescriptor, opts Options) error {
	switch {
	case fd.IsList():
		// For array, we want to give one example of the type
		// so like an array of one element
		sb.WriteString("[")
		if !opts.WithoutArrayExample {
			err := getJsonTemplateValue_SingleField(sb, fd, opts)
			if err != nil {
				return err
			}
		}
		sb.WriteString("]")
		return nil
	case fd.IsMap():
		keyType := "\"string\":"
		// Map keys are integers or strings
		if fd.MapKey().Kind() != protoreflect.StringKind {
			keyType = "\"0\":"
		}
		sb.WriteString("{")
		if !opts.WithoutMapExample {
			sb.WriteString(keyType)
			err := getJsonTemplateValue_SingleField(sb, fd.MapValue(), opts)
			if err != nil {
				return err
			}
		}
		sb.WriteString("}")
		return nil

	default:
		return getJsonTemplateValue_SingleField(sb, fd, opts)
	}
}

func getJsonTemplateValue_SingleField(sb *strings.Builder, fd protoreflect.FieldDescriptor, opts Options) error {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		sb.WriteString("false")
	case protoreflect.StringKind:
		sb.WriteString("\"\"")
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
			sb.WriteString("\"")
			vals := []string{}
			for i := 0; i < fd.Enum().Values().Len(); i++ {
				vals = append(vals, string(fd.Enum().Values().Get(i).Name()))
			}
			sb.WriteString(strings.Join(vals, "|"))
			sb.WriteString("\"")
		} else {
			sb.WriteString("\"UNKNOWN\"")
		}
	case protoreflect.MessageKind:
		return formatMessageToJson(sb, fd.Message(), opts)
	case protoreflect.BytesKind:
		sb.WriteString("[")
		sb.WriteString("0, 255")
		sb.WriteString("]")
	default:
		return fmt.Errorf("unsupported field kind: %v", fd.Kind())
	}
	return nil
}
