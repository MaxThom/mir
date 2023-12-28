package main

import (
	"fmt"
	"os"

	"github.com/maxthom/mir/api/gen/clients/store"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func main() {

	fmt.Println("--- marshal in, marshal out")
	todo := &store.Todo{
		Id:          "1",
		Title:       "hello",
		Description: "world !",
	}
	out, _ := proto.Marshal(todo)
	fmt.Println(out)

	todoIn := &store.Todo{}
	proto.Unmarshal(out, todoIn)
	fmt.Println(todoIn)
	fmt.Println("---")

	fmt.Println("--- reflect on type")
	reflectOnExistingType(todoIn)
	fmt.Println("---")

	// ProtoRegistry = play with a set of protofiles
	// ProtoReflect = dynamically manipulate messages
	// Descriptor = description of the schema. used to build the callbacks
	// snet use the grpc routing to know which object to deserialize
	// tui will have to use maybe a header in the msg containing the fullname of the type
	fmt.Println("--- Read value dynamicly")
	s := todoIn.ProtoReflect()
	readDyn(out, s.Descriptor())
	fmt.Println("---")

	fmt.Println("--- Read proto from disk")
	protoRegistry := &protoregistry.Files{}
	pf, err := readProtoFromDisk("/home/hexory/code/go/mir/api/clients/store/todo.bproto")
	if err != nil {
		fmt.Println(err)
	}
	err = registerProto(pf, protoRegistry)
	if err != nil {
		fmt.Println(err)
	}

	protoRegistry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		fmt.Println(fd)
		fmt.Println("Fullname: " + fd.FullName())
		fmt.Println("Name: " + fd.Name())
		fmt.Printf("fd.Messages(): %v\n", fd.Messages().Get(0).FullName())
		return true
	})
	fmt.Println("---")

	fmt.Println("--- Description of message")
	desc, err := protoRegistry.FindDescriptorByName("store.Todo")
	fmt.Println(desc)
	fmt.Println(err)
	readDyn(out, desc.(protoreflect.MessageDescriptor))
	fmt.Println("---")
}

func readDyn(in []byte, desc protoreflect.MessageDescriptor) {
	m := dynamicpb.NewMessage(desc)

	if err := proto.Unmarshal(in, m); err != nil {
		//		return nil, err
	}

	fmt.Println(m.Descriptor().FullName())
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fmt.Printf("  %s: %s\n", fd.Name(), v)
		return true
	})
}

func reflectOnExistingType(msg proto.Message) {
	m := msg.ProtoReflect()
	//fds := m.Descriptor().Fields()
	fmt.Println("----- newReflect:")
	fmt.Println(m.Descriptor())
	// for k := 0; k < fds.Len(); k++ {
	// 	fd := fds.Get(k)
	// 	fmt.Println(fd)
	// }
}

func registerProto(p *descriptorpb.FileDescriptorProto, r *protoregistry.Files) error {
	// Initialize the File descriptor object
	fd, err := protodesc.NewFile(p, r)
	if err != nil {
		return err
	}

	return r.RegisterFile(fd)
}

func readProtoFromDisk(path string) (*descriptorpb.FileDescriptorProto, error) {
	// Now load that temporary file as a file descriptor set protobuf
	protoFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(protoFile, pbSet); err != nil {
		return nil, err
	}

	// We know protoc was invoked with a single .proto file
	return pbSet.GetFile()[0], nil
}
