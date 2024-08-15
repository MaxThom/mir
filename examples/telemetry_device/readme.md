https://github.com/protocolbuffers/protobuf-go

protoc --go_out=. --go_opt=paths=source_relative --descriptor_set_out=example.pb example.proto

// reflect/protoreflect: Package protoreflect provides interfaces to dynamically manipulate protobuf messages.
// reflect/protoregistry: Package protoregistry provides data structures to register and lookup protobuf descriptor types.
// reflect/protodesc: Package protodesc provides functionality for converting descriptorpb.FileDescriptorProto messages to/from the reflective protoreflect.FileDescriptor.
// reflect/protopath: Package protopath provides a representation of a sequence of protobuf reflection operations on a message.
// reflect/protorange: Package protorange provides functionality to traverse a protobuf message.
//types/descriptorpb: Package descriptorpb is the generated package for google/protobuf/descriptor.proto.


descriptorpb: Best for schema inspection and working with the serialized form of descriptors.
reflect: Ideal for runtime manipulation and introspection of protobuf messages.
protodesc: Converts between descriptorpb.FileDescriptorProto messages and the reflective protoreflect.FileDescriptor.

protoc command.proto telemetry.proto utils.proto --descriptor_set_out=./gen/schema.bproto --include_imports
