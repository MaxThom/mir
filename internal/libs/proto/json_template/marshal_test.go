package jsontemplate

import (
	"fmt"
	"testing"

	json_template_testv1 "github.com/maxthom/mir/internal/libs/proto/json_template/proto_test/gen/json_template_test/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPrimitives(t *testing.T) {
	msg := json_template_testv1.Primitives{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	fmt.Println(string(tpl))
	fmt.Println(err)
}
