package protocmd_srv

import (
	"context"
	"fmt"

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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoCmdServer struct {
	sub      *nats.Subscription
	m        *mir.Mir
	devStore mng.DeviceStore
	// devWriters     map[deviceProtoKey]proto_lineprotocol.ProtoBytesToLpFn
	// devWritersLock sync.RWMutex
	// devSchemas     map[string]deviceProtoSchema
	// devSchemasLock sync.RWMutex
}

// TODO prom metics
// - number of device schema fetch

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
		m:        m,
		devStore: devStore,
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

		devs, err := s.devStore.ListDevice(&core_apiv1.ListDeviceRequest{Targets: req.Targets})
		if err != nil {
			l.Error().Err(e).Msg("error occure while listing devices")
			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
				Response: &cmd_apiv1.SendCommandResponse_Error{
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

			// TODO add no validation option
			msgReqDesc, _ := reg.FindDescriptorByName(protoreflect.FullName(req.Name))
			cmdReq := dynamicpb.NewMessage(msgReqDesc.(protoreflect.MessageDescriptor))

			if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				_ = protojson.Unmarshal(req.Payload, cmdReq)
			} else {
				_ = proto.Unmarshal(req.Payload, cmdReq)
			}

			fmt.Println("SERVER RECEIVE", cmdReq)

			cmdResp := &mir.ProtoCmdDesc{}
			err = s.m.SendRequest(mir.Command().V1Alpha().SendCommand(dev.Spec.DeviceId, cmdReq, cmdResp))
			if err != nil {
				// TODO handle error
				l.Error().Err(err).Msg("")
			}
			fmt.Println("SERVER RESPONSE", cmdResp)

			// if proto encoding, dont need to marshal it (except for validation)
			// TODO
			// If cant find command, try refresh devices
			// for each device, prepare the command, any errors returns the errors list
			// if no errors, send the command to the devices
			// if force flag, send commands to working devices
			// Return json of response

			respPayload := cmdResp.Payload
			if req.PayloadEncoding == common_apiv1.Encoding_ENCODING_JSON {
				msgRespDesc, _ := reg.FindDescriptorByName(protoreflect.FullName(cmdResp.Name))
				msgResp := dynamicpb.NewMessage(msgRespDesc.(protoreflect.MessageDescriptor))
				err = proto.Unmarshal(respPayload, msgResp)
				respPayload, err = protojson.Marshal(msgResp)
				fmt.Println(string(respPayload))
			}

			bus.SendProtoReplyOrAck(s.m.Bus, msg, &cmd_apiv1.SendCommandResponse{
				Response: &cmd_apiv1.SendCommandResponse_Ok{
					Ok: &cmd_apiv1.SendCommandResponse_Payload{
						Name:     cmdResp.Name,
						Payload:  respPayload,
						Encoding: req.PayloadEncoding,
					},
				},
			})
		}

	}
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
