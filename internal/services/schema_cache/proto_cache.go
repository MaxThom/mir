package schema_cache

import (
	"fmt"
	"sync"
	"time"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/proto/proto_mir"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TODO listen to device update event

var l zerolog.Logger

type MirProtoCache struct {
	m         *mir.Mir
	store     mng.DeviceStore
	cache     map[string]cacheEntry
	cacheLock sync.RWMutex
}

func NewMirProtoCache(logger zerolog.Logger, m *mir.Mir, store mng.DeviceStore) *MirProtoCache {
	l = logger.With().Str("sub", "proto_cache").Logger()
	return &MirProtoCache{
		m:     m,
		store: store,
		cache: make(map[string]cacheEntry),
	}
}

type cacheEntry struct {
	dev mir_models.Device
	sch *proto_mir.MirProtoSchema
}

// Get the device schema from cache. If missing or refresh schema is true,
// the cache will be invalidated and schema will be fetch from database or device
// If hard refresh is true, it will fetch from device skipping database
func (c *MirProtoCache) GetDeviceSchema(deviceId string, refreshSchema bool, hardRefreshSchema bool) (*proto_mir.MirProtoSchema, mir_models.Device, error) {
	val, ok := c.cache[deviceId]
	if !ok || val.sch == nil || refreshSchema || hardRefreshSchema {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		dev, sch, err := c.reconcileDeviceSchema(deviceId, hardRefreshSchema)
		c.cache[deviceId] = cacheEntry{
			dev: dev,
			sch: sch,
		}
		if err != nil {
			return nil, mir_models.Device{}, errors.Wrap(err, "cannot reconcile device schema")
		}
	}

	return c.cache[deviceId].sch, c.cache[deviceId].dev, nil
}

// Get device schema and descriptor from cache
// If schema missing, get from db.
// If db missing, fetch from device.
// If refreshSchema is true, force refresh from db
// If hardRefreshSchema is true, force refresh from device
func (c *MirProtoCache) GetDeviceSchemaAndDescriptor(deviceId string, descName string, refreshSchema bool, hardRefreshSchema bool) (protoreflect.Descriptor, *proto_mir.MirProtoSchema, mir_models.Device, error) {
	sch, dev, err := c.GetDeviceSchema(deviceId, refreshSchema, hardRefreshSchema)
	if err != nil {
		return nil, nil, dev, err
	}
	desc, err := sch.FindDescriptorByName(protoreflect.FullName(descName))
	if err != nil {
		// If error finding descriptor, we force a hard refresh
		sch, dev, err = c.GetDeviceSchema(deviceId, refreshSchema, true)
		if err != nil {
			return nil, nil, dev, err
		}
		desc, err = sch.FindDescriptorByName(protoreflect.FullName(descName))
		if err != nil {
			return nil, nil, dev, err
		}
	}
	return desc, sch, dev, nil
}

// Get the proto schema from surrealdb, if missing fetch from device
func (c *MirProtoCache) reconcileDeviceSchema(deviceId string, forceDeviceFetch bool) (mir_models.Device, *proto_mir.MirProtoSchema, error) {
	// 1. Go get schema in surrealdb
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	devs, err := c.store.ListDevice(&core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{deviceId},
		},
	})
	if err != nil {
		return mir_models.Device{}, nil, err
	}
	if !forceDeviceFetch {
		if len(devs) > 0 {
			if devs[0].Status.Schema.CompressedSchema != nil &&
				len(devs[0].Status.Schema.CompressedSchema) != 0 {
				_, reg, err := mir_models.DecompressFileDescriptorSet(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					l.Debug().Msgf("reconciled schema for %s from db", deviceId)
					return devs[0].Device, &proto_mir.MirProtoSchema{Files: reg}, nil
				}
			}
		}
	}

	reg, pbSet, err := c.getProtoSchemaFromDevice(deviceId)
	if err != nil {
		return mir_models.Device{}, nil, err
	}

	// Mainly for extra info
	packNames := []string{}
	reg.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		packNames = append(packNames, string(f.FullName()))
		return true
	})

	compSch, err := mir_models.CompressFileDescriptorSet(pbSet)
	if err != nil {
		return mir_models.Device{}, nil, err
	}

	_, err = c.store.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
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

	l.Debug().Msgf("reconciled schema for %s from device", deviceId)
	return devs[0].Device, &proto_mir.MirProtoSchema{Files: reg}, err
}

func (c *MirProtoCache) getProtoSchemaFromDevice(deviceId string) (*protoregistry.Files, *descriptorpb.FileDescriptorSet, error) {
	schemaResp := &device_apiv1.SchemaRetrieveResponse{}
	err := c.m.SendRequest(mir.Command().V1Alpha().RequestSchema(deviceId, schemaResp))
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
