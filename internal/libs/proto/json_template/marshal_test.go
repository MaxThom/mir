package json_template

import (
	"testing"

	json_template_testv1 "github.com/maxthom/mir/internal/libs/proto/json_template/proto_test/gen/json_template_test/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gotest.tools/assert"
)

func TestPrimitives(t *testing.T) {
	msg := json_template_testv1.Primitives{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":0.0,"b":0.0,"c":0,"d":0,"e":0,"f":0,"g":0,"h":0,"i":0,"j":0,"k":0,"l":0,"m":false,"n":""}`)
}

func TestEnumsSimple(t *testing.T) {
	msg := json_template_testv1.EnumsSimple{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","b":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","c":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C"}`)
}

func TestEnumsSimpleOptional(t *testing.T) {
	msg := json_template_testv1.OptionalEnumsSimple{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","b":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","c":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C"}`)
}

func TestRepeatedEnum(t *testing.T) {
	msg := json_template_testv1.RepeatedEnum{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[{"a":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","b":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C","c":"ENUM_ABC_UNSPECIFIED|ENUM_ABC_A|ENUM_ABC_B|ENUM_ABC_C"}]}`)
}

func TestRepeatedPrimitives(t *testing.T) {
	msg := json_template_testv1.RepeatedPrimitives{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[0.0],"b":[0],"c":[0],"d":[""]}`)
}

func TestOneLevelNesting(t *testing.T) {
	msg := json_template_testv1.OneLevelNesting{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"a":0.0,"b":0,"c":0,"d":""}}`)
}

func TestMixIndexOneLevelNesting(t *testing.T) {
	msg := json_template_testv1.MixIndexOneLevelNesting{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"a":0.0,"b":0,"c":0,"d":""},"b":0,"c":0,"d":""}`)
}

func TestRepeatedOneLevelNesting(t *testing.T) {
	msg := json_template_testv1.RepeatedOneLevelNesting{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[{"a":0.0,"b":0,"c":0,"d":""}]}`)
}

func TestThreeLevelNesting(t *testing.T) {
	msg := json_template_testv1.RepeatedThreeLevelNestingAndSomeChads{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[{"a":{"a":0.0,"b":0,"c":0,"d":""}}],"b":0,"c":"","d":0.0}`)
}

func TestRepeatedThreeLevelNesting(t *testing.T) {
	msg := json_template_testv1.RepeatedThreeLevelNestingAndSomeChads{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[{"a":{"a":0.0,"b":0,"c":0,"d":""}}],"b":0,"c":"","d":0.0}`)
}

func TestMultipleThreeLevelNesting(t *testing.T) {
	msg := json_template_testv1.MultipleThreeLevelNestingAndSomeChads{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"a":{"a":0.0,"b":0,"c":0,"d":""}},"b":{"a":{"a":{"a":0.0,"b":0,"c":0,"d":""}}},"c":0,"d":"","e":0.0}`)
}

func TestRepeatedMultipleThreeLevelNesting(t *testing.T) {
	msg := json_template_testv1.MultipleRepeatedThreeLevelNestingAndSomeChads{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[{"a":{"a":0.0,"b":0,"c":0,"d":""}}],"b":[{"a":{"a":{"a":0.0,"b":0,"c":0,"d":""}}}],"c":0,"d":"","e":0.0}`)
}

func TestOptionalRepeatedMultipleThreeLevelNesting(t *testing.T) {
	msg := json_template_testv1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"a":{"a":0.0,"b":0,"c":0,"d":""}},"b":[{"a":{"a":{"a":0.0,"b":0,"c":0,"d":""}}}],"c":0,"d":"","e":0.0}`)
}

func TestMapString(t *testing.T) {
	msg := json_template_testv1.MapString{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"string":""}}`)
}

func TestMapMessage(t *testing.T) {
	msg := json_template_testv1.MapMessage{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":{"0":{"a":0.0,"b":0,"c":0,"d":""}}}`)
}

func TestBasicBytes(t *testing.T) {
	msg := json_template_testv1.BasicBytes{}
	desc := msg.ProtoReflect().Descriptor()

	tpl, err := GenerateTemplate(desc.(protoreflect.MessageDescriptor))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(tpl), `{"a":[0, 255]}`)
}
