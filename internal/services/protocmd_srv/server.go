package protocmd_srv

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	"github.com/maxthom/mir/internal/libs/api/metrics"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/mir_utils"
	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	common_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/common_api"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
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
	sub            *nats.Subscription
	m              *mir.Mir
	devStore       mng.DeviceStore
	devSchemas     map[string]*mir_utils.MirProtoSchema
	devSchemasLock sync.RWMutex
}

// TODO prom metics

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

func NewProtoCmdServer(logger zerolog.Logger, m *mir.Mir, devStore mng.DeviceStore) *ProtoCmdServer {
	l = logger.With().Str("srv", "protoflux_server").Logger()
	return &ProtoCmdServer{
		m:          m,
		devStore:   devStore,
		devSchemas: make(map[string]*mir_utils.MirProtoSchema),
	}
}

func (s *ProtoCmdServer) Listen(ctx context.Context) {
	s.m.Subscribe(mir.Client().V1Alpha().SendCommand(s.sendCommandSub()))
	s.m.Subscribe(mir.Client().V1Alpha().ListCommands(s.listCommandsSub()))
}

func (s *ProtoCmdServer) sendCommandSub() func(msg *nats.Msg, req *cmd_apiv1.SendCommandRequest, e error) {
	return func(msg *nats.Msg, req *cmd_apiv1.SendCommandRequest, e error) {
		if e != nil {
			l.Error().Err(e).Msg("error occure while receiving request")
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
				Response: &cmd_apiv1.SendCommandResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", e.Error()},
					},
				},
			})
		}

		if req.Targets == nil ||
			len(req.Targets.Ids) == 0 &&
				len(req.Targets.Names) == 0 &&
				len(req.Targets.Namespaces) == 0 &&
				len(req.Targets.Labels) == 0 &&
				len(req.Targets.Annotations) == 0 {
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
				Response: &cmd_apiv1.SendCommandResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorNoDeviceTargetProvided.Error(),
						Details: []string{"400 Bad Request", mir_models.ErrorNoDeviceTargetProvided.Error()},
					},
				},
			})
		}

		resp, err := s.sendCommandToDevices(req)
		if err != nil {
			l.Error().Err(e).Msg("error occure while processing send command request")
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
				Response: &cmd_apiv1.SendCommandResponse_Error{
					Error: &common_apiv1.Error{
						Code:    500,
						Message: err.Error(),
						Details: []string{"500 Internal Server Error", err.Error()},
					},
				},
			})
		}

		bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
			Response: &cmd_apiv1.SendCommandResponse_Ok{
				Ok: &cmd_apiv1.SendCommandResponse_CommandResponses{
					DeviceResponses: resp,
					Encoding:        req.PayloadEncoding,
				},
			},
		})
	}
}

type cmdDevicePayload struct {
	deviceId string
	payload  []byte
}

// TODO reconcile device schema with descriptor name as well
func (s *ProtoCmdServer) sendCommandToDevices(req *cmd_apiv1.SendCommandRequest) (map[string]*cmd_apiv1.SendCommandResponse_CommandResponse, error) {
	devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
	if err != nil {
		return nil, err
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

			s.devSchemasLock.RLock()
			devSchema, ok := s.devSchemas[dev.Spec.DeviceId]
			s.devSchemasLock.RUnlock()
			if !ok || req.RefreshSchema {
				devSchema, err = mir_utils.ReconcileDeviceSchema(s.m, s.devStore, dev.Spec.DeviceId, false)
				s.devSchemasLock.Lock()
				s.devSchemas[dev.Spec.DeviceId] = devSchema
				s.devSchemasLock.Unlock()
				if err != nil {
					devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error reconciling device schema").Error(),
					}
					devInError = true
					continue
				}
			}

			msgReqDesc, err := devSchema.FindDescriptorByName(protoreflect.FullName(req.Name))
			if err != nil {
				// This time we reconcile and make sure we fetch the schema from the device itself and not db
				devSchema, err = mir_utils.ReconcileDeviceSchema(s.m, s.devStore, dev.Spec.DeviceId, true)
				s.devSchemasLock.Lock()
				s.devSchemas[dev.Spec.DeviceId] = devSchema
				s.devSchemasLock.Unlock()
				if err != nil {
					devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error reconciling device schema").Error(),
					}
					devInError = true
					continue
				}
				msgReqDesc, err = devSchema.FindDescriptorByName(protoreflect.FullName(req.Name))
				if err != nil {
					devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error finding command. make sure schema is up to date and command name is correct").Error(),
					}
					devInError = true
					continue
				}
			}
			payloadReq := dynamicpb.NewMessage(msgReqDesc.(protoreflect.MessageDescriptor))

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
				devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:  errors.Wrap(err, "error unmarshaling payload").Error(),
				}
				devInError = true
				continue
			}

			devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
				Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_VALIDATED,
			}
			commandsToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: bytePayload, deviceId: dev.Spec.DeviceId}

			l.Debug().Str("payload", fmt.Sprintf("%s", payloadReq)).Msgf("command %s validated for device %s", req.Name, dev.GetNameNamespace())
		}
	} else {
		for _, dev := range devs {
			devResp[dev.GetNameNamespace()] = &cmd_apiv1.SendCommandResponse_CommandResponse{
				Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_PENDING,
			}
			commandsToSend[dev.GetNameNamespace()] = &cmdDevicePayload{payload: req.Payload, deviceId: dev.Spec.DeviceId}
		}
	}

	// If we we have validation error or it is a dry run, we return the request
	// If ForcePush is on, means we send the command to each validated devices
	if (devInError && !req.ForcePush) || req.DryRun {
		return devResp, nil
	}

	// Sends commands
	wg := &sync.WaitGroup{}
	wg.Add(len(commandsToSend))

	timeout := 10 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}
	for nameNs, p := range commandsToSend {
		go func() {
			defer wg.Done()
			cmdResp := &mir.ProtoCmdDesc{}
			err = s.m.SendRequestWithTimeout(
				mir.Command().V1Alpha().SendRawCommand(
					p.deviceId, &mir.ProtoCmdDesc{
						Name:    req.Name,
						Payload: p.payload,
					}, cmdResp), timeout)
			if err != nil {
				devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
					Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
					Error:  errors.Wrap(err, "error during sent command request to device").Error(),
				}
				return
			}

			respPayload := cmdResp.Payload
			if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				msgRespDesc, err := s.devSchemas[p.deviceId].FindDescriptorByName(protoreflect.FullName(cmdResp.Name))
				// TODO refetch schema
				if err != nil {
					devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error finding command response in schema. make sure schema is up to date and command name is correct").Error(),
					}
					return
				}
				msgResp := dynamicpb.NewMessage(msgRespDesc.(protoreflect.MessageDescriptor))
				err = proto.Unmarshal(respPayload, msgResp)
				if err != nil {
					devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error unmarshaling payload").Error(),
					}
					return
				}
				respPayload, err = protojson.Marshal(msgResp)
				if err != nil {
					devResp[nameNs] = &cmd_apiv1.SendCommandResponse_CommandResponse{
						Status: cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_ERROR,
						Error:  errors.Wrap(err, "error marshaling proto payload to json").Error(),
					}
					return
				}
			}

			devResp[nameNs].Name = cmdResp.Name
			devResp[nameNs].Payload = respPayload
			devResp[nameNs].Status = cmd_apiv1.CommandResponseStatus_COMMAND_RESPONSE_STATUS_SUCCESS
		}()
	}
	wg.Wait()

	return devResp, nil
}

func (s *ProtoCmdServer) listCommandsSub() func(msg *nats.Msg, req *cmd_apiv1.SendListCommandsRequest, e error) {
	return func(msg *nats.Msg, req *cmd_apiv1.SendListCommandsRequest, e error) {
		if e != nil {
			l.Error().Err(e).Msg("error occure while receiving request")
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendListCommandsResponse{
				Response: &cmd_apiv1.SendListCommandsResponse_Error{
					Error: &common_apiv1.Error{
						Code:    400,
						Message: mir_models.ErrorApiDeserializingRequest.Error(),
						Details: []string{"400 Bad Request", e.Error()},
					},
				},
			})
		}

		// 1. get device list
		// 2. for each device, get stored schema, if empty, fetch from device
		// 3. return list of commands

		devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
		if err != nil {
			l.Error().Err(e).Msg("error occure while listing devices")
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendListCommandsResponse{
				Response: &cmd_apiv1.SendListCommandsResponse_Error{
					Error: &common_apiv1.Error{
						Code:    500,
						Message: mir_models.ErrorDbExecutingQuery.Error(),
						Details: []string{"500 Bad Request", e.Error()},
					},
				},
			})
		}

		devsCmds := make(map[string]*cmd_apiv1.Commands)
		for _, dev := range devs {
			cmdsList := []*cmd_apiv1.CommandDescriptor{}
			// TODO force options
			reg, err := mir_utils.ReconcileDeviceSchema(s.m, s.devStore, dev.Spec.DeviceId, req.RefreshSchema)
			if err != nil {
				cmdsList = append(cmdsList, &cmd_apiv1.CommandDescriptor{
					Name: err.Error(),
				})
				devsCmds[dev.Spec.DeviceId] = &cmd_apiv1.Commands{
					Commands: cmdsList,
				}
				continue
			}

			cmds, err := reg.GetCommandsList()
			if err != nil {
				cmdsList = append(cmdsList, &cmd_apiv1.CommandDescriptor{
					Name: err.Error(),
				})
			} else {
				for _, cmd := range cmds {
					cmdsList = append(cmdsList, cmd)
				}
			}
			devsCmds[dev.Spec.DeviceId] = &cmd_apiv1.Commands{
				Commands: cmdsList,
			}
		}

		bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendListCommandsResponse{
			Response: &cmd_apiv1.SendListCommandsResponse_Ok{
				Ok: &cmd_apiv1.DevicesCommands{
					DeviceCommands: devsCmds,
				},
			},
		})
	}
}
