package mir

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/maxthom/mir/internal/clients/cfg_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	HeaderMsgName = "mir-msg"
	HeaderTime    = "mir-time"
	HeaderSchema  = "mir-schema"
)

func (m *Mir) schemaRetrieveHandler(msg *nats.Msg) error {
	shouldZstd := false
	if msg.Header.Get("mir-request-encoding") == "mir-zstd" {
		shouldZstd = true
	}
	bytes, err := proto.Marshal(m.schema)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &mir_apiv1.SchemaRetrieveResponse{
			Response: &mir_apiv1.SchemaRetrieveResponse_Error{
				Error: fmt.Sprintf("error occure while marshalling schema: %s", err.Error()),
			},
		}, nil, false)
	}

	return sendReplyOrAck(m.b, msg, &mir_apiv1.SchemaRetrieveResponse{
		Response: &mir_apiv1.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, shouldZstd)
}

func (m *Mir) definedCommandHandler(msg *nats.Msg) error {
	shouldZstd := false
	if msg.Header.Get("mir-request-encoding") == "mir-zstd" {
		shouldZstd = true
	}
	descName := msg.Header.Get(HeaderMsgName)
	desc, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(descName))
	if err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error while looking for command descriptor: %w", err).Error(),
		}, nil, false)
	}

	h, ok := m.cmdHandlers[descName]
	if !ok {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: "device error: no handler for command " + descName + " found",
		}, nil, false)
	}

	var cmdMsg proto.Message
	if h.t == reflect.TypeFor[dynamicpb.Message]() {
		cmdMsg = dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
	} else {
		v := reflect.New(h.t).Interface()
		cmdMsg = v.(proto.Message)
	}

	if err = proto.Unmarshal(msg.Data, cmdMsg); err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error while unmarshalling command payload: %w", err).Error(),
		}, nil, false)
	}

	cmdResp, err := h.h(cmdMsg)
	if err != nil {
		if err := sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error in command handler: %w", err).Error(),
		}, nil, false); err != nil {
			m.l.Error().Err(err).Msg("error sending command response")
		}
		return nil
	}

	if cmdResp == nil {
		if err := sendReplyOrAck(m.b, msg, &devicev1.Void{}, nil, false); err != nil {
			m.l.Error().Err(err).Msg("error sending command response")
		}
		return nil
	}
	if err := sendReplyOrAck(m.b, msg, cmdResp, nil, shouldZstd); err != nil {
		m.l.Error().Err(err).Msg("error sending command response")
	}

	return nil
}

func (m *Mir) definedConfigHandler(msg *nats.Msg) error {
	descName := msg.Header.Get(HeaderMsgName)
	timeStr := msg.Header.Get(HeaderTime)
	updTime, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return fmt.Errorf("device error while decoding time: %w", err)
	}
	if new, err := m.store.UpdatePropsIfNew(descName, propsValue{
		LastUpdate: updTime,
		Value:      msg.Data,
	}); !new {
		return nil
	} else if err != nil {
		return fmt.Errorf("device error while updating properties: %w", err)
	}

	// Props are newer
	desc, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(descName))
	if err != nil {
		return fmt.Errorf("device error while looking for property descriptor: %w", err)
	}

	h, ok := m.cfgHandlers[descName]
	if !ok {
		return fmt.Errorf("device error: no handler for property %s found", descName)
	}

	var cfgMsg proto.Message
	if h.t == reflect.TypeFor[dynamicpb.Message]() {
		cfgMsg = dynamicpb.NewMessage(desc.(protoreflect.MessageDescriptor))
	} else {
		v := reflect.New(h.t).Interface()
		cfgMsg = v.(proto.Message)
	}

	if err = proto.Unmarshal(msg.Data, cfgMsg); err != nil {
		return fmt.Errorf("device error while unmarshalling property payload: %w", err)
	}

	for _, handler := range h.h {
		go handler(cfgMsg)
	}

	return nil
}

func sendReplyOrAck(bus *bus.BusConn, msg *nats.Msg, m protoreflect.ProtoMessage, h nats.Header, shouldZstdCompress bool) error {
	if msg.Reply != "" {
		if h == nil {
			h = nats.Header{}
		}
		bResp, err := proto.Marshal(m)
		if err != nil {
			return err
		}
		h.Add(HeaderMsgName, string(m.ProtoReflect().Descriptor().FullName()))

		data := bResp
		if shouldZstdCompress {
			compressedBytes, err := zstd.CompressData(bResp)
			if err == nil {
				data = compressedBytes
				h.Add("mir-content-encoding", "mir-zstd")
			}
		}

		err = bus.PublishMsg(&nats.Msg{
			Subject: msg.Reply,
			Header:  h,
			Data:    data,
		})
		if err != nil {
			return err
		}
	} else {
		return msg.Ack()
	}
	return nil
}

func (m Mir) sendProtoMsg(subject string, protoMsg protoreflect.ProtoMessage, h nats.Header, shouldZstdCompress bool) error {
	if h == nil {
		h = nats.Header{}
	}
	bResp, err := proto.Marshal(protoMsg)
	if err != nil {
		return err
	}
	h.Add(HeaderMsgName, string(protoMsg.ProtoReflect().Descriptor().FullName()))

	data := bResp
	if shouldZstdCompress {
		compressedBytes, err := zstd.CompressData(bResp)
		if err == nil {
			data = compressedBytes
			h.Add("mir-content-encoding", "mir-zstd")
		}
	}

	return m.sendMsg(&nats.Msg{
		Subject: subject,
		Header:  h,
		Data:    data,
	})
}

// Fill the properties store with the latest properties from Mir server
// Also write to the persistent store
func (m Mir) requestDesiredProperties() error {
	resp, err := cfg_client.PublishRequestDesiredPropertiesStream(m.b.Conn, m.cfg.Device.Id)
	if err != nil {
		return err
	}
	if resp.GetError() != "" {
		return fmt.Errorf("%s", resp.GetError())
	}

	props := resp.GetOk()

	var errs error
	for msgName, cfg := range props.Properties {
		updTime, err := time.Parse(time.RFC3339, cfg.Time)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		if _, err := m.store.UpdatePropsIfNew(msgName, propsValue{LastUpdate: updTime, Value: cfg.Property}); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}

	return errs
}
