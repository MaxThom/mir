package proto_lineprotocol

// TODO could used virtual FS from embed to load all proto file in directory

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	lp_testv1 "github.com/maxthom/mir/internal/libs/proto/line_protocol/proto_test/gen/lp_test/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"gotest.tools/assert"
)

// Generate the lp_test.v1.pb file and codegen for unit testing
//go:generate protoc --go_out=proto_test/ --descriptor_set_out=./proto_test/gen/lp_test.v1.pb --include_imports ./proto_test/lp_test.v1.proto

var (
	//proto_test/gen/lp_test.v1.pb
	//go:embed proto_test/gen/lp.binpb
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

	// protoRegistry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
	// 	for i := 0; i < fd.Messages().Len(); i++ {
	// 		fmt.Println(fd.Messages().Get(i).FullName())
	// 	}
	// 	return true
	// })
}

func TestPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.Primitives")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.Primitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.Primitives a=23.200000,b=123.330002,c=23i,d=312i,e=12u,f=667u,g=231i,h=4234i,i=32u,j=1u,k=23i,l=23333i,m=true,n=\"hello old friend\""))
}

func TestEnums(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.EnumsSimple")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.EnumsSimple{
		A: lp_testv1.EnumABC_ENUM_ABC_A,
		B: lp_testv1.EnumABC_ENUM_ABC_B,
		C: lp_testv1.EnumABC_ENUM_ABC_C,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.EnumsSimple a=1u,b=2u,c=3u"))
}

func TestRepeatedPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedPrimitives{
		A: []float64{0.5, 1.5, 2.5, 3.5},
		B: []uint32{0, 1, 2, 3},
		C: []int32{-2, -1, 0, 1, 2},
		D: []string{"abc", "def", "ghi"},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedPrimitives a_0=0.500000,a_1=1.500000,a_2=2.500000,a_3=3.500000,b_0=0u,b_1=1u,b_2=2u,b_3=3u,c_0=-2i,c_1=-1i,c_2=0i,c_3=1i,c_4=2i,d_0=\"abc\",d_1=\"def\",d_2=\"ghi\""))
}

func TestOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OneLevelNesting{
		A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OneLevelNesting a.a=2.000000,a.b=3u,a.c=4i,a.d=\"5\""))
}

func TestTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.TwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.TwoLevelNesting{
		A: &lp_testv1.OneLevelNesting{
			A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.TwoLevelNesting a.a.a=2.000000,a.a.b=3u,a.a.c=4i,a.a.d=\"5\""))
}

func TestOptionalTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalTwoLevelNesting{
		A: &lp_testv1.OptionalOneLevelNesting{
			A: &lp_testv1.OptionalSmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalTwoLevelNesting a.a.b=4u,a.a.c=4i"))
}

func TestOptionalTwoLevelNestingNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalTwoLevelNesting{
		A: &lp_testv1.OptionalOneLevelNesting{
			A: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalTwoLevelNesting "))
}

func TestOptionalTwoLevelNestingNilNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalTwoLevelNesting{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalTwoLevelNesting "))
}

func TestRepeatedOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedOneLevelNesting{
		A: []*lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedOneLevelNesting a_0.a=2.000000,a_0.b=3u,a_0.c=4i,a_0.d=\"5\",a_1.a=22.000000,a_1.b=33u,a_1.c=44i,a_1.d=\"55\",a_2.a=222.000000,a_2.b=333u,a_2.c=444i,a_2.d=\"555\""))
}

func TestRepeatedTwoLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedTwoLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedTwoLevelNesting{
		A: []*lp_testv1.RepeatedOneLevelNesting{
			{
				A: []*lp_testv1.SmallSetPrimitives{
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
				A: []*lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedTwoLevelNesting a_0.a_0.a=2.000000,a_0.a_0.b=3u,a_0.a_0.c=4i,a_0.a_0.d=\"5\",a_0.a_1.a=22.000000,a_0.a_1.b=33u,a_0.a_1.c=44i,a_0.a_1.d=\"55\",a_1.a_0.a=222.000000,a_1.a_0.b=333u,a_1.a_0.c=444i,a_1.a_0.d=\"555\",a_1.a_1.a=2222.000000,a_1.a_1.b=3333u,a_1.a_1.c=4444i,a_1.a_1.d=\"5555\""))
}

func TestMapString(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MapString")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.MapString{
		A: map[string]string{
			"hello":   "world",
			"goodbye": "world",
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert

	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MapString"))
	assert.Equal(t, true, strings.Contains(lp, "a_hello=\"world\""))
	assert.Equal(t, true, strings.Contains(lp, "a_goodbye=\"world\""))
}

func TestMapMessage(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MapMessage")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.MapMessage{
		A: map[int32]*lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MapMessage"))
	assert.Equal(t, true, strings.Contains(lp, "a_3.a=2.000000,a_3.b=3u,a_3.c=4i,a_3.d=\"5\""))
	assert.Equal(t, true, strings.Contains(lp, "a_5.a=22.000000,a_5.b=33u,a_5.c=44i,a_5.d=\"55\""))
	assert.Equal(t, true, strings.Contains(lp, "a_7.a=222.000000,a_7.b=333u,a_7.c=444i,a_7.d=\"555\""))
}

func TestOptionalSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalSmallSetPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalSmallSetPrimitives{
		A: nil,
		B: ptr[uint32](3),
		C: nil,
		D: ptr("5"),
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalSmallSetPrimitives b=3u,d=\"5\""))
}

func TestOptionalOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalOneLevelNesting{
		A: &lp_testv1.OptionalSmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalOneLevelNesting a.b=3u,a.c=4i,a.d=\"5\""))
}

func TestOptionalOneLevelNestingNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalOneLevelNesting{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalOneLevelNesting"))
}

func TestOptionalEnumNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalEnum{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalEnum"))
}

func TestOptionalEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalEnum{
		A: &lp_testv1.OptionalEnumsSimple{
			A: nil,
			B: ptr(lp_testv1.EnumABC_ENUM_ABC_B),
			C: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalEnum a.b=2u"))
}

func TestOptionalNestedEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalNestedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalNestedEnum{
		A: &lp_testv1.OptionalEnumsSimple{
			A: nil,
			B: ptr(lp_testv1.EnumABC_ENUM_ABC_B),
			C: nil,
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalNestedEnum a.b=2u"))
}

func TestOptionalNestedEnumNil(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalNestedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalNestedEnum{
		A: nil,
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalNestedEnum"))
}

func TestRepeatedEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedEnum{
		A: []*lp_testv1.EnumsSimple{
			{},
			{
				A: lp_testv1.EnumABC_ENUM_ABC_A,
				B: lp_testv1.EnumABC_ENUM_ABC_B,
				C: lp_testv1.EnumABC_ENUM_ABC_C,
			},
			{
				A: lp_testv1.EnumABC_ENUM_ABC_A,
				B: lp_testv1.EnumABC_ENUM_ABC_B,
				C: lp_testv1.EnumABC_ENUM_ABC_C,
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedEnum a_0.a=0u,a_0.b=0u,a_0.c=0u,a_1.a=1u,a_1.b=2u,a_1.c=3u,a_2.a=1u,a_2.b=2u,a_2.c=3u"))
}

func TestRepeatedOptionalEnum(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedOptionalEnum")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedOptionalEnum{
		A: []*lp_testv1.OptionalEnumsSimple{
			{
				C: nil,
			},
			{
				A: ptr(lp_testv1.EnumABC_ENUM_ABC_A),
				B: nil,
				C: ptr(lp_testv1.EnumABC_ENUM_ABC_C),
			},
			{
				A: ptr(lp_testv1.EnumABC_ENUM_ABC_A),
				B: nil,
				C: nil,
			},
		},
	}
	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedOptionalEnum a_1.a=1u,a_1.c=3u,a_2.a=1u"))
}

func TestRepeatedOptionalSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedOptionalSmallSetPrimitive")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedOptionalSmallSetPrimitive{
		A: []*lp_testv1.OptionalSmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedOptionalSmallSetPrimitive a_1.a=23.430000,a_1.c=32i"))
}

func TestThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.ThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.ThreeLevelNestingAndSomeChads{
		A: &lp_testv1.OneLevelNesting{
			A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.ThreeLevelNestingAndSomeChads a.a.a=1.000000,a.a.b=2u,a.a.c=3i,a.a.d=\"d\",b=2i,c=\"abc\",d=43.200000"))
}

func TestRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.RepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.RepeatedThreeLevelNestingAndSomeChads{
		A: []*lp_testv1.OneLevelNesting{
			{
				A: &lp_testv1.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
			{
				A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.RepeatedThreeLevelNestingAndSomeChads a_0.a.a=1.000000,a_0.a.b=2u,a_0.a.c=3i,a_0.a.d=\"d\",a_1.a.a=1.000000,a_1.a.b=2u,a_1.a.c=3i,a_1.a.d=\"d\",b=2i,c=\"abc\",d=43.200000"))
}

func TestMultipleThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MultipleThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.MultipleThreeLevelNestingAndSomeChads{
		A: &lp_testv1.OneLevelNesting{
			A: &lp_testv1.SmallSetPrimitives{
				A: 1,
				B: 2,
				C: 3,
				D: "d",
			},
		},
		B: &lp_testv1.TwoLevelNesting{
			A: &lp_testv1.OneLevelNesting{
				A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MultipleThreeLevelNestingAndSomeChads a.a.a=1.000000,a.a.b=2u,a.a.c=3i,a.a.d=\"d\",b.a.a.a=1.000000,b.a.a.b=2u,b.a.a.c=3i,b.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestMultipleRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.MultipleRepeatedThreeLevelNestingAndSomeChads{
		A: []*lp_testv1.OneLevelNesting{
			{
				A: &lp_testv1.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
			{
				A: &lp_testv1.SmallSetPrimitives{
					A: 1,
					B: 2,
					C: 3,
					D: "d",
				},
			},
		},
		B: []*lp_testv1.TwoLevelNesting{
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MultipleRepeatedThreeLevelNestingAndSomeChads a_0.a.a=1.000000,a_0.a.b=2u,a_0.a.c=3i,a_0.a.d=\"d\",a_1.a.a=1.000000,a_1.a.b=2u,a_1.a.c=3i,a_1.a.d=\"d\",b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestMultipleRepeatedThreeLevelNestingAndSomeChadsEmpty(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.MultipleRepeatedThreeLevelNestingAndSomeChads{
		A: []*lp_testv1.OneLevelNesting{},
		B: []*lp_testv1.TwoLevelNesting{
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MultipleRepeatedThreeLevelNestingAndSomeChads b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestOptionalMultipleRepeatedThreeLevelNestingAndSomeChads(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads{
		A: &lp_testv1.OptionalOneLevelNesting{
			A: &lp_testv1.OptionalSmallSetPrimitives{
				A: nil,
				B: ptr[uint32](3),
				C: nil,
				D: nil,
			},
		},
		B: []*lp_testv1.TwoLevelNesting{
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
						A: 1,
						B: 2,
						C: 3,
						D: "d",
					},
				},
			},
			{
				A: &lp_testv1.OneLevelNesting{
					A: &lp_testv1.SmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads a.a.b=3u,b_0.a.a.a=1.000000,b_0.a.a.b=2u,b_0.a.a.c=3i,b_0.a.a.d=\"d\",b_1.a.a.a=1.000000,b_1.a.a.b=2u,b_1.a.a.c=3i,b_1.a.a.d=\"d\",b_2.a.a.a=1.000000,b_2.a.a.b=2u,b_2.a.a.c=3i,b_2.a.a.d=\"d\",c=3i,d=\"43.2\",e=23.200000"))
}

func TestOptionalMultipleRepeatedThreeLevelNestingAndSomeChadsEmpty(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads")
	if err != nil {
		assert.NilError(t, err)
	}

	testCase := &lp_testv1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads{
		A: &lp_testv1.OptionalOneLevelNesting{
			A: nil,
		},
		B: []*lp_testv1.TwoLevelNesting{},
		C: 3,
		D: "43.2",
		E: nil,
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.OptionalMultipleRepeatedThreeLevelNestingAndSomeChads c=3i,d=\"43.2\""))
}

func TestImportMessage(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.ImportMessage")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.ImportMessage{
		A: &devicev1.Timestamp{
			Seconds: 123,
			Nanos:   256,
		},
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.ImportMessage a.seconds=123i,a.nanos=256i"))
}

func TestMixIndexSmallSetPrimitives(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MixIndexSmallSetPrimitives")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.MixIndexSmallSetPrimitives{
		A: 1,
		B: 2,
		C: 3,
		D: "d",
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MixIndexSmallSetPrimitives a=1.000000,b=2u,c=3i,d=\"d\""))
}

func TestMixIndexOneLevelNesting(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.MixIndexOneLevelNesting")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.MixIndexOneLevelNesting{
		A: &lp_testv1.MixIndexSmallSetPrimitives{
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
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.MixIndexOneLevelNesting a.a=1.000000,a.b=2u,a.c=3i,a.d=\"d\",b=2u,c=0i,d=\"\""))
}

func TestDeviceTlmTsFrac(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsFrac")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsFrac{
		Ts: &devicev1.Timestamp{
			Seconds: 1257894000,
			Nanos:   256,
		},
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsFrac,__tag_building=A,__tag_floor=1 temperature=12i,pressure=42i,humidity=88i 1257894000000000256"))
}

func TestDeviceTlmTsSec(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsSec")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsSec{
		Ts:          int64(1257894000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsSec,__tag_twice=fingers temperature=12i,pressure=42i,humidity=88i 1257894000000000000"))
}

func TestDeviceTlmTsMilli(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsMilli")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsMilli{
		Ts:          int64(1257894000000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsMilli temperature=12i,pressure=42i,humidity=88i 1257894000000000000"))
}

func TestDeviceTlmTsMicro(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsMicro")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsMicro{
		Ts:          int64(1257894000000000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsMicro temperature=12i,pressure=42i,humidity=88i 1257894000000000000"))
}

func TestDeviceTlmTsNano(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsNano")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsNano{
		Ts:          int64(1257894000000000000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsNano temperature=12i,pressure=42i,humidity=88i 1257894000000000000"))
}

func TestDeviceTlmTsUnspecified(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmTsUnspecified")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmTsUnspecified{
		Ts:          int64(1257894000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmTsUnspecified ts=1257894000i,temperature=12i,pressure=42i,humidity=88i"))
}

func TestDeviceTlmNestedTags(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmNestedTags")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmNestedTags{
		Ts: int64(1257894000000000),
		Env: &lp_testv1.EnvTlm{
			Temperature: int32(12),
			Pressure:    int32(42),
			Humidity:    int32(88),
		},
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmNestedTags,__tag_env.floor=two,__tag_env.temperature.field_temp=hot,__tag_level=one env.temperature=12i,env.pressure=42i,env.humidity=88i 1257894000000000"))
}

func TestDeviceTlmFieldTags(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("lp_test.v1.DeviceTlmFieldTags")
	if err != nil {
		assert.NilError(t, err)
	}
	testCase := &lp_testv1.DeviceTlmFieldTags{
		Ts:          int64(1257894000000000),
		Temperature: int32(12),
		Pressure:    int32(42),
		Humidity:    int32(88),
	}

	dataIn, _ := proto.Marshal(testCase)

	// Act
	fn, err := GenerateMarshalFn(map[string]string{}, desc.(protoreflect.MessageDescriptor))
	assert.NilError(t, err)
	lp := Marshal(dataIn, map[string]string{}, fn)
	fmt.Println(lp)

	// Assert
	assert.Equal(t, true, strings.Contains(lp, "lp_test.v1.DeviceTlmFieldTags,__tag_level=one,__tag_temperature.field_temp=hot temperature=12i,pressure=42i,humidity=88i 1257894000000000"))
}

func ptr[T any](t T) *T {
	return &t
}
