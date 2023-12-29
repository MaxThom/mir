package proto_lineprotocol

// TODO could used virtual FS from embed to load all proto file in directory

import (
	_ "embed"
	"fmt"
	"os"
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

func TestReadDyn(t *testing.T) {
	// Arrange
	desc, err := protoRegistry.FindDescriptorByName("store.Todo")
	if err != nil {
		assert.NilError(t, err)
	}

	todo := &gen.Todo{
		Id:          "1",
		Title:       "hello",
		Description: "world !",
	}
	out, _ := proto.Marshal(todo)
	fmt.Println(out)

	// Act
	fn := Marshal(map[string]string{"pin": "yup", "aa": "bb"}, desc.(protoreflect.MessageDescriptor))
	lp, err := fn(out, map[string]string{"ca": "dev", "cb": "dev2"})
	if err != nil {
		assert.NilError(t, err)
	}

	fmt.Println(lp)

	// Assert
	assert.Equal(t, 1, 1)
}
