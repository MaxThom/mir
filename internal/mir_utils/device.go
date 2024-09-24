package mir_utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"

	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	//devicev1 "github.com/maxthom/mir/internal/services/protocmd_srv/proto_test/gen/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type MirProtoSchema struct {
	*protoregistry.Files
}

func (m *MirProtoSchema) GetCommandsList() ([]*cmd_apiv1.CommandDescriptor, error) {
	commands := []*cmd_apiv1.CommandDescriptor{}
	// 1. Add labels
	// 2. Add arguments
	// 3. Add json template
	// 4. Add description metadata?
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Messages().Len(); i++ {
			msgDesc := fd.Messages().Get(i)
			opts, ok := msgDesc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				continue
			}
			//import "google.golang.org/protobuf/encoding/protojson"
			//m := protojson.MarshalOptions{EmitUnpopulated: true}
			//resp, err := m.Marshal(w)
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELECOMMAND {
				boiler, err := proto_lineprotocol.GetJsonBoilerTemplate(msgDesc)
				if err != nil {
					fmt.Println(err)
				}
				cmd := cmd_apiv1.CommandDescriptor{
					Name:        string(msgDesc.FullName()),
					Description: "",
					Labels:      proto_lineprotocol.RetrieveMessageTags(msgDesc),
					Arguments:   proto_lineprotocol.RetrieveMessageArguments(msgDesc),
					Boilerplate: string(boiler),
				}
				commands = append(commands, &cmd)
			}
		}
		return true
	})
	return commands, nil
}

func ReconcileDeviceSchema(m *mir.Mir, store mng.DeviceStore, deviceId string, forceDeviceFetch bool) (*MirProtoSchema, error) {
	// 1. Go get schema in surrealdb
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	if !forceDeviceFetch {
		devs, err := store.ListDevice(&core_apiv1.ListDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{deviceId},
			},
		})
		if err != nil {
			return nil, err
		}
		if len(devs) > 0 {
			if devs[0].Status.Schema.CompressedSchema != nil &&
				len(devs[0].Status.Schema.CompressedSchema) != 0 {
				_, reg, err := mir_models.DecompressFileDescriptorSet(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					return &MirProtoSchema{reg}, nil
				}
			}
		}
	}

	reg, pbSet, err := getProtoSchemaFromDevice(m, deviceId)
	if err != nil {
		return nil, err
	}

	// Mainly for extra info
	packNames := []string{}
	reg.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		packNames = append(packNames, string(f.FullName()))
		return true
	})

	compSch, err := mir_models.CompressFileDescriptorSet(pbSet)
	if err != nil {
		return nil, err
	}

	_, err = store.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{deviceId},
		},
		Status: &core_apiv1.UpdateDeviceRequest_Status{
			Schema: &core_apiv1.UpdateDeviceRequest_Schema{
				CompressedSchema: compSch,
				PackageNames:     packNames,
				LastSchemaFetch:  mir_models.AsProtoTimestamp(time.Now().UTC()),
			},
		},
	})

	return &MirProtoSchema{reg}, err
}

func getProtoSchemaFromDevice(m *mir.Mir, deviceId string) (*protoregistry.Files, *descriptorpb.FileDescriptorSet, error) {
	schemaResp := &device_apiv1.SchemaRetrieveResponse{}
	err := m.SendRequest(mir.Command().V1Alpha().RequestSchema(deviceId, schemaResp))
	if err != nil {
		return nil, nil, err
	} else if schemaResp.GetError() != nil {
		e := schemaResp.GetError()
		return nil, nil, errors.New(fmt.Sprintf("%d - %s\n%s", e.Code, e.Message, e.Details))
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(schemaResp.GetSchema(), pbSet); err != nil {
		return nil, nil, err
	}

	reg, err := protodesc.NewFiles(pbSet)
	if err != nil {
		return nil, nil, err
	}

	return reg, pbSet, nil
}
