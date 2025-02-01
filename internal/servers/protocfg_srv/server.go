package protocfg_srv

import (
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
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
)

type ProtoCfgServer struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	wg        *sync.WaitGroup
	m         *mir.Mir
	devStore  mng.DeviceStore
	schStore  *schema_cache.MirProtoCache
}

const (
	ServiceName = "mir_protocfg"
)

var (
	devMirErrType = devicev1.Error{}
	devMirErrStr  = string(devMirErrType.ProtoReflect().Descriptor().FullName())
)

var requestCount = metrics.NewCounterVec(prometheus.CounterOpts{
	Name: "request_count",
	Help: "Number of request for core",
}, []string{"route"})

var (
	l            zerolog.Logger
	offlineAfter = time.Second * 30
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(requestCount)
}

func NewProtoCfg(logger zerolog.Logger, m *mir.Mir, store mng.DeviceStore) (*ProtoCfgServer, error) {
	l = logger.With().Str("srv", "protocfg_server").Logger()
	cc, err := schema_cache.NewMirProtoCache(l, m)
	if err != nil {
		return nil, err
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	return &ProtoCfgServer{
		ctx:       ctx,
		cancelCtx: cancelFn,
		wg:        &sync.WaitGroup{},
		m:         m,
		devStore:  store,
		schStore:  cc,
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
	// 1. get device list
	// 2. for each device, get stored schema, if empty, fetch from device
	// 3. return list of config

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
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
		for _, cmd := range cfgs {
			cfgList = append(cfgList, cmd)
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

	errs := []string{}
	if req.Targets == nil ||
		len(req.Targets.Ids) == 0 &&
			len(req.Targets.Names) == 0 &&
			len(req.Targets.Namespaces) == 0 &&
			len(req.Targets.Labels) == 0 {
		errs = append(errs, mir_models.ErrorNoDeviceTargetProvided.Error())
	}
	if req.Name == "" {
		errs = append(errs, mir_models.ErrorCommandNameNotProvided.Error())
	}
	if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_UNSPECIFIED && !req.ShowTemplate {
		errs = append(errs, mir_models.ErrorCommandEncodingNotSpecified.Error())
	}
	if (req.Payload == nil || len(req.Payload) == 0) && !req.ShowTemplate && req.PayloadEncoding != common_apiv1.Encoding_ENCODING_PROTOBUF {
		// Proto encoding can be empty if struct is empty, not json
		errs = append(errs, mir_models.ErrorCommandPayloadNotProvided.Error())
	}
	if len(errs) > 0 {
		l.Error().Err(fmt.Errorf("%w: %s", mir_models.ErrorBadRequest, strings.Join(errs, ", "))).Msg("")
		return nil, fmt.Errorf("%w: %s", mir_models.ErrorBadRequest, errs)
	}
	// If command was specified with labels
	if index := strings.Index(req.Name, "{"); index != -1 {
		req.Name = req.Name[:index]
	}

	resp, err := s.sendConfigToDevices(req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while processing send config request")
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

// TODO
// - [x] Always validate and cast to json since we need to write the payload in the db in JSON format
// - [x] Send to device in Proto, but can receive proto or json
// - [x] Write Json to db, send proto to device
// - [x] DeviceSDK for config (send and receive)
// - [x] Have a timestamp for last updated desired properties. Used to compared if event is old on device on bootup
// - [x] Reported properties
// - [ ] Update from core
// - [x] Device desired properties multiple handler for same cfg
// - [x] More test

func (s *ProtoCfgServer) sendConfigToDevices(req *cfg_apiv1.SendConfigRequest) (map[string]*cfg_apiv1.SendConfigResponse_ConfigResponse, error) {
	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
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
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
				Error:    errors.Wrap(err, "error retrieve config descriptor from device schema").Error(),
			}
			devInError = true
			continue
		}

		if req.ShowTemplate {
			tpl, err := json_template.GenerateTemplate(msgReqDesc.(protoreflect.MessageDescriptor))
			if err != nil {
				l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error generating config template from device schema")
				devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error generating config template from device schema").Error(),
				}
			}
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Name:     req.Name,
				Payload:  tpl,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS,
			}
			continue
		}

		payloadReq := dynamicpb.NewMessage(msgReqDesc.(protoreflect.MessageDescriptor))

		// Encoding and validation
		err = nil
		bytePayload := req.Payload
		jsonPayload := req.Payload
		mapPayload := make(map[string]interface{})
		if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
			// The payload is already in JSON
			// Encoding to proto and then serialize
			err = protojson.Unmarshal(req.Payload, payloadReq)
			if err == nil {
				bytePayload, err = proto.Marshal(payloadReq)
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
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
				Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
			}
			devInError = true
			continue
		}

		// Add to map with time and name
		timeNow := time.Now().UTC()
		mapPayload["__time"] = timeNow.Format(time.RFC3339)
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
			payload:    bytePayload,
			mapPayload: mapPayload,
			deviceId:   dev.Spec.DeviceId,
			msgDesc:    msgReqDesc.(protoreflect.MessageDescriptor),
		}

		l.Debug().Str("device_id", dev.Spec.DeviceId).Msgf("config %s validated for device %s", req.Name, dev.GetNameNamespace())
	}

	// If we have validation error or it is a dry run, we return the request
	// If ForcePush is on, means we send the command to each validated devices
	// even if some errors
	if (devInError && !req.ForcePush) || req.DryRun {
		l.Info().Bool("device_in_error", devInError).Bool("force_push", req.ForcePush).Bool("dry_run", req.DryRun).Msgf("config processed but not sent")
		// Events
		// TODO not sure this should be there. same in cmd
		// for _, cfgResp := range devResp {
		// 	if err := cfg_client.PublishDeviceConfigEvent(s.m.Bus, "protocfg", cfgResp.DeviceId, cfgResp); err != nil {
		// 		l.Error().Err(err).Msg("error while publishing device config event")
		// 	}
		// }
		return devResp, nil
	}

	// Sends configs
	wg := &sync.WaitGroup{}
	wg.Add(len(configToSend))

	l.Info().Msgf("sending config %s to targeted devices", req.Name)
	for nameNs, p := range configToSend {
		go func() {
			defer wg.Done()

			// TODO write to db first
			// Write directly to store, not sdk
			// Event for desired properties update
			// Event for reported properties update
			props, err := structpb.NewStruct(p.mapPayload)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error marshalling properties to pbstruct")
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error marshalling properties to pbstruct").Error(),
				}
				return
			}
			// TODO return updated device from store
			dev, err := s.devStore.UpdateDevice(&core_apiv1.UpdateDeviceRequest{
				Targets: &core_apiv1.Targets{
					Ids: []string{p.deviceId},
				},
				Props: &core_apiv1.UpdateDeviceRequest_Properties{
					Desired: props,
				},
			})
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error updating device properties in store")
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error updating device properties in store").Error(),
				}
				return
			}

			// Event
			if err := cfg_client.PublishDesiredPropertiesEvent(s.m.Bus, "protocfg", p.deviceId, dev[0].Properties.Desired); err != nil {
				l.Error().Err(err).Msg("error while publishing device config event")
			}

			err = s.m.Device().Config().PublishRaw(p.deviceId, mir.ProtoCmdDesc{
				Name:    req.Name,
				Payload: p.payload,
			}, p.time)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error during sent config request to device")
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error during sent config request to device").Error(),
				}
				return
			}

			dt := dev[0].Properties.Desired[req.Name].(map[string]interface{})
			delete(dt, "__time")
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
	msgDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(deviceId, msgName, false)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error retrieving config descriptor from device schema")
		return
	}

	msgProto := dynamicpb.NewMessage(msgDesc.(protoreflect.MessageDescriptor))
	if err = proto.Unmarshal(data, msgProto); err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error unmarshaling reported properties")
		return
	}

	msgJson, err := protojson.Marshal(msgProto)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error marshaling reported properties to JSON")
		return
	}
	msgMap := make(map[string]interface{})
	if err = json.Unmarshal(msgJson, &msgMap); err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error unmarshaling reported properties to map")
		return
	}
	timeNow := time.Now().UTC()
	msgMap["__time"] = timeNow.Format(time.RFC3339)
	msgMap = map[string]interface{}{
		"properties": map[string]interface{}{
			"reported": map[string]interface{}{
				string(msgDesc.FullName()): msgMap,
			},
		},
	}

	jsonRaw, err := json.Marshal(msgMap)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error marshaling reported properties to JSON")
		return
	}

	dev, err := s.devStore.MergeDevice(
		&core_apiv1.Targets{
			Ids: []string{deviceId},
		},
		jsonRaw,
		mng.MergePatch,
	)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error updating device properties in store")
		return
	}
	if len(dev) > 0 {
		if err = cfg_client.PublishReportedPropertiesEvent(s.m.Bus, "protocfg", deviceId, dev[0].Properties.Reported); err != nil {
			l.Error().Err(err).Msg("error while publishing device reported properties event")
		}
	}
}

func (s *ProtoCfgServer) desiredPropsSub(msg *mir.Msg, deviceId string) (*device_apiv1.ReportedProperties, error) {
	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{
		Targets: &core_apiv1.Targets{
			Ids: []string{deviceId},
		},
	})
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error listing device from db")
		return &device_apiv1.ReportedProperties{}, err
	}
	if len(devs) == 0 {
		l.Error().Str("device_id", deviceId).Msg("device not found in store")
		return &device_apiv1.ReportedProperties{}, mng.ErrorNoDeviceFound
	}
	dev := devs[0]

	devSch, _, err := s.schStore.GetDeviceSchema(deviceId, false)
	if err != nil {
		l.Error().Err(err).Str("device_id", deviceId).Msg("error retrieving device schema")
		return &device_apiv1.ReportedProperties{}, err
	}

	// Get the desired properties
	desiredProps := &device_apiv1.ReportedProperties{
		Encoding:   common_apiv1.Encoding_ENCODING_PROTOBUF,
		Properties: make(map[string]*device_apiv1.Properties),
	}
	for msgName, p := range dev.Properties.Desired {
		var desc protoreflect.Descriptor
		desc, devSch, err = s.schStore.FindMessageDescriptor(deviceId, devSch, msgName)
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", msgName).Msg("error finding descriptor in device schema")
			continue
		}
		props := p.(map[string]interface{})
		updTime := props["__time"].(string)
		delete(props, "__time")

		propsJsonByte, err := json.Marshal(props)
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", msgName).Msg("error marshaling desired properties to JSON")
			continue
		}

		msg := dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
		if err = protojson.Unmarshal(propsJsonByte, msg); err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", msgName).Msg("error unmarshaling desired properties")
		}
		msgBytes, err := proto.Marshal(msg)
		if err != nil {
			l.Error().Err(err).Str("device_id", deviceId).Str("msg_name", msgName).Msg("error marshaling desired properties to protobuf")
			continue
		}
		desiredProps.Properties[msgName] = &device_apiv1.Properties{
			Time:     updTime,
			Property: msgBytes,
		}

	}
	return desiredProps, nil
}
