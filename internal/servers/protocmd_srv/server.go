package protocmd_srv

import (
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
	"github.com/maxthom/mir/pkgs/module/mir"
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
	m        *mir.Mir
	devStore mng.MirStore
	schStore *schema_cache.MirProtoCache
}

// TODO prom metics
const (
	ServiceName = "mir_command"
)

var (
	requestTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "cmd",
		Name:      "request_total",
		Help:      "Number of request for commands routes",
	}, []string{"route"})
	requestErrorTotal = metrics.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "cmd",
		Name:      "request_error_total",
		Help:      "Number of error request for commands routes",
	}, []string{"route"})
	deviceCmdSentTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cmd",
		Name:      "device_cmd_sent_total",
		Help:      "Number of commands sent to devices",
	})
	deviceCmdSentErrorTotal = metrics.NewCounter(prometheus.CounterOpts{
		Subsystem: "cmd",
		Name:      "device_cmd_sent_error_total",
		Help:      "Number of commands failed to sent to devices",
	})

	devMirErrType = devicev1.Error{}
	devMirErrStr  = string(devMirErrType.ProtoReflect().Descriptor().FullName())

	l zerolog.Logger
)

func init() {
	requestTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestTotal.With(prometheus.Labels{"route": "send"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "list"}).Add(0)
	requestErrorTotal.With(prometheus.Labels{"route": "send"}).Add(0)
}

func NewProtoCmd(logger zerolog.Logger, m *mir.Mir, devStore mng.MirStore, schemaCache *schema_cache.MirProtoCache) (*ProtoCmdServer, error) {
	l = logger.With().Str("srv", "protocmd_server").Logger()
	return &ProtoCmdServer{
		m:        m,
		devStore: devStore,
		schStore: schemaCache,
	}, nil
}

func (s *ProtoCmdServer) Serve() error {
	if err := s.m.Server().SendCommand().QueueSubscribe(ServiceName, s.sendCommandSub); err != nil {
		return err
	}
	if err := s.m.Server().ListCommands().QueueSubscribe(ServiceName, s.listCommandsSub); err != nil {
		return err
	}
	return nil
}

func (s *ProtoCmdServer) Shutdown() error {
	return nil
}

func (s *ProtoCmdServer) sendCommandSub(msg *mir.Msg, clientId string, req *cmd_apiv1.SendCommandRequest) (*cmd_apiv1.SendCommandResponse_CommandResponses, error) {
	l.Info().Any("req", req).Msg("send command request")
	requestTotal.WithLabelValues("send").Inc()

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
		requestErrorTotal.WithLabelValues("send").Inc()
		return nil, fmt.Errorf("%w: %s", mir_models.ErrorBadRequest, errs)
	}
	// If command was specified with labels
	if index := strings.Index(req.Name, "{"); index != -1 {
		req.Name = req.Name[:index]
	}

	resp, err := s.sendCommandToDevices(msg, req)
	if err != nil {
		l.Error().Err(err).Msg("error occure while processing send command request")
		requestErrorTotal.WithLabelValues("send").Inc()
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

func (s *ProtoCmdServer) sendCommandToDevices(msg *mir.Msg, req *cmd_apiv1.SendCommandRequest) (map[string]*cmd_apiv1.SendCommandResponse_CommandResponse, error) {
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
				deviceCmdSentErrorTotal.Inc()
				continue
			}

			if req.ShowTemplate {
				tpl, err := json_template.GenerateTemplate(msgReqDesc.(protoreflect.MessageDescriptor), json_template.Options{})
				if err != nil {
					l.Error().Err(err).Str("device_id", dev.Spec.DeviceId).Msg("error generating command template from device schema")
					deviceCmdSentErrorTotal.Inc()
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
				deviceCmdSentErrorTotal.Inc()
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

			cmdResp, err := s.m.Device().Command().RequestRaw(p.deviceId, mir.ProtoCmdDesc{
				Name:    req.Name,
				Payload: p.payload,
			}, timeout)
			if err != nil {
				l.Error().Err(err).Str("device_id", p.deviceId).Msg("error during sent command request to device")
				deviceCmdSentErrorTotal.Inc()
				devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					DeviceId: p.deviceId,
					Status:   cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:    errors.Wrap(err, "error during sent command request to device").Error(),
				}
				return
			}
			deviceCmdSentTotal.Inc()

			l.Debug().Str("device_id", p.deviceId).Msgf("received command response from device")
			respPayload := cmdResp.Payload
			if cmdResp.Name == devMirErrStr {
				var errPl devicev1.Error
				if err = proto.Unmarshal(respPayload, &errPl); err != nil {
					deviceCmdSentErrorTotal.Inc()
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
					deviceCmdSentErrorTotal.Inc()
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
					deviceCmdSentErrorTotal.Inc()
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
						deviceCmdSentErrorTotal.Inc()
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
	if len(commandsToSend) > 0 {
		for nameNs, cmdResp := range devResp {
			nns := strings.Split(nameNs, "/")
			if err = publishCommandEvent(s.m, msg, nns[0], nns[1], cmdResp); err != nil {
				l.Warn().Err(err).Msg("error while publishing device command event")
			}
		}
	}

	return devResp, nil
}

func (s *ProtoCmdServer) listCommandsSub(msg *mir.Msg, clientId string, req *cmd_apiv1.SendListCommandsRequest) (map[string]*cmd_apiv1.Commands, error) {
	l.Info().Any("req", req).Msg("list command request")
	requestTotal.WithLabelValues("list").Inc()
	// 1. get device list
	// 2. for each device, get stored schema, if empty, fetch from device
	// 3. return list of commands

	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		l.Error().Err(err).Msg("error occure while listing devices")
		requestErrorTotal.WithLabelValues("list").Inc()
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

func publishCommandEvent(m *mir.Mir, msg *mir.Msg, name, namespace string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse) error {
	payload, err := mir_models.StructToMapAny(cmd)
	if err != nil {
		return err
	}
	return m.Event().Publish(mir.NewEventSubjectString(cmd_client.DeviceCommandEvent.WithId(cmd.DeviceId)),
		mir_models.EventSpec{
			Type:    mir_models.EventTypeNormal,
			Reason:  "DeviceCommand",
			Message: "Device command executed successfully",
			Payload: payload,
			RelatedObject: mir_models.NewEvent().WithMeta(mir_models.Meta{
				Name:      name,
				Namespace: namespace,
			}).Object,
		}, msg)
}
