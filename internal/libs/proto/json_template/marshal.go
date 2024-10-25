package jsontemplate

import (
	"encoding/json"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func GenerateTemplate(desc protoreflect.MessageDescriptor) (json.RawMessage, error) {
	var errs error
	var sb strings.Builder
	sb.WriteString("{")
	sb.WriteString(string(desc.FullName()))
	sb.WriteString("}")
	return json.RawMessage(sb.String()), errs
}
