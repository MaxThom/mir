package protocmd_srv

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/clients/cmd_client"
	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	"github.com/maxthom/mir/internal/libs/proto/json_template"
	"github.com/maxthom/mir/internal/services/schema_cache"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mirv2"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// IDEA possible perf improvement, cache of descriptor
type ProtoCmdServer struct {
	sub      *nats.Subscription
	m        *mirv2.Mir
	devStore mng.DeviceStore
	schStore *schema_cache.MirProtoCache
}

// TODO prom metics
const (
	ServiceName = "mir_command"
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
var (
	devMirErrType = devicev1.Error{}
	devMirErrStr  = string(devMirErrType.ProtoReflect().Descriptor().FullName())
)

func RegisterMetrics(reg prometheus.Registerer) {
	reg.Register(uploadMetric)
	reg.Register(datapointCount)
}

func NewProtoCmdServer(logger zerolog.Logger, m *mirv2.Mir, devStore mng.DeviceStore) (*ProtoCmdServer, error) {
	l = logger.With().Str("srv", "protocmd_server").Logger()
	cc, err := schema_cache.NewMirProtoCache(l, m)
	if err != nil {
		return nil, err
	}
	return &ProtoCmdServer{
		m:        m,
		devStore: devStore,
		schStore: cc,
	}, nil
}

func (s *ProtoCmdServer) Listen(ctx context.Context) {
	s.m.Server().SendCommand().QueueSubscribe(ServiceName, s.sendCommandSub)
	s.m.Server().ListCommands().QueueSubscribe(ServiceName, s.listCommandsSub)
}

func (s *ProtoCmdServer) sendCommandSub(msg *nats.Msg, clientId string, req *cmd_apiv1.SendCommandRequest) (*cmd_apiv1.SendCommandResponse_CommandResponses, error) {
	l.Info().Any("req", req).Msg("send command request")

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

	resp, err := s.sendCommandToDevices(req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while processing send command request")
		return nil, fmt.Errorf("error sending command to devices: %w", err)
	}

	l.Info().Msg("send command request processed successfully")
	return &cmd_apiv1.SendCommandResponse_CommandResponses{
		DeviceResponses: resp,
		Encoding:        req.PayloadEncoding,
	}, nil
}

type cmdDevicePayload struct {
	deviceId string
	payload  []byte
}

func (s *ProtoCmdServer) sendCommandToDevices(req *cmd_apiv1.SendCommandRequest) (map[string]*cmd_apiv1.SendCommandResponse_CommandResponse, error) {
	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		return nil, err
	} else if len(devs) == 0 {
		return nil, mng.ErrorNoDeviceFound
	}
	devResp := make(map[string]*cmd_apiv1.SendCommandResponse_CommandResponse)

	// We do validation if NoValidation is false or if encoding is JSON
	// This fills commandsToSend with payload for validated devices
	// It also fills devResp with devices in error
	// If not, then we just put the request payload directly for the devices
	commandsToSend := make(map[string]*cmdDevicePayload)
	devInError := false
	if !req.NoValidation || req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
		for _, dev := range devs {
			// Retrieve descriptor
			msgReqDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(dev.Spec.DeviceId, req.Name, req.RefreshSchema)
			if err != nil {
				l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error retrieving command descriptor from device schema")
				devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error retrieve command descriptor from device schema").Error(),
				}
				devInError = true
				continue
			}

			if req.ShowTemplate {
				tpl, err := json_template.GenerateTemplate(msgReqDesc.(protoreflect.MessageDescriptor))
				if err != nil {
					l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error generating command template from device schema")
					devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						DeviceId: dev.Spec.DeviceId,
						Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error generating command template from device schema").Error(),
					}
				}
				devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					DeviceId: dev.Spec.DeviceId,
					Name:     req.Name,
					Payload:  tpl,
					Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS,
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
				devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					DeviceId: dev.Spec.DeviceId,
					Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
				}
				devInError = true
				continue
			}

			// Prepare
			devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_VALIDATED,
			}
			commandsToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: bytePayload, deviceId: dev.Spec.DeviceId}
			l.Debug().Str("device_id", dev.Spec.DeviceId).Msgf("command %s validated for device %s", req.Name, dev.GetNameNamespace())
		}
	} else {
		for _, dev := range devs {
			devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
				DeviceId: dev.Spec.DeviceId,
				Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_PENDING,
			}
			commandsToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: req.Payload, deviceId: dev.Spec.DeviceId}
		}
	}

	// If we have validation error or it is a dry run, we return the request
	// If ForcePush is on, means we send the command to each validated devices
	// even if some errors
	if (devInError && !req.ForcePush) || req.DryRun {
		l.Info().Bool("device_in_error", devInError).Bool("force_push", req.ForcePush).Bool("dry_run", req.DryRun).Msgf("commands processed but not sent")
		// Events
		for _, cmdResp := range devResp {
			if err := cmd_client.PublishDeviceCommandEvent(s.m.Bus, "protocmd", cmdResp.DeviceId, cmdResp); err != nil {
				l.Error().Err(err).Msg("error while publishing device command event")
			}
		}
		return devResp, nil
	}

	// Sends commands
	wg := &sync.WaitGroup{}
	wg.Add(len(commandsToSend))

	timeout := 10 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	l.Info().Msgf("sending command %s to targeted devices", req.Name)
	for nameNs, p := range commandsToSend {
		go func() {
			defer wg.Done()

			cmdResp, err := s.m.Device().Command().RequestRaw(p.deviceId, mirv2.ProtoCmdDesc{
				Name:    req.Name,
				Payload: p.payload,
			}, timeout)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error during sent command request to device")
				devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					DeviceId: p.deviceId,
					Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error during sent command request to device").Error(),
				}
				return
			}

			l.Debug().Str("device_id", p.deviceId).Msgf("received command response from device")
			respPayload := cmdResp.Payload
			if cmdResp.Name == devMirErrStr {
				var errPl devicev1.Error
				if err = proto.Unmarshal(respPayload, &errPl); err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error unmarshaling error payload")
					devResp[nameNs].Error = fmt.Errorf("error unmarshaling error payload: %w", err).Error()
				} else {
					devResp[nameNs].Error = errPl.GetMessage()
				}
				devResp[nameNs].Status = cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR
				devResp[nameNs].DeviceId = p.deviceId
				return
			}

			if !req.NoValidation || req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				msgRespDesc, _, _, err := s.schStore.GetDeviceSchemaAndDescriptor(p.deviceId, cmdResp.Name, false)
				if err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error finding command response in schema")
					devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						DeviceId: p.deviceId,
						Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error finding command response in schema").Error(),
					}
					return
				}
				msgResp := dynamicpb.NewMessage(msgRespDesc.(protoreflect.MessageDescriptor))
				err = proto.Unmarshal(respPayload, msgResp)
				if err != nil {
					l.Error().Err(err).Str("device_id", p.deviceId).Msg("error unmarshaling payload of command reponse")
					devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						DeviceId: p.deviceId,
						Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:    errors.Wrap(err, "error unmarshaling payload").Error(),
					}
					return
				}
				if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
					respPayload, err = protojson.Marshal(msgResp)
					if err != nil {
						l.Error().Err(err).Str("device_id", p.deviceId).Msg("error marshaling proto payload to json")
						devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
							DeviceId: p.deviceId,
							Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
							Error:    errors.Wrap(err, "error marshaling proto payload to json").Error(),
						}
						return
					}
				}
			}
			devResp[nameNs].DeviceId = p.deviceId
			devResp[nameNs].Status = cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS
			devResp[nameNs].Name = cmdResp.Name
			devResp[nameNs].Payload = respPayload
		}()
	}
	wg.Wait()

	// Event
	for _, cmdResp := range devResp {
		if err := cmd_client.PublishDeviceCommandEvent(s.m.Bus, "protocmd", cmdResp.DeviceId, cmdResp); err != nil {
			l.Error().Err(err).Msg("error while publishing device command event")
		}
	}

	return devResp, nil
}

func (s *ProtoCmdServer) listCommandsSub(msg *nats.Msg, clientId string, req *cmd_apiv1.SendListCommandsRequest) (map[string]*cmd_apiv1.Commands, error) {
	l.Info().Any("req", req).Msg("list command request")
	// 1. get device list
	// 2. for each device, get stored schema, if empty, fetch from device
	// 3. return list of commands

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
		return nil, fmt.Errorf("error listing devices from db: %w", err)
	}

	devsCmds := make(map[string]*cmd_apiv1.Commands)
	for _, dev := range devs {
		reg, _, err := s.schStore.GetDeviceSchema(dev.Spec.DeviceId, req.RefreshSchema)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cmd_apiv1.Commands{
				Error: err.Error(),
			}
			continue
		}

		cmds, err := reg.GetCommandsList(req.FilterLabels)
		if err != nil {
			devsCmds[dev.GetNameNamespace()] = &cmd_apiv1.Commands{
				Error: err.Error(),
			}
			continue
		}

		cmdsList := []*cmd_apiv1.CommandDescriptor{}
		for _, cmd := range cmds {
			cmdsList = append(cmdsList, cmd)
		}
		devsCmds[dev.GetNameNamespace()] = &cmd_apiv1.Commands{
			Commands: cmdsList,
		}
	}

	l.Info().Msg("list command request processed successfully")
	return devsCmds, nil
}
