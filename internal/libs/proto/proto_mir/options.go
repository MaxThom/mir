package proto_mir

import (
	"github.com/maxthom/mir/internal/libs/proto/json_template"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func findTimestampField(desc protoreflect.MessageDescriptor) {

}

func RetrieveMessageTags(desc protoreflect.MessageDescriptor) map[string]string {
	tags := make(map[string]string)
	retrieveTagsFromDescriptor(desc, tags)
	return tags
}

func retrieveTagsFromDescriptor(desc protoreflect.MessageDescriptor, tags map[string]string) {
	// Get meta opts
	opts, ok := desc.Options().(*descriptorpb.MessageOptions)
	if ok {
		meta := proto.GetExtension(opts, devicev1.E_Meta).(*devicev1.Meta)
		if meta != nil {
			for k, v := range meta.Tags {
				if v != "" {
					tags[k] = v
				}
			}
		}
	}

	// Get tags of each field and walk through nested messages
	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)
		fieldOpts, ok := fd.Options().(*descriptorpb.FieldOptions)
		if ok {
			fieldMeta := proto.GetExtension(fieldOpts, devicev1.E_FieldMeta).(*devicev1.FieldMeta)
			if fieldMeta != nil {
				for k, v := range fieldMeta.Tags {
					if v != "" {
						tags[k] = v
					}
				}
			}
		}
		if fd.Kind() == protoreflect.MessageKind {
			retrieveTagsFromDescriptor(fd.Message(), tags)
		}
	}
}

func RetrieveMessageArguments(desc protoreflect.MessageDescriptor) map[string]string {
	args := make(map[string]string)
	retrieveArgumentsFromDescriptor(desc, args)
	return args
}

func retrieveArgumentsFromDescriptor(desc protoreflect.MessageDescriptor, args map[string]string) {
	for i := 0; i < desc.Fields().Len(); i++ {
		fd := desc.Fields().Get(i)
		if fd.Kind() == protoreflect.MessageKind {
			retrieveTagsFromDescriptor(fd.Message(), args)
		} else {
			args[string(fd.FullName())] = fd.Kind().String()
		}
	}
}

func GetJsonBoilerTemplate(desc protoreflect.MessageDescriptor) ([]byte, error) {
	return json_template.GenerateTemplate(desc)
}
