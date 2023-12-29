package proto_lineprotocol

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

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
