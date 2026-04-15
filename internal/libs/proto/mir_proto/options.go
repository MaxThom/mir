package mir_proto

import (
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func findTimestampField(desc protoreflect.MessageDescriptor) {

}

func RetrieveMessageTags(desc protoreflect.MessageDescriptor) map[string]string {
	tags := make(map[string]string)
	retrieveTagsFromDescriptor(desc, "", tags)
	return tags
}

func retrieveTagsFromDescriptor(desc protoreflect.MessageDescriptor, nestedKey string, tags map[string]string) {
	// Get meta opts
	opts, ok := desc.Options().(*descriptorpb.MessageOptions)
	if ok {
		meta := proto.GetExtension(opts, devicev1.E_Meta).(*devicev1.Meta)
		if meta != nil {
			for k, v := range meta.Tags {
				if v != "" {
					tags["__tag_"+nestedKey+k] = v
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
						tags["__tag_"+nestedKey+string(fd.Name())+"."+k] = v
					}
				}
				if fieldMeta.Unit != "" {
					tags["__unit_"+nestedKey+string(fd.Name())] = fieldMeta.Unit
				}
			}
		}
		if fd.Kind() == protoreflect.MessageKind {
			retrieveTagsFromDescriptor(fd.Message(), nestedKey+string(fd.Name())+".", tags)
		}
	}
}
