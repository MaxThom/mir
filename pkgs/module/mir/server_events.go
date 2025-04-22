package mir

import (
	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/eventstore_client"
	eventstore_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/eventstore_api"
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
func (r *listEventsRoute) Subscribe(f func(msg *Msg, clientId string, req *eventstore_apiv1.SendListEventsRequest) ([]*eventstore_apiv1.Event, error)) error {
	sbj := eventstore_client.ListEventsRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list telemetry request
func (r *listEventsRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req *eventstore_apiv1.SendListEventsRequest) ([]*eventstore_apiv1.Event, error)) error {
	sbj := eventstore_client.ListEventsRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listEventsRoute) handlerWrapper(f func(msg *Msg, clientId string, req *eventstore_apiv1.SendListEventsRequest) ([]*eventstore_apiv1.Event, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &eventstore_apiv1.SendListEventsRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			// TODO log error here
			_ = r.m.sendReplyOrAck(msg, &eventstore_apiv1.SendListEventsResponse{Response: &eventstore_apiv1.SendListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ServerSubject(msg.Subject).GetId(), req)
		if err != nil {
			_ = r.m.sendReplyOrAck(msg, &eventstore_apiv1.SendListEventsResponse{Response: &eventstore_apiv1.SendListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}
		// TODO log error here
		err = r.m.sendReplyOrAck(msg, &eventstore_apiv1.SendListEventsResponse{
			Response: &eventstore_apiv1.SendListEventsResponse_Ok{
				Ok: &eventstore_apiv1.Events{
					Events: resp,
				},
			},
		})
	}
}

// Request listing of telemetry per device
func (r *listEventsRoute) Request(req *eventstore_apiv1.SendListEventsRequest) ([]*eventstore_apiv1.Event, error) {
	sbj := eventstore_client.ListEventsRequest.WithId(r.m.GetInstanceName())
	bReq, err := proto.Marshal(req)
	if err != nil {
		return []*eventstore_apiv1.Event{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []*eventstore_apiv1.Event{}, err

	}

	resp := &eventstore_apiv1.SendListEventsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []*eventstore_apiv1.Event{}, err
	}
	if resp.GetError() != "" {
		return []*eventstore_apiv1.Event{}, err
	}

	return resp.GetOk().Events, nil
}
