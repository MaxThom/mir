package event_client

import (
	"time"

	"github.com/maxthom/mir/internal/clients"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const (
	ListEventsRequest   clients.ServerSubject = "client.%s.events.v1alpha.list"
	DeleteEventsRequest clients.ServerSubject = "client.%s.events.v1alpha.delete"

	EventsStream clients.ServerSubject = "event.%s.*.*.*"
)

func PublishEventsStream(bus *bus.BusConn) error {
	msg, err := GetEventsStreamMsg()
	if err != nil {
		return err
	}

	return bus.PublishMsg(msg)
}

func GetEventsStreamMsg() (*nats.Msg, error) {
	return &nats.Msg{
		Subject: EventsStream.WithId("*"),
		Header:  nats.Header{},
		Data:    []byte{},
	}, nil
}

func PublishEventListRequest(bus *bus.BusConn, req *mir_apiv1.ListEventsRequest) (*mir_apiv1.ListEventsResponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.ListEventsResponse{}, err
	}
	resMsg, err := bus.Request(ListEventsRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.ListEventsResponse{}, err
	}

	resp := &mir_apiv1.ListEventsResponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.ListEventsResponse{}, err
	}

	return resp, nil
}

func PublishEventDeleteRequest(bus *bus.BusConn, req *mir_apiv1.DeleteEventRequest) (*mir_apiv1.DeleteEventReponse, error) {
	bReq, err := proto.Marshal(req)
	if err != nil {
		return &mir_apiv1.DeleteEventReponse{}, err
	}
	resMsg, err := bus.Request(DeleteEventsRequest.WithId("TODO"), bReq, 7*time.Second)
	if err != nil {
		return &mir_apiv1.DeleteEventReponse{}, err
	}

	resp := &mir_apiv1.DeleteEventReponse{}
	err = proto.Unmarshal(resMsg.Data, resp)
	if err != nil {
		return &mir_apiv1.DeleteEventReponse{}, err
	}

	return resp, nil
}
