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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/assert"
)

// Generate the marshal.pb file and codegen for unit testing
//go:generate protoc --go_out=tests/ --descriptor_set_out=./tests/gen/marshal.pb --include_imports ./tests/marshal.proto

var (
	//go:embed tests/gen/marshal.pb
	marshalProtoFile []byte
	protoRegistry    = &protoregistry.Files{}
)

func init() {
	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(marshalProtoFile, pbSet); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, f := range pbSet.GetFile() {
		fd, err := protodesc.NewFile(f, protoRegistry)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		protoRegistry.RegisterFile(fd)
	}
}

func TestPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.Primitives")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.Primitives{
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
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.Primitives a=23.200000,b=123.330002,c=23i,d=312i,e=12u,f=667u,g=231i,h=4234i,i=32u,j=1u,k=23i,l=23333i,m=true,n=\"hello old friend\""))
}

func TestEnums(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.EnumsSimple")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.EnumsSimple{
		A: gen.EnumABC_A,
		B: gen.EnumABC_B,
		C: gen.EnumABC_C,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
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

	testCase := &gen.RepeatedPrimitives{
		A: []float64{0.5, 1.5, 2.5, 3.5},
		B: []uint32{0, 1, 2, 3},
		C: []int32{-2, -1, 0, 1, 2},
		D: []string{"abc", "def", "ghi"},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
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

	testCase := &gen.OneLevelNesting{
		A: &gen.SmallSetPrimitives{
			A: 2,
			B: 3,
			C: 4,
			D: "5",
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
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

	testCase := &gen.TwoLevelNesting{
		A: &gen.OneLevelNesting{
			A: &gen.SmallSetPrimitives{
				A: 2,
				B: 3,
				C: 4,
				D: "5",
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.TwoLevelNesting a.a.a=2.000000,a.a.b=3u,a.a.c=4i,a.a.d=\"5\""))
}

func TestOptionalTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalTwoLevelNesting{
		A: &gen.OptionalOneLevelNesting{
			A: &gen.OptionalSmallSetPrimitives{
				A: nil,
				B: ptr[uint32](4),
				C: ptr[int32](4),
				D: nil,
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalTwoLevelNesting a.a.b=4u,a.a.c=4i"))
}

func TestOptionalTwoLevelNestingNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalTwoLevelNesting{
		A: &gen.OptionalOneLevelNesting{
			A: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalTwoLevelNesting "))
}

func TestOptionalTwoLevelNestingNilNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalTwoLevelNesting{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalTwoLevelNesting "))
}

func TestRepeatedOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.RepeatedOneLevelNesting{
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
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
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

	testCase := &gen.RepeatedTwoLevelNesting{
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
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
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

	testCase := &gen.MapString{
		A: map[string]string{
			"hello":   "world",
			"goodbye": "world",
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert

	assert.Equal(t, true, strings.Contains(lp, "marshal.MapString"))
	assert.Equal(t, true, strings.Contains(lp, "a_hello=\"world\""))
	assert.Equal(t, true, strings.Contains(lp, "a_goodbye=\"world\""))
}

func TestMapMessage(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MapMessage")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.MapMessage{
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
			7: {
				A: 222,
				B: 333,
				C: 444,
				D: "555",
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MapMessage"))
	assert.Equal(t, true, strings.Contains(lp, "a_3.a=2.000000,a_3.b=3u,a_3.c=4i,a_3.d=\"5\""))
	assert.Equal(t, true, strings.Contains(lp, "a_5.a=22.000000,a_5.b=33u,a_5.c=44i,a_5.d=\"55\""))
	assert.Equal(t, true, strings.Contains(lp, "a_7.a=222.000000,a_7.b=333u,a_7.c=444i,a_7.d=\"555\""))
}

func TestOptionalSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalSmallSetPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalSmallSetPrimitives{
		A: nil,
		B: ptr[uint32](3),
		C: nil,
		D: ptr("5"),
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalSmallSetPrimitives b=3u,d=\"5\""))
}

func TestOptionalOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalOneLevelNesting{
		A: &gen.OptionalSmallSetPrimitives{
			A: nil,
			B: ptr[uint32](3),
			C: ptr[int32](4),
			D: ptr("5"),
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalOneLevelNesting a.b=3u,a.c=4i,a.d=\"5\""))
}

func TestOptionalOneLevelNestingNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalOneLevelNesting{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalOneLevelNesting"))
}

func TestOptionalEnumNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalEnum{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalEnum"))
}

func TestOptionalEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalEnum{
		A: &gen.OptionalEnumsSimple{
			A: nil,
			B: ptr(gen.EnumABC_B),
			C: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalEnum a.b=1u"))
}

func TestOptionalNestedEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalNestedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalNestedEnum{
		A: &gen.OptionalEnumsSimple{
			A: nil,
			B: ptr(gen.EnumABC_B),
			C: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalNestedEnum a.b=1u"))
}

func TestOptionalNestedEnumNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalNestedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalNestedEnum{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalNestedEnum"))
}

func TestRepeatedEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.RepeatedEnum{
		A: []*gen.EnumsSimple{
			{},
			{
				A: gen.EnumABC_A,
				B: gen.EnumABC_B,
				C: gen.EnumABC_C,
			},
			{
				A: gen.EnumABC_A,
				B: gen.EnumABC_B,
				C: gen.EnumABC_C,
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedEnum a_0.a=0u,a_0.b=0u,a_0.c=0u,a_1.a=0u,a_1.b=1u,a_1.c=2u,a_2.a=0u,a_2.b=1u,a_2.c=2u"))
}

func TestRepeatedOptionalEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedOptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.RepeatedOptionalEnum{
		A: []*gen.OptionalEnumsSimple{
			{
				C: nil,
			},
			{
				A: ptr(gen.EnumABC_A),
				B: nil,
				C: ptr(gen.EnumABC_C),
			},
			{
				A: ptr(gen.EnumABC_A),
				B: nil,
				C: nil,
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedOptionalEnum a_1.a=0u,a_1.c=2u,a_2.a=0u"))
}

func TestRepeatedOptionalSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedOptionalSmallSetPrimitive")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.RepeatedOptionalSmallSetPrimitive{
		A: []*gen.OptionalSmallSetPrimitives{
			{
				A: nil,
				B: nil,
				C: nil,
				D: nil,
			},
			{
				A: ptr(23.43),
				B: nil,
				C: ptr[int32](32),
				D: nil,
			},
			nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedOptionalSmallSetPrimitive a_1.a=23.430000,a_1.c=32i"))
}

func TestThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.ThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.ThreeLevelNestingAndSomeChads{
		A: &gen.OneLevelNesting{
			A: &gen.SmallSetPrimitives{
				A: 1,
				B: 2,
				C: 3,
				D: "d",
			},
		},
		B: 2,
		C: "abc",
		D: 43.2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.ThreeLevelNestingAndSomeChads a.a.a=1.000000,a.a.b=2u,a.a.c=3i,a.a.d=\"d\",b=2i,c=\"abc\",d=43.200000"))
}

func TestRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.RepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.RepeatedThreeLevelNestingAndSomeChads{
		A: []*gen.OneLevelNesting{
			{
				A: &gen.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
			{
				A: &gen.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
		},
		B: 2,
		C: "abc",
		D: 43.2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.RepeatedThreeLevelNestingAndSomeChads a_0.a.a=1.000000,a_0.a.b=2u,a_0.a.c=3i,a_0.a.d=\"d\",a_1.a.a=1.000000,a_1.a.b=2u,a_1.a.c=3i,a_1.a.d=\"d\",b=2i,c=\"abc\",d=43.200000"))
}

func TestMultipleThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MultipleThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.MultipleThreeLevelNestingAndSomeChads{
		A: &gen.OneLevelNesting{
			A: &gen.SmallSetPrimitives{
				A: 1,
				B: 2,
				C: 3,
				D: "d",
			},
		},
		B: &gen.TwoLevelNesting{
			A: &gen.OneLevelNesting{
				A: &gen.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
		},
		C: 3,
		D: "43.2",
		E: 23.2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MultipleThreeLevelNestingAndSomeChads a.a.a=1.000000,a.a.b=2u,a.a.c=3i,a.a.d=\"d\",b.a.a.a=1.000000,b.a.a.b=2u,b.a.a.c=3i,b.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestMultipleRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.MultipleRepeatedThreeLevelNestingAndSomeChads{
		A: []*gen.OneLevelNesting{
			{
				A: &gen.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
			{
				A: &gen.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
		},
		B: []*gen.TwoLevelNesting{
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
		},
		C: 3,
		D: "43.2",
		E: 23.2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MultipleRepeatedThreeLevelNestingAndSomeChads a_0.a.a=1.000000,a_0.a.b=2u,a_0.a.c=3i,a_0.a.d=\"d\",a_1.a.a=1.000000,a_1.a.b=2u,a_1.a.c=3i,a_1.a.d=\"d\",b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestMultipleRepeatedThreeLevelNestingAndSomeChadsEmpty(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.MultipleRepeatedThreeLevelNestingAndSomeChads{
		A: []*gen.OneLevelNesting{},
		B: []*gen.TwoLevelNesting{
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
		},
		C: 3,
		D: "43.2",
		E: 23.2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MultipleRepeatedThreeLevelNestingAndSomeChads b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestOptionalMultipleRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads{
		A: &gen.OptionalOneLevelNesting{
			A: &gen.OptionalSmallSetPrimitives{
				A: nil,
				B: ptr[uint32](3),
				C: nil,
				D: nil,
			},
		},
		B: []*gen.TwoLevelNesting{
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &gen.OneLevelNesting{
					A: &gen.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
		},
		C: 3,
		D: "43.2",
		E: ptr[float64](23.2),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads a.a.b=3u,b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestOptionalMultipleRepeatedThreeLevelNestingAndSomeChadsEmpty(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &gen.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads{
		A: &gen.OptionalOneLevelNesting{
			A: nil,
		},
		B: []*gen.TwoLevelNesting{},
		C: 3,
		D: "43.2",
		E: nil,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads c=3i,d=\"43.2\""))
}

func TestImportMessage(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.ImportMessage")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &gen.ImportMessage{
		A: &timestamppb.Timestamp{
			Seconds: 123,
			Nanos:   256,
		},
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.ImportMessage a.seconds=123i,a.nanos=256i"))
}

func TestMixIndexSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MixIndexSmallSetPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &gen.MixIndexSmallSetPrimitives{
		A: 1,
		B: 2,
		C: 3,
		D: "d",
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MixIndexSmallSetPrimitives a=1.000000,b=2u,c=3i,d=\"d\""))
}

func TestMixIndexOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("marshal.MixIndexOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &gen.MixIndexOneLevelNesting{
		A: &gen.MixIndexSmallSetPrimitives{
			A: 1,
			B: 2,
			C: 3,
			D: "d",
		},
		B: 2,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marhsal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "marshal.MixIndexOneLevelNesting a.a=1.000000,a.b=2u,a.c=3i,a.d=\"d\",b=2u,c=0i,d=\"\""))
}

func ptr[T any](t T) *T {
	return &t
}
