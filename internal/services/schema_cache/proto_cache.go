package schema_cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
)

var (
	schemaCacheCount = metrics.NewGauge(prometheus.GaugeOpts{
		Subsystem: "schemastore",
		Name:      "schema_cache_count",
		Help:      "Number of proto schema in cache",
	})
	schemaReconcileTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "schemastore",
		Name:      "schema_reconciled_total",
		Help:      "Number of proto schema reconciled",
	}, []string{"source"})
	schemaReconcileErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "schemastore",
		Name:      "schema_reconciled_error_total",
		Help:      "Number of proto schema reconciled in error",
	})

	l zerolog.Logger
)

func init() {
	schemaReconcileTotal.With(prometheus.Labels{"source": "database"}).Add(0)
	schemaReconcileTotal.With(prometheus.Labels{"source": "device"}).Add(0)
	schemaReconcileTotal.With(prometheus.Labels{"source": "cache"}).Add(0)
	schemaReconcileTotal.With(prometheus.Labels{"source": "event"}).Add(0)
}

type MirSchemaCache struct {
	m           *mir.Mir
	cache       map[string]cacheEntry
	cacheLock   sync.RWMutex
	subscribers []func(deviceId string, device mir_v1.Device, schema mir_proto.MirProtoSchema)
}

func NewMirSchemaCache(logger zerolog.Logger, m *mir.Mir) (*MirSchemaCache, error) {
	l = logger.With().Str("sub", "proto_cache").Logger()
	cache := &MirSchemaCache{
		m:     m,
		cache: make(map[string]cacheEntry),
	}
	if err := m.Event().DeviceUpdate().Subscribe(cache.deviceUpdateSub); err != nil {
		return nil, fmt.Errorf("error subscribing to device update event: %w", err)
	}
	return cache, nil
}

type cacheEntry struct {
	dev mir_v1.Device
	sch *mir_proto.MirProtoSchema
}

func (c *MirSchemaCache) AddDeviceUpdateSub(fn func(deviceId string, device mir_v1.Device, schema mir_proto.MirProtoSchema)) {
	c.subscribers = append(c.subscribers, fn)
}

// Get the device schema from cache. If missing or refresh schema is true,
// the cache will be invalidated and schema will be fetch from database or device
func (c *MirSchemaCache) GetDeviceSchema(deviceId string, refreshSchema bool) (*mir_proto.MirProtoSchema, mir_v1.Device, error) {
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
		schemaCacheCount.Inc()
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Msg("cannot reconcile device schema")
			schemaReconcileErrorTotal.Inc()
			return nil, mir_v1.Device{}, errors.Wrap(err, "cannot reconcile device schema")
		}
	} else {
		schemaReconcileTotal.WithLabelValues("cache").Inc()
	}

	return c.cache[deviceId].sch, c.cache[deviceId].dev, nil
}

func (c *MirSchemaCache) FindMessageDescriptor(deviceId string, sch *mir_proto.MirProtoSchema, msgName string) (protoreflect.Descriptor, *mir_proto.MirProtoSchema, error) {
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
func (c *MirSchemaCache) GetDeviceSchemaAndDescriptor(deviceId string, descName string, refreshSchema bool) (protoreflect.Descriptor, *mir_proto.MirProtoSchema, mir_v1.Device, error) {
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
func (c *MirSchemaCache) reconcileDeviceSchema(deviceId string, forceDeviceFetch bool) (mir_v1.Device, *mir_proto.MirProtoSchema, error) {
	// 1. Go get schema in db
	// 2. If not there, fetch from device
	// 3. Update db
	// IDEA refresh if last fetch is older then a timespan
	if !forceDeviceFetch {
		l.Debug().Str("device_id", deviceId).Msg("device schema not in cache, reconciling...")
		devs, err := c.m.Client().ListDevice().Request(
			mir_v1.DeviceTarget{
				Ids: []string{deviceId},
			}, false)
		if err != nil {
			// If error, we fetch from device
			if !strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
				return mir_v1.Device{}, nil, fmt.Errorf("error listing devices: %s", err)
			}
		}
		if len(devs) == 0 {
			// If error, we fetch from device
			if err != nil && !strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
				return mir_v1.Device{}, nil, fmt.Errorf("device %s not found", deviceId)
			}
		}
		if len(devs) > 0 {
			if len(devs[0].Status.Schema.CompressedSchema) != 0 {
				sch, err := mir_proto.DecompressSchema(devs[0].Status.Schema.CompressedSchema)
				if err == nil {
					l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from db", deviceId)
					schemaReconcileTotal.WithLabelValues("database").Inc()
					return devs[0], sch, nil
				}
				// If error, we fetch from device
			}
		}
	}

	l.Debug().Str("device_id", deviceId).Msg("device schema not in db, fetching from device...")
	sch, err := c.getProtoSchemaFromDevice(deviceId)
	if err != nil {
		return mir_v1.Device{}, nil, err
	}
	compressSch, err := sch.CompressSchema()
	if err != nil {
		return mir_v1.Device{}, nil, err
	}

	timeNow := time.Now().UTC()
	devResp, err := c.m.Client().UpdateDevice().Request(
		mir_v1.DeviceTarget{
			Ids: []string{deviceId},
		},
		mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
			Schema: mir_v1.Schema{
				CompressedSchema: compressSch,
				PackageNames:     sch.GetPackageList(),
				LastSchemaFetch:  &surrealdbModels.CustomDateTime{Time: timeNow},
			},
		}),
	)
	if err != nil {
		if strings.Contains(err.Error(), surreal.ErrDatabaseDisconnected.Error()) {
			schemaReconcileTotal.WithLabelValues("device").Inc()
			l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from device", deviceId)
			return mir_v1.NewDevice().WithId(deviceId), sch, nil
		}
		return mir_v1.Device{}, nil, fmt.Errorf("error updating device: %w", err)
	}
	if len(devResp) == 0 {
		return mir_v1.Device{}, nil, fmt.Errorf("no device found")
	}

	schemaReconcileTotal.WithLabelValues("device").Inc()
	l.Info().Str("device_id", deviceId).Msgf("reconciled schema for %s from device", deviceId)
	return devResp[0], sch, err
}

func (c *MirSchemaCache) getProtoSchemaFromDevice(deviceId string) (*mir_proto.MirProtoSchema, error) {
	sch, err := c.m.Device().Schema().Request(deviceId)
	if err != nil {
		return nil, err
	}

	return sch, nil
}

func (c *MirSchemaCache) deviceUpdateSub(msg *mir.Msg, deviceId string, device mir_v1.Device, err error) {
	// TODO this wont work if one instance of Mir with many cache from flux or cmd. If we have single binary
	// need a subcomponent header or something
	// if slices.Contains(msg.GetTriggerChain(), c.m.GetInstanceName()) {
	// 	// if c.m.GetInstanceName() == msg.GetTriggerChain() {
	// 	msg.Ack()
	// 	return
	// }

	// We dont update the cache with new elements.
	// It has to be requested first
	// c.cacheLock.RLock()
	// if _, ok := c.cache[deviceId]; !ok {
	// 	msg.Ack()
	// 	c.cacheLock.RUnlock()
	// 	return
	// }
	// c.cacheLock.RUnlock()

	if err != nil {
		l.Error().Str("device_id", deviceId).Err(err).Msg("error deserializing event")
		return
	}
	sch, err := mir_proto.DecompressSchema(device.Status.Schema.CompressedSchema)
	if err != nil {
		l.Error().Str("device_id", deviceId).Err(err).Msg("error decompressing schema")
		return
	}
	l.Info().Str("device_id", deviceId).Msg("cache updated")
	c.cacheLock.Lock()
	if _, ok := c.cache[deviceId]; !ok {
		schemaCacheCount.Inc()
	}
	c.cache[deviceId] = cacheEntry{
		dev: device,
		sch: sch,
	}
	c.cacheLock.Unlock()
	schemaReconcileTotal.WithLabelValues("event").Inc()
	for _, fn := range c.subscribers {
		fn(deviceId, device, *sch)
	}
	msg.Ack()
}

func (c *MirSchemaCache) GetDynamicMsg(deviceId string, protoMsgName string, data []byte) (*dynamicpb.Message, error) {
	desc, _, _, err := c.GetDeviceSchemaAndDescriptor(deviceId, protoMsgName, false)
	if err != nil {
		return nil, err
	}

	m := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
	if err := proto.Unmarshal(data, m); err != nil {
		return nil, err
	}

	return m, nil
}
