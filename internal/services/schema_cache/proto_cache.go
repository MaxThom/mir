package schema_cache

import (
	"fmt"
	"sync"
	"time"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var l zerolog.Logger

type MirProtoCache struct {
	m           *mir.Mir
	cache       map[string]cacheEntry
	cacheLock   sync.RWMutex
	subscribers []func(deviceId string, device mir_models.Device, schema mir_proto.MirProtoSchema)
}

func NewMirProtoCache(logger zerolog.Logger, m *mir.Mir) (*MirProtoCache, error) {
	l = logger.With().Str("sub", "proto_cache").Logger()
	cache := &MirProtoCache{
		m:     m,
		cache: make(map[string]cacheEntry),
	}
	if err := m.Event().DeviceUpdate().Subscribe(cache.deviceUpdateSub); err != nil {
		return nil, fmt.Errorf("error subscribing to device update event: %w", err)
	}
	return cache, nil
}

type cacheEntry struct {
	dev mir_models.Device
	sch *mir_proto.MirProtoSchema
}

func (c *MirProtoCache) AddDeviceUpdateSub(fn func(deviceId string, device mir_models.Device, schema mir_proto.MirProtoSchema)) {
	c.subscribers = append(c.subscribers, fn)
}

// Get the device schema from cache. If missing or refresh schema is true,
// the cache will be invalidated and schema will be fetch from database or device
func (c *MirProtoCache) GetDeviceSchema(deviceId string, refreshSchema bool) (*mir_proto.MirProtoSchema, mir_models.Device, error) {
	c.cacheLock.RLock()
	val, ok := c.cache[deviceId]
	c.cacheLock.RUnlock()
	if !ok || val.sch == nil || refreshSchema {
		dev, sch, err := c.reconcileDeviceSchema(deviceId, refreshSchema)
		c.cacheLock.Lock()
		c.cache[deviceId] = cacheEntry{
			dev: dev,
			sch: sch,
		}
		c.cacheLock.Unlock()
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Msg("cannot reconcile device schema")
			return nil, mir_models.Device{}, errors.Wrap(err, "cannot reconcile device schema")
		}
	}

	return c.cache[deviceId].sch, c.cache[deviceId].dev, nil
}

func (c *MirProtoCache) FindMessageDescriptor(deviceId string, sch *mir_proto.MirProtoSchema, msgName string) (protoreflect.Descriptor, *mir_proto.MirProtoSchema, error) {
	desc, err := sch.FindDescriptorByName(protoreflect.FullName(msgName))
	if err != nil {
		// If error finding descriptor, we force a hard refresh
		sch, _, err = c.GetDeviceSchema(deviceId, true)
		if err != nil {
			return nil, nil, err
		}
		desc, err = sch.FindDescriptorByName(protoreflect.FullName(msgName))
		if err != nil {
			return nil, nil, err
		}
	}
	return desc, sch, nil
}

// Get device schema and descriptor from cache
// If schema missing, get from db.
// If db missing, fetch from device.
// If refreshSchema is true, force refresh from db
func (c *MirProtoCache) GetDeviceSchemaAndDescriptor(deviceId string, descName string, refreshSchema bool) (protoreflect.Descriptor, *mir_proto.MirProtoSchema, mir_models.Device, error) {
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
func (c *MirProtoCache) reconcileDeviceSchema(deviceId string, forceDeviceFetch bool) (mir_models.Device, *mir_proto.MirProtoSchema, error) {
	// 1. Go get schema in db
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	l.Debug().Str("device_id", deviceId).Msg("device schema not in cache, reconciling...")
	respList := &core_apiv1.ListDeviceResponse{}
	devs, err := c.m.Server().ListDevice().Request(
		&core_apiv1.ListDeviceRequest{
			Targets: &core_apiv1.Targets{
				Ids: []string{deviceId},
			},
		},
	)
	if err != nil {
		return mir_models.Device{}, nil, fmt.Errorf("error listing devices: %s", respList.GetError())
	}
	if len(devs) == 0 {
		return mir_models.Device{}, nil, fmt.Errorf("device %s not found", deviceId)
	}
	if !forceDeviceFetch {
		if len(devs) > 0 {
			if devs[0].Status.Schema.CompressedSchema != nil &&
				len(devs[0].Status.Schema.CompressedSchema) != 0 {
				sch, err := mir_proto.DecompressSchema(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from db", deviceId)
					return devs[0], sch, nil
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

	_, err = c.m.Server().UpdateDevice().Request(&core_apiv1.UpdateDeviceRequest{
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
	},
	)
	if err != nil {
		return mir_models.Device{}, nil, fmt.Errorf("error updating device: %w", err)
	}

	l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from device", deviceId)
	return devs[0], sch, err
}

func (c *MirProtoCache) getProtoSchemaFromDevice(deviceId string) (*mir_proto.MirProtoSchema, error) {
	sch, err := c.m.Device().Schema().Request(deviceId)
	if err != nil {
		return nil, err
	}

	return sch, nil
}

func (c *MirProtoCache) deviceUpdateSub(msg *mir.Msg, deviceId string, device mir_models.Device) {
	// TODO this wont work if one instance of Mir with many cache from flux or cmd. If we have single binary
	// need a subcomponent header or something
	if c.m.GetInstanceName() == msg.GetOriginalTriggerId() {
		msg.Ack()
		return
	}
	sch, err := mir_proto.DecompressSchema(device.Status.Schema.CompressedSchema)
	if err != nil {
		l.Error().Str("device_id", deviceId).Err(err).Msg("error decompressing schema")
		return
	}
	l.Info().Str("device_id", deviceId).Msg("cache updated")
	c.cacheLock.Lock()
	c.cache[deviceId] = cacheEntry{
		dev: device,
		sch: sch,
	}
	c.cacheLock.Unlock()
	for _, fn := range c.subscribers {
		fn(deviceId, device, *sch)
	}
	msg.Ack()
}
