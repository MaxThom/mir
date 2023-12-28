package proto_store

//
// TODO
//  - need to add functions to return file description from meta
//  - the store will need persistence with sqlite or surreal
//  - the store can communicate to other instance of stores
//    to retrieve other schma. it could use bfs or disjkstra algorithm
//    and context token from the start point
//  - could be a generic artifact store

import (
	"os"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	GlobalRegistry *Registry = NewRegistry()
)

type (
	Registry struct {
		registry *protoregistry.Files
		// uuid is key
		meta map[uuid.UUID]Meta
	}
	Meta struct {
		Uuid uuid.UUID
		Name string
		Desc string
		Tags map[string]string
	}
)

func NewRegistry() *Registry {
	return &Registry{
		registry: &protoregistry.Files{},
		meta:     make(map[uuid.UUID]Meta),
	}
}

func (r *Registry) RegisterFile(meta Meta, fd protoreflect.FileDescriptor) error {
	if meta.Uuid.String() == "" {
		meta.Uuid = uuid.New()

	}
	r.meta[meta.Uuid] = meta
	return r.registry.RegisterFile(fd)
}

func (r *Registry) RegisterFileProto(meta Meta, p *descriptorpb.FileDescriptorProto) error {
	// Initialize the File descriptor object
	fd, err := protodesc.NewFile(p, r.registry)
	if err != nil {
		return err
	}
	return r.RegisterFile(meta, fd)
}

func (r *Registry) LoadProtoBinaryFileFromDisk(meta Meta, path string) error {
	fd, err := readProtoFromDisk(path)
	if err != nil {
		return err
	}
	return r.RegisterFileProto(meta, fd)
}

func (r *Registry) RangeFiles(fn func(fd protoreflect.FileDescriptor) bool) {
	r.registry.RangeFiles(fn)
}

func (r *Registry) RangeFilesByPackage(name protoreflect.FullName, fn func(fd protoreflect.FileDescriptor) bool) {
	r.registry.RangeFilesByPackage(name, fn)
}

func (r *Registry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return r.registry.FindDescriptorByName(name)
}

func (r *Registry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	return r.registry.FindFileByPath(path)
}

func (r *Registry) NumFiles() int {
	return r.registry.NumFiles()
}

func (r *Registry) NumFilesByPackage(name protoreflect.FullName) int {
	return r.registry.NumFilesByPackage(name)
}

func (r *Registry) ListFilesMeta() []Meta {
	var files []Meta
	for _, m := range r.meta {
		files = append(files, Meta{
			Uuid: m.Uuid,
			Name: m.Name,
			Desc: m.Desc,
			Tags: m.Tags,
		})
	}
	return files
}

func (r *Registry) FindMetaByName(name string) (Meta, bool) {
	for _, m := range r.meta {
		if m.Name == name {
			return m, true
		}
	}
	return Meta{}, false
}

func (r *Registry) FindMetaByUuid(uuid uuid.UUID) (Meta, bool) {
	if meta, ok := r.meta[uuid]; ok {
		return meta, true

	}
	return Meta{}, false
}

func readProtoFromDisk(path string) (*descriptorpb.FileDescriptorProto, error) {
	// Now load that temporary file as a file descriptor
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
