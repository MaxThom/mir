package prototlm_srv

import (
	"context"
	"fmt"
	"sync"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/externals/ts"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	proto_lineprotocol "github.com/maxthom/mir/internal/libs/proto/line_protocol"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	"github.com/maxthom/mir/internal/services/schema_cache"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	tlm_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/tlm_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoTlmServer struct {
	ctx            context.Context
	cancelCtx      context.CancelFunc
	wg             *sync.WaitGroup
	tlmStore       ts.TelemetryStore
	sub            *nats.Subscription
	m              *mir.Mir
	devStore       mng.DeviceStore
	devWriters     map[string]map[string]proto_lineprotocol.ProtoBytesToLpFn
	devWritersLock sync.RWMutex
	schStore       *schema_cache.MirProtoCache
}

// TODO prom metics
// - count on number of dev schema
// - count on nb of writers
// - dp count
// - number of device schema fetch

// IDEA clean ingesters map for schema refresh after timespan
// for better schema mng

// Will have to listen to device update event for new
// data that are saved as tag such as namespace and name

type deviceProtoKey struct {
	deviceId     string
	protoMsgName string
}

// DILLEMA could remove all the fields
// and just use the lpPn since each change need to update
// the lpFn

const (
	ServiceName = "mir_telemetry"
)

var (
	uploadMetric = metrics.NewCounter(prometheus.CounterOpts{
		Name: "upload_schema_counter",
		Help: "Upload schema",
	})
	datapointCount = metrics.NewCounter(prometheus.CounterOpts{
		Name: "datapoint_count",
		Help: "Number of datapoint fed into protoproxy from nats",
	})
)

var l zerolog.Logger

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(uploadMetric)
	reg.Register(datapointCount)
}

func NewProtoTlm(logger zerolog.Logger, m *mir.Mir, devStore mng.DeviceStore, tlmStore ts.TelemetryStore) (*ProtoTlmServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	l = logger.With().Str("srv", "prototlm_server").Logger()
	cc, err := schema_cache.NewMirProtoCache(l, m)
	if err != nil {
		cancel()
		return nil, err
	}
	srv := &ProtoTlmServer{
		ctx:        ctx,
		cancelCtx:  cancel,
		wg:         &sync.WaitGroup{},
		tlmStore:   tlmStore,
		m:          m,
		devStore:   devStore,
		devWriters: make(map[string]map[string]proto_lineprotocol.ProtoBytesToLpFn),
		schStore:   cc,
	}
	srv.schStore.AddDeviceUpdateSub(srv.handleDeviceUpdate)
	return srv, nil
}

func (s *ProtoTlmServer) handleInfluxErrorChannel() {
	s.wg.Add(1)
	errorsCh := s.tlmStore.Errors()
	go func() {
		defer s.wg.Done()
		for {
			select {
			case err, ok := <-errorsCh:
				if !ok {
					return
				}
				l.Error().Err(err).Msg("Error writing to InfluxDB")
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *ProtoTlmServer) Serve() error {
	s.handleInfluxErrorChannel()
	if err := s.m.Device().Telemetry().QueueSubscribe(ServiceName, "", s.handleTelemetryStream); err != nil {
		return fmt.Errorf("cannot listen to device telemetry stream: %w", err)
	}
	if err := s.m.Server().ListTelemetry().QueueSubscribe(ServiceName, s.handleTelemetryListRequest); err != nil {
		return fmt.Errorf("cannot listen to device telemetry list request: %w", err)
	}
	return nil
}

func (s *ProtoTlmServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *ProtoTlmServer) handleTelemetryStream(msg *mir.Msg, deviceId string, protoMsgName string, data []byte) {
	// TODO prometheus
	// TODO set maximum relidelivery in subscribe
	// TODO handler error with schema if we can't have it.
	// Nak might just create to many relideveries in case of can't find the schema
	// Maybe a buffer zone using channels to connect many routine
	// of this function

	s.devWritersLock.RLock()
	var devWriter proto_lineprotocol.ProtoBytesToLpFn
	devMsgs, ok := s.devWriters[deviceId]
	if ok {
		devWriter, ok = devMsgs[protoMsgName]
	}
	s.devWritersLock.RUnlock()
	// Mean no ingesters for proto msg, but we might have the schema
	if !ok {
		desc, _, dev, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, protoMsgName, false)
		if err != nil {
			l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
			return
		}
		fn, err := generateIngesters(desc, deviceId, dev)
		// If error, means schema is invalid so request new from device
		if err != nil {
			// TODO possibly different flow depending on error type
			l.Warn().Err(err).Str("deviceId", deviceId).Str("protoMsg", protoMsgName).Msg("Failed to generate ingester function, requesting schema from device")
			desc, _, dev, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, protoMsgName, true)
			if err != nil {
				l.Error().Err(err).Str("deviceId", deviceId).Msg("Failed to retrieve schema from device")
				return
			}
			fn, err = generateIngesters(desc, deviceId, dev)
			if err != nil {
				l.Warn().Err(err).Msg("")
			}
			// TODO what to do with error here, we cant reask the schema
			// forever if fail, we need maybe a retry of 2-3 times and else
			// it creates an alert in prometheus
		}
		l.Info().Str("deviceId", deviceId).Msg("Generated ingesters functions from proto schema")
		s.devWritersLock.Lock()
		if _, ok := s.devWriters[deviceId]; !ok {
			s.devWriters[deviceId] = make(map[string]proto_lineprotocol.ProtoBytesToLpFn)
		}
		s.devWriters[deviceId][protoMsgName] = fn
		devWriter = fn
		s.devWritersLock.Unlock()
	}
	// TODO update function to return error
	lp := devWriter(data, map[string]string{})
	// fmt.Println(lp)
	s.tlmStore.WriteDatapoint(lp)
}

func (s *ProtoTlmServer) handleDeviceUpdate(deviceId string, device mir_models.Device, schema mir_proto.MirProtoSchema) {
	l.Debug().Str("device_id", deviceId).Msg("device updated, invalidating device ingesters")
	s.devWritersLock.Lock()
	delete(s.devWriters, deviceId)
	s.devWritersLock.Unlock()
}

type schemaPerDevices struct {
	sch        *mir_proto.MirProtoSchema
	err        error
	devsId     []string
	devsNameNs []string
}

func (s *ProtoTlmServer) handleTelemetryListRequest(msg *mir.Msg, clientId string, req *tlm_apiv1.SendListTelemetryRequest) ([]*tlm_apiv1.DevicesTelemetry, error) {
	l.Info().Any("req", req).Msg("list telemetry request")

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
		return nil, fmt.Errorf("error listing device from db: %w", err)
	}

	devsTlm := []*tlm_apiv1.DevicesTelemetry{}
	devSchemas := []*schemaPerDevices{}
	for _, dev := range devs {
		reg, _, err := s.schStore.GetDeviceSchema(dev.Spec.DeviceId, req.RefreshSchema)
		if err != nil {
			found := false
			for _, d := range devsTlm {
				if d.Error == err.Error() {
					d.DevicesNamens = append(d.DevicesNamens, dev.GetNameNamespace())
					found = true
				}
			}
			if !found {
				devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
					DevicesNamens: []string{dev.GetNameNamespace()},
					Error:         err.Error(),
				})
			}
			continue
		}
		found := false
		for _, sch := range devSchemas {
			if mir_proto.AreSchemaEqual(sch.sch, reg) {
				sch.devsId = append(sch.devsId, dev.Spec.DeviceId)
				sch.devsNameNs = append(sch.devsNameNs, dev.GetNameNamespace())
				found = true
			}

		}
		if !found {
			devSchemas = append(devSchemas, &schemaPerDevices{
				sch:        reg,
				devsId:     []string{dev.Spec.DeviceId},
				devsNameNs: []string{dev.GetNameNamespace()},
			})
		}
	}

	for _, sch := range devSchemas {
		tlms, err := sch.sch.GetTelemetryList(req.Measurements, req.Filters)
		if err != nil {
			devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
				DevicesNamens: sch.devsNameNs,
				Error:         err.Error(),
			})
			continue
		}

		for _, tlm := range tlms {
			tlm.Fields, err = s.tlmStore.RetrieveMeasurementsFields(s.ctx, tlm.Name)
			if err != nil {
				tlm.Error = err.Error()
				continue
			}
			tlm.ExploreQuery = s.tlmStore.GetExploreQuery(sch.devsId, tlm.Name)
		}
		devsTlm = append(devsTlm, &tlm_apiv1.DevicesTelemetry{
			DevicesNamens:  sch.devsNameNs,
			TlmDescriptors: tlms,
		})
	}

	l.Info().Msg("list command request processed successfully")
	return devsTlm, nil
}

// TODO have some device info
func generateIngesters(desc protoreflect.Descriptor, deviceId string, device mir_models.Device) (proto_lineprotocol.ProtoBytesToLpFn, error) {
	tags := map[string]string{
		"__id":        deviceId,
		"__name":      device.Meta.Name,
		"__namespace": device.Meta.Namespace,
	}
	for k, v := range device.Meta.Labels {
		tags["__label_"+k] = v
	}
	fn, err := proto_lineprotocol.GenerateMarshalFn(tags, desc.(protoreflect.MessageDescriptor))
	if err != nil {
		return nil, err
	}

	return fn, err
}
