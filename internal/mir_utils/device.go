package mir_utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
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

func (m *MirProtoSchema) GetCommandsList() ([]string, error) {
	var commands []string
	m.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		for i := 0; i < fd.Messages().Len(); i++ {
			msgDesc := fd.Messages().Get(i)
			opts, ok := msgDesc.Options().(*descriptorpb.MessageOptions)
			if !ok {
				continue
			}
			msgType, ok := proto.GetExtension(opts, devicev1.E_MessageType).(devicev1.MessageType)
			if ok && msgType == devicev1.MessageType_MESSAGE_TYPE_TELECOMMAND {
				commands = append(commands, string(msgDesc.FullName()))
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
