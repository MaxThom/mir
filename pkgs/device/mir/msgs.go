package mir

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	devicev1 "github.com/maxthom/mir/pkgs/device/gen/proto/mir/device/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	msgHandlers map[string]func(msg *nats.Msg, m *Mir) error
	msgSender   func(msg *nats.Msg) error
)

func init() {
	msgHandlers = make(map[string]func(msg *nats.Msg, m *Mir) error)
	msgHandlers[device_client.SchemaRequest.GetVersionAndFunction()] = SchemaRetrieveHandler
	msgHandlers[device_client.CommandRequest.GetVersionAndFunction()] = DefinedCommandHandler
	msgHandlers[device_client.ConfigRequest.GetVersionAndFunction()] = DefinedConfigHandler
}

func SchemaRetrieveHandler(msg *nats.Msg, m *Mir) error {
	shouldZstd := false
	if msg.Header.Get("request-encoding") == "zstd" {
		shouldZstd = true
	}
	bytes, err := proto.Marshal(m.schema)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &device_apiv1.SchemaRetrieveResponse{
			Response: &device_apiv1.SchemaRetrieveResponse_Error{
				Error: fmt.Sprintf("error occure while marshaiing schema: %s", err.Error()),
			},
		}, nil, false)
	}

	return sendReplyOrAck(m.b, msg, &device_apiv1.SchemaRetrieveResponse{
		Response: &device_apiv1.SchemaRetrieveResponse_Schema{
			Schema: bytes,
		},
	}, nil, shouldZstd)
}

func DefinedCommandHandler(msg *nats.Msg, m *Mir) error {
	shouldZstd := false
	if msg.Header.Get("request-encoding") == "zstd" {
		shouldZstd = true
	}
	descName := msg.Header.Get("__msg")
	_, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(descName))
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

	v := reflect.New(h.t).Interface()
	cmdMsg := v.(proto.Message)

	if err = proto.Unmarshal(msg.Data, cmdMsg); err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error while unmarshalling command payload: %w", err).Error(),
		}, nil, false)
	}

	cmdResp, err := h.h(cmdMsg)
	if err != nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Error{
			Message: fmt.Errorf("device error in command handler: %w", err).Error(),
		}, nil, false)
	}

	if cmdResp == nil {
		return sendReplyOrAck(m.b, msg, &devicev1.Void{}, nil, false)
	}
	return sendReplyOrAck(m.b, msg, cmdResp, nil, shouldZstd)
}

func DefinedConfigHandler(msg *nats.Msg, m *Mir) error {
	descName := msg.Header.Get("__msg")
	timeStr := msg.Header.Get("__time")
	// _, err := m.schemaReg.FindDescriptorByName(protoreflect.FullName(descName))
	// if err != nil {
	// 	return fmt.Errorf("device error while looking for config descriptor: %w", err)
	// }
	updTime, err := time.Parse(time.RFC3339, timeStr)
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

	h, ok := m.cfgHandlers[descName]
	if !ok {
		return fmt.Errorf("device error: no handler for config %s found", descName)
	}

	v := reflect.New(h.t).Interface()
	cmdMsg := v.(proto.Message)
	if err = proto.Unmarshal(msg.Data, cmdMsg); err != nil {
		return fmt.Errorf("device error while unmarshalling config payload: %w", err)
	}

	for _, handler := range h.h {
		handler(cmdMsg)
	}

	// TODO maybe should return reported properties as dx
	return nil // sendReplyOrAck(m.b, msg, cmdResp, nil, shouldZstd)
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
		h.Add("__msg", string(m.ProtoReflect().Descriptor().FullName()))

		data := bResp
		if shouldZstdCompress {
			compressedBytes, err := zstd.CompressData(bResp)
			if err == nil {
				data = compressedBytes
				h.Add("content-encoding", "zstd")
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
	h.Add("__msg", string(protoMsg.ProtoReflect().Descriptor().FullName()))

	data := bResp
	if shouldZstdCompress {
		compressedBytes, err := zstd.CompressData(bResp)
		if err == nil {
			data = compressedBytes
			h.Add("content-encoding", "zstd")
		}
	}

	return m.sendMsg(&nats.Msg{
		Subject: subject,
		Header:  h,
		Data:    data,
	})
}

func (m Mir) setOnlineHandler() {
	if m.store.opts.Msgs.MsgStorageType == StorageTypeNoStorage || m.store.opts.InMemory {
		msgSender = m.sendMsgOnly
		m.l.Info().Msg("set online handler: no storage")
	} else if m.store.opts.Msgs.MsgStorageType == StorageTypeOnlyIfOffline {
		msgSender = m.sendMsgOnly
		m.l.Info().Msg("set online handler: no storage")
	} else if m.store.opts.Msgs.MsgStorageType == StorageTypePersistent {
		msgSender = m.sendMsgWithStorage
		m.l.Info().Msg("set online handler: persistent storage")
	}
}

func (m Mir) setOfflineHandler() {
	if m.store.opts.Msgs.MsgStorageType == StorageTypeNoStorage || m.store.opts.InMemory {
		msgSender = m.sendNothing
		m.l.Info().Msg("set offline handler: no storage")
	} else if m.store.opts.Msgs.MsgStorageType == StorageTypeOnlyIfOffline {
		msgSender = m.saveMsgInPending
		m.l.Info().Msg("set offline handler: pending storage")
	} else if m.store.opts.Msgs.MsgStorageType == StorageTypePersistent {
		msgSender = m.saveMsgInPending
		m.l.Info().Msg("set offline handler: pending storage")
	}
}

func (m Mir) sendMsg(msg *nats.Msg) error {
	return msgSender(msg)
}

func (m Mir) sendNothing(msg *nats.Msg) error {
	return nil
}
func (m Mir) sendMsgOnly(msg *nats.Msg) error {
	return m.b.PublishMsg(msg)
}

func (m Mir) saveMsgInPending(msg *nats.Msg) error {
	return m.store.SaveMsgToPending(*msg)
}

// Performance solution would be to do batch writes
func (m Mir) sendMsgWithStorage(msg *nats.Msg) error {
	if err := m.store.SaveMsgToPermanent(*msg); err != nil {
		m.l.Warn().Err(err).Msg("error saving msg to sent store")
	}
	return m.b.PublishMsg(msg)
}

func (m Mir) sendPendingMsgs() {
	if !m.store.opts.InMemory {
		batchSize := 100
		if m.cfg.Store.Msgs.MsgStorageType == StorageTypePersistent {
			count := 0
			if err := m.store.SwapMsgByBatch(msgPendingBucket, msgPersistentBucket, batchSize, func(msgs []nats.Msg) error {
				var errs error
				for _, msg := range msgs {
					errs = errors.Join(m.sendMsgOnly(&msg))
				}
				count += len(msgs)
				return errs
			}); err != nil {
				m.l.Error().Err(err).Msg("error sending pending messages to Mir")
			}
			m.l.Info().Msgf("%d pending messages sent to Mir and moved to persistent storage", count)
		} else {
			count := 0
			if err := m.store.DeleteMsgByBatch(msgPendingBucket, batchSize, func(msgs []nats.Msg) error {
				var errs error
				for _, msg := range msgs {
					errs = errors.Join(m.sendMsgOnly(&msg))
				}
				count += len(msgs)
				return errs
			}); err != nil {
				m.l.Error().Err(err).Msg("error sending pending messages to Mir")
			}
			m.l.Info().Msgf("%d pending messages sent to Mir", count)
		}
	}
}
