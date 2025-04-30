package mir

import (
	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/eventstore_client"
	event_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/event_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

/// ListEvents

type listEventsRoute struct {
	m *Mir
}

// List events
func (r *serverRoutes) ListEvents() *listEventsRoute {
	return &listEventsRoute{m: r.m}
}

// Subscribe to list events request
func (r *listEventsRoute) Subscribe(f func(msg *Msg, clientId string, req mir_models.ObjectTarget) ([]mir_models.Event, error)) error {
	sbj := eventstore_client.ListEventsRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list telemetry request
func (r *listEventsRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req mir_models.ObjectTarget) ([]mir_models.Event, error)) error {
	sbj := eventstore_client.ListEventsRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listEventsRoute) handlerWrapper(f func(msg *Msg, clientId string, req mir_models.ObjectTarget) ([]mir_models.Event, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &event_apiv1.ListEventsRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			_ = r.m.sendReplyOrAck(msg, &event_apiv1.ListEventsResponse{Response: &event_apiv1.ListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), mir_models.ProtoObjectTargetToMirObjectTarget(req.Targets))
		if err != nil {
			_ = r.m.sendReplyOrAck(msg, &event_apiv1.ListEventsResponse{Response: &event_apiv1.ListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		err = r.m.sendReplyOrAck(msg, &event_apiv1.ListEventsResponse{
			Response: &event_apiv1.ListEventsResponse_Ok{
				Ok: &event_apiv1.Events{
					Events: mir_models.MirEventsToProtoEvents(resp),
				},
			},
		})
	}
}

// Request listing of telemetry per device
func (r *listEventsRoute) Request(t mir_models.ObjectTarget) ([]mir_models.Event, error) {
	sbj := eventstore_client.ListEventsRequest.WithId(r.m.GetInstanceName())

	bReq, err := proto.Marshal(&event_apiv1.ListEventsRequest{
		Targets: mir_models.MirObjectTargetToProtoObjectTarget(t),
	})
	if err != nil {
		return []mir_models.Event{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_models.Event{}, err

	}

	resp := &event_apiv1.ListEventsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_models.Event{}, err
	}
	if resp.GetError() != "" {
		return []mir_models.Event{}, err
	}

	return mir_models.ProtoEventsToMirEvents(resp.GetOk().Events), nil
}
