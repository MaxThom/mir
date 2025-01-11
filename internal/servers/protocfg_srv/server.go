package protocfg_srv

import (
	"context"
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
	deviceId string
	payload  []byte
}

func (s *ProtoCfgServer) sendConfigToDevices(req *cfg_apiv1.SendConfigRequest) (map[string]*cfg_apiv1.SendConfigResponse_ConfigResponse, error) {
	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		return nil, err
	} else if len(devs) == 0 {
		return nil, mng.ErrorNoDeviceFound
	}
	devResp := make(map[string]*cfg_apiv1.SendConfigResponse_ConfigResponse)

	// We do validation if NoValidation is false or if encoding is JSON
	// This fills configToSend with payload for validated devices
	// It also fills devResp with devices in error
	// If not, then we just put the request payload directly for the devices
	configToSend := make(map[string]*cmdDevicePayload)
	devInError := false
	if !req.NoValidation || req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
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
			if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				err = protojson.Unmarshal(req.Payload, payloadReq)
				if err == nil {
					bytePayload, err = proto.Marshal(payloadReq)
				}
			} else {
				err = proto.Unmarshal(req.Payload, payloadReq)
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

			// Prepare
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_VALIDATED,
			}
			configToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: bytePayload, deviceId: dev.Spec.DeviceId}
			l.Debug().Str("device_id", dev.Spec.DeviceId).Msgf("config %s validated for device %s", req.Name, dev.GetNameNamespace())
		}
	} else {
		for _, dev := range devs {
			devResp[dev.GetNameNamespace()] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_PENDING,
			}
			configToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: req.Payload, deviceId: dev.Spec.DeviceId}
		}
	}

	// If we have validation error or it is a dry run, we return the request
	// If ForcePush is on, means we send the command to each validated devices
	// even if some errors
	if (devInError && !req.ForcePush) || req.DryRun {
		l.Info().Bool("device_in_error", devInError).Bool("force_push", req.ForcePush).Bool("dry_run", req.DryRun).Msgf("config processed but not sent")
		// Events
		// TODO not sure this should be there. same in cmd
		for _, cfgResp := range devResp {
			if err := cfg_client.PublishDeviceConfigEvent(s.m.Bus, "protocfg", cfgResp.DeviceId, cfgResp); err != nil {
				l.Error().Err(err).Msg("error while publishing device config event")
			}
		}
		return devResp, nil
	}

	// Sends configs
	wg := &sync.WaitGroup{}
	wg.Add(len(configToSend))

	timeout := 10 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	l.Info().Msgf("sending config %s to targeted devices", req.Name)
	for nameNs, p := range configToSend {
		go func() {
			defer wg.Done()

			cmdResp, err := s.m.Device().Command().RequestRaw(p.deviceId, mir.ProtoCmdDesc{
				Name:    req.Name,
				Payload: p.payload,
			}, timeout)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error during sent config request to device")
				devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
					DeviceId: p.deviceId,
					Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error during sent config request to device").Error(),
				}
				return
			}

			l.Debug().Str("device_id", p.deviceId).Msgf("received config response from device")
			respPayload := cmdResp.Payload
			if cmdResp.Name == devMirErrStr {
				var errPl devicev1.Error
				if err = proto.Unmarshal(respPayload, &errPl); err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error unmarshaling error payload")
					devResp[nameNs].Error = fmt.Errorf("error unmarshaling error payload: %w", err).Error()
				} else {
					devResp[nameNs].Error = errPl.GetMessage()
				}
				devResp[nameNs].Status = cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR
				devResp[nameNs].DeviceId = p.deviceId
				return
			}

			if !req.NoValidation || req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				msgRespDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(p.deviceId, cmdResp.Name, false)
				if err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error finding config response in schema")
					devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
						DeviceId: p.deviceId,
						Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error finding config response in schema").Error(),
					}
					return
				}
				msgResp := dynamicpb.NewMessage(msgRespDesc.(protoreflect.MessageDescriptor))
				err = proto.Unmarshal(respPayload, msgResp)
				if err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error unmarshaling payload of config reponse")
					devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
						DeviceId: p.deviceId,
						Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
					}
					return
				}
				if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
					respPayload, err = protojson.Marshal(msgResp)
					if err != nil {
						l.Error().Err(err).Str("device_id", p.deviceId).Msg("error marshaling proto payload to json")
						devResp[nameNs] = &cfg_apiv1.SendConfigResponse_ConfigResponse{
							DeviceId: p.deviceId,
							Status:   cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_ERROR,
							Error:    errors.Wrap(err, "error marshaling proto payload to json").Error(),
						}
						return
					}
				}
			}
			devResp[nameNs].DeviceId = p.deviceId
			devResp[nameNs].Status = cfg_apiv1.ConfigResponseStatus_CONFIG_RESPONSE_STATUS_SUCCESS
			devResp[nameNs].Name = cmdResp.Name
			devResp[nameNs].Payload = respPayload
		}()
	}
	wg.Wait()

	// Event
	for _, cmdResp := range devResp {
		if err := cfg_client.PublishDeviceConfigEvent(s.m.Bus, "protocmd", cmdResp.DeviceId, cmdResp); err != nil {
			l.Error().Err(err).Msg("error while publishing device config event")
		}
	}

	return devResp, nil
}
