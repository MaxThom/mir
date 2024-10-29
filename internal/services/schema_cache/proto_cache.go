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
	"google.golang.org/protobuf/reflect/protoreflect"
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
func (c *MirProtoCache) GetDeviceSchema(deviceId string, refreshSchema bool) (*proto_mir.MirProtoSchema, mir_models.Device, error) {
	val, ok := c.cache[deviceId]
	if !ok || val.sch == nil || refreshSchema {
		c.cacheLock.Lock()
		defer c.cacheLock.Unlock()
		dev, sch, err := c.reconcileDeviceSchema(deviceId, refreshSchema)
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
func (c *MirProtoCache) GetDeviceSchemaAndDescriptor(deviceId string, descName string, refreshSchema bool) (protoreflect.Descriptor, *proto_mir.MirProtoSchema, mir_models.Device, error) {
	sch, dev, err := c.GetDeviceSchema(deviceId, refreshSchema)
	if err != nil {
		return nil, nil, dev, err
	}
	desc, err := sch.FindDescriptorByName(protoreflect.FullName(descName))
	if err != nil {
		// If error finding descriptor, we force a hard refresh
		sch, dev, err = c.GetDeviceSchema(deviceId, true)
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
	// 1. Go get schema in db
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	l.Debug().Str("device_id", deviceId).Msg("device schema not in cache, reconciling...")
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
				sch, err := proto_mir.DecompressSchema(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from db", deviceId)
					return devs[0].Device, sch, nil
				}
				// If error, we fetch from device
			}
		}
	}

	l.Debug().Str("device_id", deviceId).Msg("device schema not in db, fetching from device...")
	sch, err := c.getProtoSchemaFromDevice(deviceId)
	if err != nil {
		return mir_models.Device{}, nil, err
	}
	compressSch, err := sch.CompressSchema()
	if err != nil {
		return mir_models.Device{}, nil, err
	}

	updResp := &core_apiv1.UpdateDeviceResponse{}
	err = c.m.SendRequest(mir.Resquest().V1Alpha().UpdateDevice(
		core_apiv1.UpdateDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{deviceId},
			},
			Status: &core_apiv1.UpdateDeviceRequest_Status{
				Schema: &core_apiv1.UpdateDeviceRequest_Schema{
					CompressedSchema: compressSch,
					PackageNames:     sch.GetPackageList(),
					LastSchemaFetch:  mir_models.AsProtoTimestamp(time.Now().UTC()),
				},
			},
		}, updResp))
	if err != nil {
		return mir_models.Device{}, nil, err
	}
	if updResp.GetError() != nil {
		return mir_models.Device{}, nil, fmt.Errorf("%s", updResp.GetError().Message)
	}

	l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from device", deviceId)
	return devs[0].Device, sch, err
}

func (c *MirProtoCache) getProtoSchemaFromDevice(deviceId string) (*proto_mir.MirProtoSchema, error) {
	schemaResp := &device_apiv1.SchemaRetrieveResponse{}
	err := c.m.SendRequest(mir.Command().V1Alpha().RequestSchema(deviceId, schemaResp))
	if err != nil {
		return nil, err
	} else if schemaResp.GetError() != nil {
		e := schemaResp.GetError()
		return nil, errors.New(fmt.Sprintf("%d - %s\n%s", e.Code, e.Message, e.Details))
	}

	// Decompress already from using the sdk
	return proto_mir.UnmarshalSchema(schemaResp.GetSchema())
}
