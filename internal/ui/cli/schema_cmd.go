package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/ui"
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
	Refresh SchemaRefreshCmd `cmd:"" help:"Refresh schema from device"`
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

type SchemaRefreshCmd struct {
	Output string `short:"o" help:"output format for response [pretty|json|yaml]" default:"pretty"`
	Target `embed:"" prefix:"target."`
	NameNs string `name:"name/namespace" arg:"" optional:"" help:"edit single device"`
}

func (d *SchemaUploadCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Schemas) == 0 &&
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

func (d *SchemaUploadCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
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

	list, err := m.Client().UpdateDevice().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
		Schemas:    d.Target.Schemas,
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
		len(d.Target.Schemas) == 0 &&
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

func (d *SchemaExploreCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	list, err := m.Client().ListDevice().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
		Schemas:    d.Target.Schemas,
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
		if len(dev.Status.Schema.CompressedSchema) == 0 {
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

func (d *SchemaRefreshCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Schemas) == 0 &&
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

type DevSchemaOutput struct {
	NameNs          string
	PackageNames    []string
	LastSchemaFetch time.Time
	Status          string
	ErrMsg          string
}

func (d *SchemaRefreshCmd) Run(log zerolog.Logger, m *mir.Mir, cfg ui.Config) error {
	list, err := m.Client().RefreshSchema().Request(mir_v1.DeviceTarget{
		Ids:        d.Target.Ids,
		Names:      d.Target.Names,
		Namespaces: d.Target.Namespaces,
		Labels:     d.Target.Labels,
		Schemas:    d.Target.Schemas,
	})
	if err != nil {
		return fmt.Errorf("error publishing device refresh schema request: %w", err)
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

	output := make([]DevSchemaOutput, len(list))
	for i, d := range list {
		nameNs := fmt.Sprintf("%s/%s", d.Device.Meta.Name, d.Device.Meta.Namespace)
		status := "success"
		if d.Error != "" {
			status = "failure"
		}
		output[i] = DevSchemaOutput{
			NameNs:          nameNs,
			PackageNames:    d.Device.Status.Schema.PackageNames,
			Status:          status,
			LastSchemaFetch: d.Device.Status.Schema.LastSchemaFetch.UTC(),
			ErrMsg:          d.Error,
		}
	}

	switch d.Output {
	case "pretty":
		fmt.Println(prettyRefreshSchema(output))
	case "json", "yaml":
		if out, e := marshalResponse(d.Output, output); e != nil {
			return fmt.Errorf("error marshalling response: %w", e)
		} else {
			fmt.Println(out)
		}
	default:
		return fmt.Errorf("invalid output format: %s", d.Output)
	}

	return nil
}

func prettyRefreshSchema(devs []DevSchemaOutput) string {
	format := "%-45s %-8s %-24s %-40s %s\n"
	timeFormat := "2006-01-02 15:04:05 UTC"
	var sb strings.Builder
	fmt.Fprintf(&sb, format, "NAMESPACE/NAME", "STATUS", "LAST_SCHEMA_FETCH", "PACKAGE_NAMES", "ERROR")

	for _, d := range devs {
		fmt.Fprintf(&sb, format, d.NameNs, d.Status, d.LastSchemaFetch.Format(timeFormat), strings.Join(d.PackageNames, ","), d.ErrMsg)
	}
	return sb.String()
}
