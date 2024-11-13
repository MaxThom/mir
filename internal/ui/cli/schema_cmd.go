package cli

import (
	"fmt"
	"os"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func (d *SchemaUploadCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	schemaData, err := os.ReadFile(d.Path)
	if err != nil {
		e := MirProcessError{e: err}
		fmt.Println(e)
		return e
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(schemaData, pbSet); err != nil {
		e := MirProcessError{e: err}
		fmt.Println(e)
		return e
	}
	compSch, err := mir_models.CompressFileDescriptorSet(pbSet)
	if err != nil {
		e := MirProcessError{e: err}
		fmt.Println(e)
		return e
	}
	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		e := MirProcessError{e: err}
		fmt.Println(e)
		return e
	}

	packNames := []string{}
	reg.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		packNames = append(packNames, string(f.FullName()))
		return true
	})

	req := &core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
		Status: &core_apiv1.UpdateDeviceRequest_Status{
			Schema: &core_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: compSch,
				PackageNames:     packNames,
			},
		},
	}
	resp, err := core_client.PublishDeviceUpdateRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "device.update", e: err}
		fmt.Println(e)
		return e
	}
	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.update",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		fmt.Println(e)
		return e
	}

	if resp.GetOk() != nil {
		list := mir_models.NewDeviceListFromProtoDevices(resp.GetOk().Devices)
		if d.Output == "pretty" {
			fmt.Println(prettyStringDevices(list))
		} else {
			if out, e := MarshalResponse(d.Output, list); e != nil {
				return e
			} else {
				fmt.Println(out)
			}
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

func (d *SchemaExploreCmd) Run(c CLI) error {
	var err error
	msgBus, err := bus.New(c.Target)
	if err != nil {
		e := MirConnectionError{Target: c.Target, e: err}
		fmt.Println(e)
		return e
	}
	defer msgBus.Close()

	req := &core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids:        d.Target.Ids,
			Names:      d.Target.Names,
			Namespaces: d.Target.Namespaces,
			Labels:     d.Target.Labels,
		},
	}
	resp, err := core_client.PublishDeviceListRequest(msgBus, req)
	if err != nil {
		e := MirRequestError{Route: "device.list", e: err}
		return e
	}
	if resp.GetError() != nil {
		e := MirResponseError{
			Route: "device.list",
			e: MirHttpError{
				Code:    resp.GetError().GetCode(),
				Message: resp.GetError().GetMessage(),
				Details: resp.GetError().GetDetails(),
			}}
		return e
	}

	devs := resp.GetOk().GetDevices()
	if len(devs) == 0 {
		e := MirDeviceNotFoundError{
			Targets: &core_apiv1.Targets{
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
	for _, dev := range devs {
		if dev.Status.Schema == nil || dev.Status.Schema.CompressedSchema == nil {
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

	if out, e := MarshalResponse(d.Output, devSchemasArray); e != nil {
		return e
	} else {
		fmt.Println(out)
	}

	return nil
}
