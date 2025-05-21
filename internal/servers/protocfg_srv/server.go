package protocfg_srv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/proto/json_template"
	"github.com/maxthom/mir/internal/services/schema_cache"
	cfg_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cfg_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoCfgServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	devStore  mng.MirStore
	schStore  *schema_cache.MirProtoCache
}

const (
	ServiceName = "mir_protocfg"
)

var (
	requestTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "request_total",
		Help:      "Number of request for config routes",
	}, []string{"route"})
	requestErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "request_error_total",
		Help:      "Number of error request for config routes",
	}, []string{"route"})
	deviceCfgSentTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_cfg_sent_total",
		Help:      "Number of config sent to devices",
	})
	deviceCfgSentErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_cfg_sent_error_total",
		Help:      "Number of config failed to sent to devices",
	})
	deviceDesiredPropsRequestTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_desired_props_request_total",
		Help:      "Total number of desired properties requests from devices",
	})
	deviceDesiredPropsRequestErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_desired_props_request_error_total",
		Help:      "Total number of desired properties error requests from devices",
	})
	deviceReportedPropsRequestTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_reported_props_request_total",
		Help:      "Total number of reported properties requests from devices",
	})
	deviceReportedPropsRequestErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cfg",
		Name:      "device_reported_props_request_error_total",
		Help:      "Total number of reported properties error requests from devices",
	})

	devMirErrType = devicev1.Error{}
	devMirErrStr  = string(devMirErrType.ProtoReflect().Descriptor().FullName())

	l            zerolog.Logger
	offlineAfter = time.Second * 30
)

func init() {
	requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestTotal.With(prometheus.Labels{"route": "send"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "send"}).Add(0)
}

func NewProtoCfg(logger zerolog.Logger, m *mir.Mir, store mng.MirStore, schemaCache *schema_cache.MirProtoCache) (*ProtoCfgServer, error) {
	l = logger.With().Str("srv", "protocfg_server").Logger()
	ctx, cancelFn := context.WithCancel(context.Background())
	return &ProtoCfgServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		wg:        &sync.WaitGroup{},
		m:         m,
		devStore:  store,
		schStore:  schemaCache,
	}, nil
}

// Using the db and bus, listen for telemetry, deserialize using proto and push to line protocol db
func (s *ProtoCfgServer) Serve() error {
	if err := s.m.Server().SendConfig().QueueSubscribe(ServiceName, s.sendConfigSub); err != nil {
		return err
	}
	if err := s.m.Server().ListConfig().QueueSubscribe(ServiceName, s.listCfgSub); err != nil {
		return err
	}
	if err := s.m.Device().ReportedProperties().QueueSubscribe(ServiceName, "*", s.reportedPropsSub); err != nil {
		return err
	}
	if err := s.m.Device().DesiredProperties().QueueSubscribe(ServiceName, "*", s.desiredPropsSub); err != nil {
		return err
	}
	return nil
}

func (s *ProtoCfgServer) Shutdown() error {
	s.cancelCtx()
	s.wg.Wait()
	return nil
}

func (s *ProtoCfgServer) listCfgSub(msg *mir.Msg, clientId string, req *cfg_apiv1.SendListConfigRequest) (map[string]*cfg_apiv1.Configs, error) {
	l.Info().Any("req", req).Msg("list config request")
	requestTotal.WithLabelValues("list").Inc()
	// 1. get device list
	// 2. for each device, get stored schema, if empty, fetch from device
	// 3. return list of config

	devs, err := s.devStore.ListDevice(mir_v1.ProtoDeviceTargetToMirDeviceTarget(req.Targets), false)
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
		requestErrorTotal.WithLabelValues("list").Inc()
		return nil, fmt.Errorf("error listing devices from db: %w", err)
	}

	devsCmds := make(map[string]*cfg_apiv1.Configs)
	for _, dev := range devs {
		reg, _, err := s.schStore.GetDeviceSchema(dev.Spec.DeviceId, req.RefreshSchema)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
				Error: err.Error(),
			}
			continue
		}

		cfgs, err := reg.GetConfigList(req.FilterLabels)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
				Error: err.Error(),
			}
			continue
		}

		cfgList := []*cfg_apiv1.ConfigDescriptor{}
		for _, cfg := range cfgs {
			if v, ok := dev.Properties.Desired[cfg.Name]; ok {
				b, err := json.Marshal(v)
				if err != nil {
					cfg.Error = err.Error()
					continue
				}
				cfg.Values = string(b)
			}
			cfgList = append(cfgList, cfg)
		}
		devsCmds[dev.GetNameNamespace()] = &cfg_apiv1.Configs{
			Configs: cfgList,
		}
	}

	l.Info().Msg("list config request processed successfully")
	return devsCmds, nil
}

func (s *ProtoCfgServer) sendConfigSub(msg *mir.Msg, clientId string, req *cfg_apiv1.SendConfigRequest) (*cfg_apiv1.SendConfigResponse_ConfigResponses, error) {
	l.Info().Any("req", req).Msg("send config request")
	requestTotal.WithLabelValues("send").Inc()

	errs := []string{}
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 {
		errs = append(errs, mir_v1.ErrorNoDeviceTargetProvided.Error())
	}
	if req.Name == "" {
		errs = append(errs, mir_v1.ErrorCommandNameNotProvided.Error())
	}
	if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_UNSPECIFIED && !req.ShowTemplate {
		errs = append(errs, mir_v1.ErrorCommandEncodingNotSpecified.Error())
	}
	if (req.Payload == nil || len(req.Payload) == 0) && !req.ShowTemplate && !req.ShowValues && req.PayloadEncoding != common_apiv1.Encoding_ENCODING_PROTOBUF {
		// Proto encoding can be empty if struct is empty, not json
		errs = append(errs, mir_v1.ErrorCommandPayloadNotProvided.Error())
	}
	if len(errs) > 0 {
		l.Error().Err(fmt.Errorf("%w: %s", mir_v1.ErrorBadRequest, strings.Join(errs, ", "))).Msg("")
		requestErrorTotal.WithLabelValues("send").Inc()
		return nil, fmt.Errorf("%w: %s", mir_v1.ErrorBadRequest, errs)
	}
	// If command was specified with labels
	if index := strings.Index(req.Name, "{"); index != -1 {
		req.Name = req.Name[:index]
	}

	resp, err := s.sendConfigToDevices(msg, req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while processing send config request")
		requestErrorTotal.WithLabelValues("send").Inc()
		return nil, fmt.Errorf("error sending config to devices: %w", err)
	}

	l.Info().Msg("send config request processed successfully")
	return &cfg_apiv1.SendConfigResponse_ConfigResponses{
		DeviceResponses: resp,
		Encoding:        req.PayloadEncoding,
	}, nil
}

type cmdDevicePayload struct {
	deviceId   string
	payload    []byte
	mapPayload map[string]any
	time       time.Time
	msgDesc    protoreflect.MessageDescriptor
}

func (s *ProtoCfgServer) sendConfigToDevices(msg *mir.Msg, req *cfg_apiv1.SendConfigRequest) (map[string]*cfg_apiv1.SendConfigResponse_ConfigResponse, error) {
	devs, err := s.devStore.ListDevice(mir_v1.ProtoDeviceTargetToMirDeviceTarget(req.Targets), false)
	if err != nil {
		return nil, err
	} else if len(devs) == 0 {
		return nil, mng.ErrorNoDeviceFound
	}
	devResp := make(map[string]*cfg_apiv1.SendConfigResponse_ConfigResponse)

	// We do validation as we need to cast to JSON for storage
	// This fills configToSend with payload for validated devices
	// It also fills devResp with devices in error
	configToSend := make(map[string]*cmdDevicePayload)
	devInError := false
	for _, dev := range devs {
		// Retrieve descriptor
		msgReqDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(dev.Spec.DeviceId, req.Name, req.RefreshSchema)
		if err != nil {
			l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error retrieving config descriptor from device schema")
			deviceCfgSentErrorTotal.Inc()
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
				Error:    errors.Wrap(err, "error retrieve config descriptor from device schema").Error(),
			}
			devInError = true
			continue
		}

		if req.ShowTemplate {
			tpl, err := json_template.GenerateTemplate(msgReqDesc.(protoreflect.MessageDescriptor), json_template.Options{})
			if err != nil {
				l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error generating config template from device schema")
				deviceCfgSentErrorTotal.Inc()
				devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error generating config template from device schema").Error(),
				}
				continue
			}
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Name:     req.Name,
				Payload:  tpl,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS,
			}
			continue
		} else if req.ShowValues {
			b := []byte{}
			if v, ok := dev.Properties.Desired[req.Name]; ok {
				b, err = json.Marshal(v)
				if err != nil {
					deviceCfgSentErrorTotal.Inc()
					l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error marshalling config from device")
					devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
						DeviceId: dev.Spec.DeviceId,
						Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error marshalling config from device").Error(),
					}
					continue
				}
			} else {
				// Send template in case config is not there yet
				b, err = json_template.GenerateTemplate(msgReqDesc.(protoreflect.MessageDescriptor), json_template.Options{})
				if err != nil {
					l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error generating config template from device schema")
					deviceCfgSentErrorTotal.Inc()
					devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
						DeviceId: dev.Spec.DeviceId,
						Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error generating config template from device schema").Error(),
					}
					continue
				}
			}
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Name:     req.Name,
				Payload:  b,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS,
			}
			continue
		}

		payloadReq := dynamicpb.NewMessage(msgReqDesc.(protoreflect.MessageDescriptor))

		// Encoding and validation
		err = nil
		protoPayload := req.Payload
		jsonPayload := req.Payload
		mapPayload := make(map[string]interface{})
		if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
			// The payload is already in JSON
			// Encoding to proto and then serialize
			err = protojson.Unmarshal(req.Payload, payloadReq)
			if err == nil {
				protoPayload, err = proto.Marshal(payloadReq)
				if err == nil {
					err = json.Unmarshal(jsonPayload, &mapPayload)
				}
			}
		} else if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_PROTOBUF {
			// The payload is in proto, encode to JSON for storage
			err = proto.Unmarshal(req.Payload, payloadReq)
			if err == nil {
				jsonPayload, err = protojson.Marshal(payloadReq)
				if err == nil {
					err = json.Unmarshal(jsonPayload, &mapPayload)
				}
			}
		}
		if err != nil {
			l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error unmarshaling payload")
			deviceCfgSentErrorTotal.Inc()
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
				Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
			}
			devInError = true
			continue
		}
		// Compare if the properties are the same as the last one
		// This might not always work using JSON, check for alternative
		if req.SendOnlyDifferent {
			new, err := minifyJSON(jsonPayload)
			if err != nil {
				l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error minifying json")
				deviceCfgSentErrorTotal.Inc()
				devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
				}
				devInError = true
				continue
			}
			old, err := json.Marshal(dev.Properties.Desired[string(msgReqDesc.FullName())])
			if err != nil {
				l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error unmarshaling device current config")
				deviceCfgSentErrorTotal.Inc()
				devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
				}
				devInError = true
				continue
			}
			if bytes.EqualFold(new, old) {
				devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: dev.Spec.DeviceId,
					Name:     req.Name,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_NOCHANGE,
				}
				continue
			}
		}

		// Add to map with time and name
		timeNow := time.Now().UTC()
		mapPayload = map[string]interface{}{
			string(msgReqDesc.FullName()): mapPayload,
		}

		// Prepare
		devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
			DeviceId: dev.Spec.DeviceId,
			Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED,
		}
		configToSend[dev.GetNameNamespace()] = &cmdDevicePayload{
			time:       timeNow,
			payload:    protoPayload,
			mapPayload: mapPayload,
			deviceId:   dev.Spec.DeviceId,
			msgDesc:    msgReqDesc.(protoreflect.MessageDescriptor),
		}

		l.Info().Str("device_id", dev.Spec.DeviceId).Msgf("config %s validated for device %s", req.Name, dev.GetNameNamespace())
	}

	// If we have validation error or it is a dry run, we return the request
	// If ForcePush is on, means we send the command to each validated devices
	// even if some errors
	if (devInError && !req.ForcePush) || req.DryRun {
		l.Info().Bool("device_in_error", devInError).Bool("force_push", req.ForcePush).Bool("dry_run", req.DryRun).Msgf("config processed but not sent")
		return devResp, nil
	}

	// Sends configs
	wg := &sync.WaitGroup{}
	wg.Add(len(configToSend))

	l.Info().Msgf("sending config %s to targeted devices", req.Name)
	for nameNs, p := range configToSend {
		go func() {
			defer wg.Done()

			timeMap := map[string]time.Time{}
			for k := range p.mapPayload {
				timeMap[k] = p.time
			}
			d := mir_v1.NewDevice().WithProps(mir_v1.DeviceProperties{
				Desired: p.mapPayload,
			}).WithStatus(mir_v1.DeviceStatus{
				Properties: mir_v1.PropertiesTime{
					Desired: timeMap,
				},
			})
			dev, err := s.devStore.UpdateDevice(mir_v1.DeviceTarget{
				Ids: []string{p.deviceId},
			}, d)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error updating device properties in store")
				deviceCfgSentErrorTotal.Inc()
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error updating device properties in store").Error(),
				}
				return
			}

			// Event
			nns := strings.Split(nameNs, "/")
			if err = publishDesiredPropertiesEvent(s.m, msg, nns[0], nns[1], p.deviceId, dev[0].Properties.Desired); err != nil {
				l.Error().Err(err).Msg("error while publishing device config event")
			}

			err = s.m.Device().Config().PublishRaw(p.deviceId, mir.ProtoCmdDesc{
				Name:    req.Name,
				Payload: p.payload,
			}, p.time)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error during sent config request to device")
				deviceCfgSentErrorTotal.Inc()
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error during sent config request to device").Error(),
				}
				return
			}
			deviceCfgSentTotal.Inc()

			dt := dev[0].Properties.Desired[req.Name].(map[string]interface{})
			byteResp, err := json.Marshal(dt)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error marshalling properties to json")
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error marshalling properties to json").Error(),
				}
				return
			}
			if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_PROTOBUF {
				protoResp := dynamicpb.NewMessage(p.msgDesc)
				err = protojson.Unmarshal(byteResp, protoResp)
				if err == nil {
					byteResp, err = proto.Marshal(protoResp)
				}
				if err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error marshalling properties to protobuf")
					deviceCfgSentErrorTotal.Inc()
					devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
						DeviceId: p.deviceId,
						Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error marshalling properties to protobuf").Error(),
					}
					return
				}
			}
			devResp[nameNs].DeviceId = p.deviceId
			devResp[nameNs].Name = req.Name
			devResp[nameNs].Payload = byteResp
			devResp[nameNs].Status = cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS
		}()
	}
	wg.Wait()

	return devResp, nil
}

func (s *ProtoCfgServer) reportedPropsSub(msg *mir.Msg, deviceId string, msgName string, data []byte) {
	deviceReportedPropsRequestTotal.Inc()
	msgDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, msgName, false)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error retrieving config descriptor from device schema")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}

	msgProto := dynamicpb.NewMessage(msgDesc.(protoreflect.MessageDescriptor))
	if err = proto.Unmarshal(data, msgProto); err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error unmarshaling reported properties")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}

	opts := protojson.MarshalOptions{EmitDefaultValues: true}
	msgJson, err := opts.Marshal(msgProto)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error marshaling reported properties to JSON")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}
	msgMap := make(map[string]interface{})
	if err = json.Unmarshal(msgJson, &msgMap); err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error unmarshaling reported properties to map")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}
	timeNow := time.Now().UTC()
	msgMap = map[string]interface{}{
		"properties": map[string]interface{}{
			"reported": map[string]interface{}{
				string(msgDesc.FullName()): msgMap,
			},
		},
		"status": map[string]interface{}{
			"properties": map[string]interface{}{
				"reported": map[string]interface{}{
					string(msgDesc.FullName()): timeNow.Format(time.RFC3339Nano),
				},
			},
		},
	}

	jsonRaw, err := json.Marshal(msgMap)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error marshaling reported properties to JSON")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}

	dev, err := s.devStore.MergeDevice(
		mir_v1.DeviceTarget{
			Ids: []string{deviceId},
		},
		jsonRaw,
		mng.MergePatch,
	)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error updating device properties in store")
		deviceReportedPropsRequestErrorTotal.Inc()
		return
	}
	if len(dev) > 0 {
		if err = publishReportedPropertiesEvent(s.m, msg, dev[0].Meta.Name, dev[0].Meta.Namespace, deviceId, dev[0].Properties.Reported); err != nil {
			l.Error().Err(err).Msg("error while publishing device reported properties event")
		}
	}
}

func (s *ProtoCfgServer) desiredPropsSub(msg *mir.Msg, deviceId string) (*device_apiv1.ReportedProperties, error) {
	deviceDesiredPropsRequestTotal.Inc()
	devs, err := s.devStore.ListDevice(
		mir_v1.DeviceTarget{
			Ids: []string{deviceId},
		}, false)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error listing device from db")
		deviceDesiredPropsRequestErrorTotal.Inc()
		return &device_apiv1.ReportedProperties{}, err
	}
	if len(devs) == 0 {
		l.Error().Str("device_id", deviceId).Msg("device not found in store")
		deviceDesiredPropsRequestErrorTotal.Inc()
		return &device_apiv1.ReportedProperties{}, mng.ErrorNoDeviceFound
	}
	dev := devs[0]

	devSch, _, err := s.schStore.GetDeviceSchema(deviceId, false)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error retrieving device schema")
		deviceDesiredPropsRequestErrorTotal.Inc()
		return &device_apiv1.ReportedProperties{}, err
	}

	desiredProps := &device_apiv1.ReportedProperties{
		Encoding:   common_apiv1.Encoding_ENCODING_PROTOBUF,
		Properties: make(map[string]*device_apiv1.Properties),
	}

	cfgDescs, err := devSch.GetConfigList(nil)
	if err != nil {
		l.Error().Str("device_id", deviceId).Err(err).Msg("error getting config list")
		deviceDesiredPropsRequestErrorTotal.Inc()
		return &device_apiv1.ReportedProperties{}, err
	}

	// List all config in schema
	// If not written in db, means we need to write empty config
	// Then we return the config to the device
	missingCfg := make(map[string]any)
	missingTime := make(map[string]time.Time)
	for _, cfgDesc := range cfgDescs {
		var jsonRaw []byte
		var updTime string
		var desc protoreflect.Descriptor
		desc, devSch, err = s.schStore.FindMessageDescriptor(deviceId, devSch, cfgDesc.Name)
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", cfgDesc.Name).Msg("error finding descriptor in device schema")
			deviceDesiredPropsRequestErrorTotal.Inc()
			continue
		}

		if p, ok := dev.Properties.Desired[cfgDesc.Name]; ok {
			// Mean we have a config already for this
			props := p.(map[string]interface{})
			updTime = dev.Status.Properties.Desired[cfgDesc.Name].Format(time.RFC3339Nano)

			jsonRaw, err = json.Marshal(props)
			if err != nil {
				l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", cfgDesc.Name).Msg("error marshaling desired properties to JSON")
				deviceDesiredPropsRequestErrorTotal.Inc()
				continue
			}

		} else {
			jsonRaw, err = json_template.GenerateTemplate(
				desc.(protoreflect.MessageDescriptor),
				json_template.Options{
					WithoutMapExample:   true,
					WithoutArrayExample: true,
				},
			)
			if err != nil {
				l.Error().Err(err).Str("device_id", deviceId).Msg("error marshaling default reported properties to JSON")
				deviceDesiredPropsRequestErrorTotal.Inc()
				continue
			}

			msgMap := make(map[string]interface{})
			if err = json.Unmarshal(jsonRaw, &msgMap); err != nil {
				l.Error().Err(err).Str("device_id", deviceId).Msg("error unmarshaling reported properties to map")
				deviceDesiredPropsRequestErrorTotal.Inc()
				continue
			}

			timeNow := time.Now().UTC()
			updTime = timeNow.Format(time.RFC3339Nano)
			missingCfg[cfgDesc.Name] = msgMap
			missingTime[cfgDesc.Name] = timeNow
		}

		msg := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
		if err = protojson.Unmarshal(jsonRaw, msg); err != nil {
			deviceDesiredPropsRequestErrorTotal.Inc()
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", cfgDesc.Name).Msg("error unmarshaling desired properties")
			continue
		}
		protoRaw, err := proto.Marshal(msg)
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", cfgDesc.Name).Msg("error marshaling desired properties to protobuf")
			deviceDesiredPropsRequestErrorTotal.Inc()
			continue
		}
		desiredProps.Properties[cfgDesc.Name] = &device_apiv1.Properties{
			Time:     updTime,
			Property: protoRaw,
		}
	}

	if len(missingCfg) > 0 {
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Msg("error marshalling properties to pbstruct")
			deviceDesiredPropsRequestErrorTotal.Inc()
		} else {
			d := mir_v1.NewDevice().WithProps(mir_v1.DeviceProperties{
				Desired: missingCfg,
			}).WithStatus(mir_v1.DeviceStatus{
				Properties: mir_v1.PropertiesTime{
					Desired: missingTime,
				},
			})
			devs, err := s.devStore.UpdateDevice(mir_v1.DeviceTarget{
				Ids: []string{deviceId},
			}, d)
			if err != nil {
				l.Error().Err(err).Str("device_id", deviceId).Msg("error updating device properties in store")
				deviceDesiredPropsRequestErrorTotal.Inc()
			} else {
				if len(devs) > 0 {
					dev = devs[0]
				}
			}
		}
	}

	return desiredProps, nil
}

func minifyJSON(input []byte) ([]byte, error) {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, input); err != nil {
		return []byte{}, err
	}
	return buffer.Bytes(), nil
}

func publishDesiredPropertiesEvent(m *mir.Mir, msg *mir.Msg, name, namespace, deviceId string, props map[string]any) error {
	payload, err := json.Marshal(props)
	if err != nil {
		return err
	}
	return m.Event().Publish(mir.NewEventSubjectString(cfg_client.DesiredPropertiesEvent.WithId(deviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceDesiredProps",
			Message: "Device desired properties updated successfully",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      name,
				Namespace: namespace,
			}).Object,
		}, msg)
}

func publishReportedPropertiesEvent(m *mir.Mir, msg *mir.Msg, name, namespace, deviceId string, props map[string]any) error {
	payload, err := json.Marshal(props)
	if err != nil {
		return err
	}
	return m.Event().Publish(mir.NewEventSubjectString(cfg_client.ReportedPropertiesEvent.WithId(deviceId)),
		mir_v1.EventSpec{
			Type:    mir_v1.EventTypeNormal,
			Reason:  "DeviceReportedProps",
			Message: "Device reported properties updated successfully",
			Payload: payload,
			RelatedObject: mir_v1.NewDevice().WithMeta(mir_v1.Meta{
				Name:      name,
				Namespace: namespace,
			}).Object,
		}, msg)
}
