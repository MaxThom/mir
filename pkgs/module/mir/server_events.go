package mir

import (
	"errors"

	"github.com/maxthom/mir/internal/clients"
	"github.com/maxthom/mir/internal/clients/event_client"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

/// ListEvents

type listEventsRoute struct {
	m *Mir
}

// List events
func (r *clientRoutes) ListEvents() *listEventsRoute {
	return &listEventsRoute{m: r.m}
}

// Subscribe to list events request
func (r *listEventsRoute) Subscribe(f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) error {
	sbj := event_client.ListEventsRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to list events request
func (r *listEventsRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) error {
	sbj := event_client.ListEventsRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *listEventsRoute) handlerWrapper(f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.ListEventsRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.ListEventsResponse{Response: &mir_apiv1.ListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ClientSubject(msg.Subject).GetId(), mir_v1.ProtoEventTargetToMirEventTarget(req.Target))
		if err != nil {
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.ListEventsResponse{Response: &mir_apiv1.ListEventsResponse_Error{
				Error: err.Error(),
			}})
			return
		}

		err = r.m.sendReplyOrAck(msg, &mir_apiv1.ListEventsResponse{
			Response: &mir_apiv1.ListEventsResponse_Ok{
				Ok: &mir_apiv1.Events{
					Events: mir_v1.MirEventsToProtoEvents(resp),
				},
			},
		})
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &mir_apiv1.ListEventsResponse{Response: &mir_apiv1.ListEventsResponse_Error{
				Error: err.Error(),
			}})
		}
	}
}

// Request listing of events
func (r *listEventsRoute) Request(t mir_v1.EventTarget) ([]mir_v1.Event, error) {
	sbj := event_client.ListEventsRequest.WithId(r.m.GetInstanceName())

	bReq, err := proto.Marshal(&mir_apiv1.ListEventsRequest{
		Target: mir_v1.MirEventTargetToProtoEventTarget(t),
	})
	if err != nil {
		return []mir_v1.Event{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Event{}, err

	}

	resp := &mir_apiv1.ListEventsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Event{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Event{}, errors.New(resp.GetError())
	}

	return mir_v1.ProtoEventsToMirEvents(resp.GetOk().Events), nil
}

/// DeleteEvents

type deleteEventsRoute struct {
	m *Mir
}

// Delete events
func (r *clientRoutes) DeleteEvents() *deleteEventsRoute {
	return &deleteEventsRoute{m: r.m}
}

// Queue subscribe to delete event request
func (r *deleteEventsRoute) Subscribe(f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) error {
	sbj := event_client.DeleteEventsRequest.WithId("*")
	return r.m.subscribe(sbj, r.handlerWrapper(f))
}

// Queue subscribe to delete event request
func (r *deleteEventsRoute) QueueSubscribe(queue string, f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) error {
	sbj := event_client.DeleteEventsRequest.WithId("*")
	return r.m.queueSubscribe(queue, sbj, r.handlerWrapper(f))
}

func (r *deleteEventsRoute) handlerWrapper(f func(msg *Msg, clientId string, req mir_v1.EventTarget) ([]mir_v1.Event, error)) nats.MsgHandler {
	return func(msg *nats.Msg) {
		req := &mir_apiv1.DeleteEventRequest{}
		if err := proto.Unmarshal(msg.Data, req); err != nil {
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.DeleteEventReponse{Response: &mir_apiv1.DeleteEventReponse_Error{
				Error: err.Error(),
			}})
			return
		}

		resp, err := f(&Msg{msg}, clients.ClientSubject(msg.Subject).GetId(), mir_v1.ProtoEventTargetToMirEventTarget(req.Target))
		if err != nil {
			_ = r.m.sendReplyOrAck(msg, &mir_apiv1.DeleteEventReponse{Response: &mir_apiv1.DeleteEventReponse_Error{
				Error: err.Error(),
			}})
			return
		}

		err = r.m.sendReplyOrAck(msg, &mir_apiv1.DeleteEventReponse{
			Response: &mir_apiv1.DeleteEventReponse_Ok{
				Ok: &mir_apiv1.Events{
					Events: mir_v1.MirEventsToProtoEvents(resp),
				},
			},
		})
		if err != nil {
			err = r.m.sendReplyOrAck(msg, &mir_apiv1.DeleteEventReponse{Response: &mir_apiv1.DeleteEventReponse_Error{
				Error: err.Error(),
			}})
		}
	}
}

// Request deletion of events
func (r *deleteEventsRoute) Request(t mir_v1.EventTarget) ([]mir_v1.Event, error) {
	sbj := event_client.DeleteEventsRequest.WithId(r.m.GetInstanceName())

	bReq, err := proto.Marshal(&mir_apiv1.DeleteEventRequest{
		Target: mir_v1.MirEventTargetToProtoEventTarget(t),
	})
	if err != nil {
		return []mir_v1.Event{}, err
	}

	resMsg, err := r.m.request(sbj, bReq, nil, defaultTimeout)
	if err != nil {
		return []mir_v1.Event{}, err
	}

	resp := &mir_apiv1.DeleteEventReponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return []mir_v1.Event{}, err
	}
	if resp.GetError() != "" {
		return []mir_v1.Event{}, errors.New(resp.GetError())
	}

	return mir_v1.ProtoEventsToMirEvents(resp.GetOk().Events), nil
}
