package mir

import (
	"testing"
	"time"

	"github.com/maxthom/mir/internal/clients/core_client"
	"github.com/maxthom/mir/internal/clients/device_client"
	"github.com/maxthom/mir/internal/clients/tlm_client"
	"github.com/maxthom/mir/internal/libs/compression/zstd"
	"github.com/maxthom/mir/internal/libs/proto/mir_proto"
	device_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/device_api"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"gotest.tools/assert"
)

func TestDeviceRoutes_NewSubject(t *testing.T) {
	tests := []struct {
		module   string
		version  string
		function string
		extra    []string
		expected string
	}{
		{
			module:   "test",
			version:  "v1",
			function: "func1",
			extra:    []string{},
			expected: "device.*.test.v1.func1",
		},
		{
			module:   "app",
			version:  "v2",
			function: "func2",
			extra:    []string{"extra1", "extra2"},
			expected: "device.*.app.v2.func2.extra1.extra2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			subject := m.Device().NewSubject(tt.module, tt.version, tt.function, tt.extra...)
			assert.Equal(t, tt.expected, subject.String())
		})
	}
}

func TestDeviceRoutes_Subscribe(t *testing.T) {
	deviceID := "test-device-1"
	subject := m.Device().NewSubject("test", "v1", "function")

	// Channel to synchronize test
	received := make(chan bool)

	err := m.Device().Subscribe(subject, func(msg *Msg, id string) {
		assert.Equal(t, deviceID, id)
		assert.Equal(t, "test-data", string(msg.Data))
		received <- true
	})
	assert.NilError(t, err)

	subject[1] = deviceID
	err = m.Bus.Publish(subject.String(), []byte("test-data"))
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestDeviceRoutes_QueueSubscribe(t *testing.T) {
	deviceID := "test-device-1"
	subject := m.Device().NewSubject("test", "v1", "function")
	queueName := "test-queue-device"
	messageCount := 10
	receivedCount1 := 0
	receivedCount2 := 0

	// Create two queue subscribers
	err := m.Device().QueueSubscribe(queueName, subject, func(msg *Msg, id string) {
		assert.Equal(t, deviceID, id)
		receivedCount1++
	})
	assert.NilError(t, err)

	err = m.Device().QueueSubscribe(queueName, subject, func(msg *Msg, id string) {
		assert.Equal(t, deviceID, id)
		receivedCount2++
	})
	assert.NilError(t, err)

	subject[1] = deviceID
	for i := 0; i < messageCount; i++ {
		err = m.Bus.Publish(subject.String(), []byte("test-data"))
		assert.NilError(t, err)
	}

	time.Sleep(1 * time.Second)

	// Verify message distribution
	assert.Equal(t, messageCount, receivedCount1+receivedCount2)
	assert.Assert(t, receivedCount1 > 0 && receivedCount2 > 0)
}

func TestDeviceRoutes_Hearthbeat(t *testing.T) {
	deviceID := "test-device-1"
	received := make(chan bool)

	err := m.Device().Hearthbeat().Subscribe(func(msg *Msg, id string) {
		assert.Equal(t, deviceID, id)
		received <- true
	})
	assert.NilError(t, err)

	err = m.Bus.Publish(core_client.HearthbeatDeviceStream.WithId(deviceID), []byte{})
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for hearthbeat")
	}
}

func TestDeviceRoutes_Telemetry(t *testing.T) {
	deviceID := "test-device-1"
	protoMsgName := "test.Message"
	testData := []byte("test-telemetry-data")
	received := make(chan bool)

	err := m.Device().Telemetry().Subscribe(func(msg *Msg, id string, msgName string, data []byte) {
		assert.Equal(t, deviceID, id)
		assert.Equal(t, protoMsgName, msgName)
		assert.DeepEqual(t, testData, data)
		received <- true
	})
	assert.NilError(t, err)

	msg := nats.NewMsg(tlm_client.TelemetryDeviceStream.WithId(deviceID))
	msg.Header = nats.Header{"__msg": []string{protoMsgName}}
	msg.Data = testData
	err = m.Bus.PublishMsg(msg)
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for telemetry")
	}
}

func TestDeviceRoutes_Schema(t *testing.T) {
	deviceID := "test-device-1"
	mockSchema := &mir_proto.MirProtoSchema{}

	// Setup mock response handler
	sub, err := m.Bus.Subscribe(device_client.SchemaRequest.WithId(deviceID),
		func(msg *nats.Msg) {
			bSchema, err := mockSchema.MarshalSchema()
			assert.NilError(t, err)
			response := &device_apiv1.SchemaRetrieveResponse{
				Response: &device_apiv1.SchemaRetrieveResponse_Schema{
					Schema: bSchema,
				},
			}
			responseBytes, err := proto.Marshal(response)
			assert.NilError(t, err)
			data, err := zstd.CompressData(responseBytes)
			assert.NilError(t, err)
			msgResp := nats.Msg{
				Subject: msg.Reply,
				Header: map[string][]string{
					HeaderContentEncoding: {HeaderZstdEncoding},
				},
				Data: data,
			}
			err = m.Bus.PublishMsg(&msgResp)
			assert.NilError(t, err)
		})
	assert.NilError(t, err)

	// Test schema request
	schema, err := m.Device().Schema().Request(deviceID)
	assert.NilError(t, err)
	assert.Equal(t, len(mockSchema.GetPackageList()), len(schema.GetPackageList()))
	sub.Unsubscribe()
}

func TestDeviceRoutes_Command(t *testing.T) {
	deviceID := "test-device-1"
	cmdName := "test.Command"
	cmdPayload := []byte("test-command-payload")

	// Setup mock response handler
	sub, err := m.Bus.Subscribe(device_client.CommandRequest.WithId(deviceID),
		func(msg *nats.Msg) {
			assert.Equal(t, cmdName, msg.Header.Get("__msg"))
			assert.DeepEqual(t, cmdPayload, msg.Data)

			responseMsg := nats.NewMsg(msg.Reply)
			responseMsg.Header = nats.Header{"__msg": []string{"test.Response"}}
			responseMsg.Data = []byte("test-response-payload")

			err := m.Bus.PublishMsg(responseMsg)
			assert.NilError(t, err)
		})
	assert.NilError(t, err)

	// Test command request
	resp, err := m.Device().Command().RequestRaw(deviceID, ProtoCmdDesc{
		Name:    cmdName,
		Payload: cmdPayload,
	}, time.Second*7)
	assert.NilError(t, err)
	assert.Equal(t, "test.Response", resp.Name)
	assert.DeepEqual(t, []byte("test-response-payload"), resp.Payload)
	sub.Unsubscribe()
}
