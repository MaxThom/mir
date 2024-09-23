package protocmd_srv

import (
	"context"
	"fmt"

	"github.com/maxthom/mir/internal/clients/cmd_client"
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
	s.m.SubscribeRaw(cmd_client.SendCommandRequest.WithId("*"), sendCommandsSub(s.m.Bus))
	s.m.Subscribe(mir.Client().V1Alpha().ListCommands(s.listCommandsSub()))
}

func sendCommandsSub(b *nats.Conn) nats.MsgHandler {
	return func(msg *nats.Msg) {
		fmt.Println("SEND COMMAND RECEIVED")
		bus.SendProtoReplyOrAck(b, msg, &cmd_apiv1.SendCommandResponse{
			Response: &cmd_apiv1.SendCommandResponse_Ok{
				Ok: "COMMAND EXECUTED",
			},
		})
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

		fmt.Println("LIST COMMAND RECEIVED")
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
			reg, err := mir_utils.ReconcileDeviceSchema(s.m, s.devStore, dev.Spec.DeviceId, false)
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
					cmdsList = append(cmdsList, &cmd_apiv1.CommandDescriptor{
						Name: cmd,
					})
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
