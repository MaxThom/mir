package proto_lineprotocol

// TODO could used virtual FS from embed to load all proto file in directory

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/maxthom/mir/libs/proto/line_protocol/tests/gen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

// Generate the marshal.pb file and codegen for unit testing
//go:generate protoc --go_out=tests/ --descriptor_set_out=./tests/gen/marshal.pb ./tests/marshal.proto

//go:embed tests/gen/marshal.pb
var marshalProtoFile []byte
var protoRegistry = &protoregistry.Files{}

func init() {
	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(marshalProtoFile, pbSet); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// TODO do this better if multiple proto file
	fd, err := protodesc.NewFile(pbSet.GetFile()[0], protoRegistry)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	protoRegistry.RegisterFile(fd)
}

func TestPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.Primitives")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.Primitives{
		A: 23.2,
		B: 123.33,
		C: 23,
		D: 312,
		E: 12,
		F: 667,
		G: 231,
		H: 4234,
		I: 32,
		J: 1,
		K: 23,
		L: 23333,
		M: true,
		N: "hello old friend",
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{"pin": "yup", "aa": "bb"}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.Primitives,aa=bb,pin=yup a=23.200000,b=123.330002,c=23i,d=312i,e=12u,f=667u,g=231i,h=4234i,i=32u,j=1u,k=23i,l=23333i,m=true,n=\"hello old friend\""))
}

func TestEnums(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.EnumsSimple")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.EnumsSimple{
		A: gen.EnumABC_A,
		B: gen.EnumABC_B,
		C: gen.EnumABC_C,
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.EnumsSimple a=0u,b=1u,c=2u"))
}

func TestRepeatedPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.RepeatedPrimitives{
		A: []float64{0.5, 1.5, 2.5, 3.5},
		B: []uint32{0, 1, 2, 3},
		C: []int32{-2, -1, 0, 1, 2},
		D: []string{"abc", "def", "ghi"},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedPrimitives a_0=0.500000,a_1=1.500000,a_2=2.500000,a_3=3.500000,b_0=0u,b_1=1u,b_2=2u,b_3=3u,c_0=-2i,c_1=-1i,c_2=0i,c_3=1i,c_4=2i,d_0=\"abc\",d_1=\"def\",d_2=\"ghi\""))
}

func TestOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.OneLevelNesting{
		A: &gen.SmallSetPrimitives{
			A: 2,
			B: 3,
			C: 4,
			D: "5",
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OneLevelNesting a.a=2.000000,a.b=3u,a.c=4i,a.d=\"5\""))
}

func TestTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.TwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.TwoLevelNesting{
		A: &gen.OneLevelNesting{
			A: &gen.SmallSetPrimitives{
				A: 2,
				B: 3,
				C: 4,
				D: "5",
			},
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.TwoLevelNesting a.a.a=2.000000,a.a.b=3u,a.a.c=4i,a.a.d=\"5\""))
}

func TestRepeatedOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.RepeatedOneLevelNesting{
		A: []*gen.SmallSetPrimitives{
			{
				A: 2,
				B: 3,
				C: 4,
				D: "5",
			},
			{
				A: 22,
				B: 33,
				C: 44,
				D: "55",
			},
			{
				A: 222,
				B: 333,
				C: 444,
				D: "555",
			},
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedOneLevelNesting a_0.a=2.000000,a_0.b=3u,a_0.c=4i,a_0.d=\"5\",a_1.a=22.000000,a_1.b=33u,a_1.c=44i,a_1.d=\"55\",a_2.a=222.000000,a_2.b=333u,a_2.c=444i,a_2.d=\"555\""))
}

func TestRepeatedTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.RepeatedTwoLevelNesting{
		A: []*gen.RepeatedOneLevelNesting{
			{
				A: []*gen.SmallSetPrimitives{
					{
						A: 2,
						B: 3,
						C: 4,
						D: "5",
					},
					{
						A: 22,
						B: 33,
						C: 44,
						D: "55",
					},
				},
			},
			{
				A: []*gen.SmallSetPrimitives{
					{
						A: 222,
						B: 333,
						C: 444,
						D: "555",
					},
					{
						A: 2222,
						B: 3333,
						C: 4444,
						D: "5555",
					},
				},
			},
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedTwoLevelNesting a_0.a_0.a=2.000000,a_0.a_0.b=3u,a_0.a_0.c=4i,a_0.a_0.d=\"5\",a_0.a_1.a=22.000000,a_0.a_1.b=33u,a_0.a_1.c=44i,a_0.a_1.d=\"55\",a_1.a_0.a=222.000000,a_1.a_0.b=333u,a_1.a_0.c=444i,a_1.a_0.d=\"555\",a_1.a_1.a=2222.000000,a_1.a_1.b=3333u,a_1.a_1.c=4444i,a_1.a_1.d=\"5555\""))
}

func TestMapString(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MapString")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.MapString{
		A: map[string]string{
			"hello":   "world",
			"goodbye": "world",
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert

	assert.Equal(t, true, strings.Contains(lp, "marshal.MapString a_hello=\"world\",a_goodbye=\"world\""))
}

func TestMapMessage(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MapMessage")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.MapMessage{
		A: map[int32]*gen.SmallSetPrimitives{
			3: {
				A: 2,
				B: 3,
				C: 4,
				D: "5",
			},
			5: {
				A: 22,
				B: 33,
				C: 44,
				D: "55",
			},
		},
	}
	out, _ := proto.Marshal(todo)

	// Act
	lp, err := Marhsal(out, map[string]string{}, GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert

	//assert.Equal(t, true, strings.Contains(lp, "marshal.MapMessage a_5.a=22.000000,a_5.b=33u,a_5.c=44i,a_5.d=\"55\",a_5.a=22.000000,a_5.b=33u,a_5.c=44i,a_5.d=\"55\",a_7.a=222.000000,a_7.b=333u,a_7.c=444i,a_7.d=\"555\",a_5.a=22.000000,a_5.b=33u,a_5.c=44i,a_5.d=\"55\",a_7.a=222.000000,a_7.b=333u,a_7.c=444i,a_7.d=\"555\",a_3.a=2.000000,a_3.b=3u,a_3.c=4i,a_3.d=\"5\""))
}
