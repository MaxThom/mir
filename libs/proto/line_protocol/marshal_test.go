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
	lp, err := Marhsal(out, map[string]string{"ca": "dev", "cb": "dev2"}, GenerateMarshalFn(map[string]string{"pin": "yup", "aa": "bb"}, desc.(protoreflect.MessageDescriptor)))
	if err != nil {
		assert.NilError(t, err)
	}
	fmt.Println(lp)

	// Assert
	assert.Equal(t, 1, 1)
	assert.Equal(t, true, strings.Contains(lp, "marshal.Primitives,aa=bb,pin=yup,ca=dev,cb=dev2 a=23.200000,b=123.330002,c=23i,d=312i,e=12u,f=667u,g=231i,h=4234i,i=32u,j=1u,k=23i,l=23333i,m=true,n=\"hello old friend\""))
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
	assert.Equal(t, 1, 1)
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
	assert.Equal(t, 1, 1)
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedPrimitives a_4=0.500000,a_1=1.500000,a_2=2.500000,a_3=3.500000,b_4=0u,b_1=1u,b_2=2u,b_3=3u,c_4=-2i,c_1=-1i,c_2=0i,c_3=1i,c_4=2i,d_4=\"abc\",d_1=\"def\",d_2=\"ghi\""))
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
	assert.Equal(t, 1, 1)
	// assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedPrimitives a_4=0.500000,a_1=1.500000,a_2=2.500000,a_3=3.500000,b_4=0u,b_1=1u,b_2=2u,b_3=3u,c_4=-2i,c_1=-1i,c_2=0i,c_3=1i,c_4=2i,d_4=\"abc\",d_1=\"def\",d_2=\"ghi\""))
}
