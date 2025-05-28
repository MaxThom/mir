package cli

import (
	"fmt"
	"os"

	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO maybe add a schema refresh command
type SchemaCmd struct {
	Upload  SchemaUploadCmd  `cmd:"" help:"Upload schema of a set of devices. To generate: 'protoc <proto_files...> --descriptor_set_out=./<file_name>.bproto --include_imports'"`
	Explore SchemaExploreCmd `cmd:"" help:"Explore a device schema"`
}

type SchemaUploadCmd struct {
	Output string `short:"o" help:"output format for response" default:"json"`
	Target `embed:"" prefix:"target."`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
	// TODO could be an array, I think there is a path type
	Path string `type:"path" help:"Path to protobuf schema"`
}

type SchemaExploreCmd struct {
	Output            string `short:"o" help:"output format for response [json|yaml]" default:"yaml"`
	IncludeMirImports bool   `short:"i" help:"includes Mir proto dependencies" default:"false"`
	Target            `embed:"" prefix:"target."`
	NameNs            string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
}

func (d *SchemaUploadCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		d.NameNs == "" {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
	}

	if d.Path == "" {
		err.Details = append(err.Details, "Invalid protobuf schema path")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *SchemaUploadCmd) Run(log zerolog.Logger, m *mir.Mir, cfg Config) error {
	schemaData, err := os.ReadFile(d.Path)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	sch, err := mir_proto.UnmarshalSchema(schemaData)
	if err != nil {
		return fmt.Errorf("error unmarshalling schema: %w", err)
	}
	compSch, err := sch.CompressSchema()
	if err != nil {
		return fmt.Errorf("error compressing schema: %w", err)
	}
	packNames := sch.GetPackageList()

	list, err := m.Server().UpdateDevice().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
	}, mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
		Schema: mir_v1.Schema{
			CompressedSchema: compSch,
			PackageNames:     packNames,
		},
	}))
	if err != nil {
		return fmt.Errorf("error publishing device update request: %w", err)
	}

	if d.Output == "pretty" {
		fmt.Println(prettyStringDevices(list))
	} else {
		if out, e := marshalResponse(d.Output, list); e != nil {
			return fmt.Errorf("error marshalling response: %w", e)
		} else {
			fmt.Println(out)
		}
	}
	return nil
}

func (d *SchemaExploreCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		d.NameNs == "" {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.NameNs != "" {
		d.Target = getTargetFromNameNs(d.NameNs)
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

type schemaDevices struct {
	DeviceIds []string
	PbSet     *descriptorpb.FileDescriptorSet
}

func (d *SchemaExploreCmd) Run(log zerolog.Logger, m *mir.Mir, cfg Config) error {
	list, err := m.Server().ListDevice().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
	}, false)
	if err != nil {
		return fmt.Errorf("error publishing device list request: %w", err)
	}

	if len(list) == 0 {
		e := MirDeviceNotFoundError{
			Targets: &mir_apiv1.DeviceTarget{
				Ids:        d.Target.Ids,
				Names:      d.Target.Names,
				Namespaces: d.Target.Namespaces,
				Labels:     d.Target.Labels,
			},
		}
		return e
	}

	devSchemas := map[string]schemaDevices{}
	errs := []MirProcessError{}
	for _, dev := range list {
		if dev.Status.Schema.CompressedSchema == nil || len(dev.Status.Schema.CompressedSchema) == 0 {
			continue
		}

		bDecomp, err := zstd.DecompressData(dev.Status.Schema.CompressedSchema)
		if err != nil {
			errs = append(errs, MirProcessError{e: err})
		}

		pbSet := new(descriptorpb.FileDescriptorSet)
		if err := proto.Unmarshal(bDecomp, pbSet); err != nil {
			errs = append(errs, MirProcessError{e: err})
		}
		if !d.IncludeMirImports {
			pbSetSmall := new(descriptorpb.FileDescriptorSet)
			for _, v := range pbSet.File {
				if v.GetName() != "google/protobuf/descriptor.proto" &&
					v.GetName() != "mir/device/v1/mir.proto" {
					pbSetSmall.File = append(pbSetSmall.File, v)
				}
			}
			pbSet = pbSetSmall
		}

		sch, ok := devSchemas[string(bDecomp)]
		if !ok {
			devSchemas[string(bDecomp)] = schemaDevices{
				DeviceIds: []string{dev.Spec.DeviceId},
				PbSet:     pbSet,
			}
		} else {
			sch.DeviceIds = append(sch.DeviceIds, dev.Spec.DeviceId)
			devSchemas[string(bDecomp)] = sch
		}
	}

	devSchemasArray := []struct {
		DeviceIds []string
		PbSet     *descriptorpb.FileDescriptorSet
	}{}

	for _, schema := range devSchemas {
		devSchemasArray = append(devSchemasArray, schema)
	}

	if len(errs) > 0 {
		fmt.Println(errs)
	}

	if out, e := marshalResponse(d.Output, devSchemasArray); e != nil {
		return fmt.Errorf("error marshalling response: %w", e)
	} else {
		fmt.Println(out)
	}

	return nil
}
