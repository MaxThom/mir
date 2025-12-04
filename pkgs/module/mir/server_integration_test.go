package mir

import (
	"encoding/json"
	"testing"
	"time"

	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"google.golang.org/protobuf/proto"
	"gotest.tools/assert"
)

func TestServerRoutes_NewSubject(t *testing.T) {
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
			expected: "client.*.test.v1.func1",
		},
		{
			module:   "app",
			version:  "v2",
			function: "func2",
			extra:    []string{"extra1", "extra2"},
			expected: "client.*.app.v2.func2.extra1.extra2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			subject := m.Client().NewSubject(tt.module, tt.version, tt.function, tt.extra...)
			assert.Equal(t, tt.expected, subject.String())
		})
	}
}

func TestServerRoutes_Subscribe(t *testing.T) {
	subject := m.Client().NewSubject("test", "v1", "function")
	received := make(chan bool)

	err := m.Client().Subscribe(subject, func(msg *Msg, id string, data []byte) {
		assert.Equal(t, "test-data", string(data))
		received <- true
	})
	assert.NilError(t, err)

	err = m.Client().Publish(subject, []byte("test-data"))
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestServerRoutes_QueueSubscribe(t *testing.T) {
	subject := m.Client().NewSubject("test", "v1", "function")
	queueName := "test-queue-server"
	messageCount := 10
	receivedCount1 := 0
	receivedCount2 := 0

	err := m.Client().QueueSubscribe(queueName, subject, func(msg *Msg, id string, data []byte) {
		receivedCount1++
	})
	assert.NilError(t, err)

	err = m.Client().QueueSubscribe(queueName, subject, func(msg *Msg, id string, data []byte) {
		receivedCount2++
	})
	assert.NilError(t, err)

	for i := 0; i < messageCount; i++ {
		err = m.Client().Publish(subject, []byte("test-data"))
		assert.NilError(t, err)
	}

	time.Sleep(1 * time.Second)
	assert.Equal(t, messageCount, receivedCount1+receivedCount2)
	assert.Assert(t, receivedCount1 > 0 && receivedCount2 > 0)
}

func TestServerRoutes_CreateDevice(t *testing.T) {
	deviceID := "test-server-create"
	testDevice := mir_v1.NewDevice().WithMeta(
		mir_v1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		}).WithSpec(
		mir_v1.DeviceSpec{
			DeviceId: deviceID,
		})

	err := m.Client().CreateDevice().Subscribe(
		func(msg *Msg, clientId string, d mir_v1.Device) (mir_v1.Device, error) {
			return mir_v1.NewDevice().WithMeta(
				mir_v1.Meta{
					Name:      testDevice.Meta.Name,
					Namespace: testDevice.Meta.Namespace,
				}).WithSpec(
				mir_v1.DeviceSpec{
					DeviceId: testDevice.Spec.DeviceId,
				},
			), nil
		})
	assert.NilError(t, err)

	// Test request
	req := mir_v1.NewDevice().WithMeta(
		mir_v1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		}).WithSpec(
		mir_v1.DeviceSpec{
			DeviceId: deviceID,
		})

	device, err := m.Client().CreateDevice().Request(req)
	assert.NilError(t, err)
	assert.Equal(t, testDevice.Meta.Name, device.Meta.Name)
	assert.Equal(t, testDevice.Spec.DeviceId, device.Spec.DeviceId)
}

func TestServerRoutes_UpdateDevice(t *testing.T) {
	deviceID := "test-server-update"
	testDevice := mir_v1.NewDevice().WithMeta(
		mir_v1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: deviceID,
	})

	err := m.Client().UpdateDevice().Subscribe(
		func(msg *Msg, clientId string, t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error) {
			return []mir_v1.Device{
				testDevice,
			}, nil
		})
	assert.NilError(t, err)

	// Test request

	device, err := m.Client().UpdateDevice().RequestSingle(testDevice)
	assert.NilError(t, err)
	assert.Equal(t, testDevice.Meta.Name, device[0].Meta.Name)
	assert.Equal(t, testDevice.Spec.DeviceId, device[0].Spec.DeviceId)
}

func TestServerRoutes_DeleteDevice(t *testing.T) {
	deviceID := "test-server-delete"
	testDevice := mir_v1.NewDevice().WithMeta(
		mir_v1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: deviceID,
	})

	err := m.Client().DeleteDevice().Subscribe(
		func(msg *Msg, clientId string, req mir_v1.DeviceTarget) ([]mir_v1.Device, error) {
			return []mir_v1.Device{
				testDevice,
			}, nil
		})
	assert.NilError(t, err)

	// Test request

	device, err := m.Client().DeleteDevice().Request(testDevice.ToTarget())
	assert.NilError(t, err)
	assert.Equal(t, testDevice.Meta.Name, device[0].Meta.Name)
	assert.Equal(t, testDevice.Spec.DeviceId, device[0].Spec.DeviceId)
}

func TestServerRoutes_ListDevice(t *testing.T) {
	deviceID := "test-server-list"
	testDevice := mir_v1.NewDevice().WithMeta(
		mir_v1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: deviceID,
	})

	err := m.Client().ListDevice().Subscribe(
		func(msg *Msg, clientId string, d mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error) {
			return []mir_v1.Device{testDevice}, nil
		})
	assert.NilError(t, err)

	// Test request

	device, err := m.Client().ListDevice().Request(testDevice.ToTarget(), false)
	assert.NilError(t, err)
	assert.Equal(t, testDevice.Meta.Name, device[0].Meta.Name)
	assert.Equal(t, testDevice.Spec.DeviceId, device[0].Spec.DeviceId)
}

func TestServerRoutes_PublishSubscribe(t *testing.T) {
	// Test custom server publish/subscribe
	subject := m.Client().NewSubject("test", "v1", "custom")
	received := make(chan bool)

	// Subscribe
	err := m.Client().Subscribe(subject, func(msg *Msg, clientId string, data []byte) {
		assert.Equal(t, "test-data", string(data))
		received <- true
	})
	assert.NilError(t, err)

	// Publish
	err = m.Client().Publish(subject, []byte("test-data"))
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestServerRoutes_PublishProto(t *testing.T) {
	subject := m.Client().NewSubject("test", "v1", "proto")
	received := make(chan bool)

	testDevice := &mir_apiv1.Device{
		Meta: &mir_apiv1.Meta{
			Name:      "Test Device",
			Namespace: "default",
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "test-server-publishproto",
		},
	}

	// Subscribe
	err := m.Client().Subscribe(subject, func(msg *Msg, clientId string, data []byte) {
		device := &mir_apiv1.Device{}
		err := proto.Unmarshal(data, device)
		assert.NilError(t, err)
		assert.Equal(t, testDevice.Meta.Name, device.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, device.Spec.DeviceId)
		received <- true
	})
	assert.NilError(t, err)

	// Publish
	err = m.Client().PublishProto(subject, testDevice)
	assert.NilError(t, err)

	select {
	case <-received:
		// Test passed
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestServerRoutes_PublishJson(t *testing.T) {
	subject := m.Client().NewSubject("test", "v1", "json")
	received := make(chan bool)

	testData := mir_v1.Device{
		Object: mir_v1.Object{
			Meta: mir_v1.Meta{
				Name:      "Test Device",
				Namespace: "default",
			},
		},
		Spec: mir_v1.DeviceSpec{
			DeviceId: "test-server-publishjson",
		},
	}

	// Subscribe
	err := m.Client().Subscribe(subject, func(msg *Msg, clientId string, data []byte) {
		var device mir_v1.Device
		err := json.Unmarshal(data, &device)
		assert.NilError(t, err)
		assert.Equal(t, testData.Meta.Name, device.Meta.Name)
		assert.Equal(t, testData.Spec.DeviceId, device.Spec.DeviceId)
		received <- true
	})
	assert.NilError(t, err)

	// Publish
	err = m.Client().PublishJson(subject, testData)
	assert.NilError(t, err)

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestServerRoutes_ListTelemetry(t *testing.T) {
	deviceID := "test-server-listtlm"
	testTelemetry := []*mir_apiv1.DevicesTelemetry{
		{
			DevicesNamens: []string{deviceID},
			TlmDescriptors: []*mir_apiv1.TelemetryDescriptor{
				{
					Name: "test",
				},
			},
		},
	}

	err := m.Client().ListTelemetry().Subscribe(
		func(msg *Msg, clientId string, req *mir_apiv1.SendListTelemetryRequest) ([]*mir_apiv1.DevicesTelemetry, error) {
			return testTelemetry, nil
		},
	)
	assert.NilError(t, err)

	resp, err := m.Client().ListTelemetry().Request(&mir_apiv1.SendListTelemetryRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{deviceID},
		},
	})
	assert.NilError(t, err)
	assert.Equal(t, resp[0].DevicesNamens[0], deviceID)
}

func TestServerRoutes_ListCommand(t *testing.T) {
	deviceID := "test-server-listcmd"
	testCommands := []*mir_apiv1.DevicesCommands{
		{
			DevicesNamens: []string{"test-server-listcmd"},
			CmdDescriptors: []*mir_apiv1.CommandDescriptor{
				{
					Name: "cmd_test",
				},
			},
		},
	}

	err := m.Client().ListCommands().Subscribe(
		func(msg *Msg, clientId string, req *mir_apiv1.SendListCommandsRequest) ([]*mir_apiv1.DevicesCommands, error) {
			return testCommands, nil
		},
	)
	assert.NilError(t, err)

	resp, err := m.Client().ListCommands().Request(&mir_apiv1.SendListCommandsRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{deviceID},
		},
	})
	assert.NilError(t, err)
	assert.Equal(t, resp[0].DevicesNamens[0], testCommands[0].DevicesNamens[0])
}

func TestServerRoutes_SendCommand(t *testing.T) {
	deviceID := "test-server-sendcmd"
	testCommands := &mir_apiv1.SendCommandResponse_CommandResponses{
		DeviceResponses: map[string]*mir_apiv1.SendCommandResponse_CommandResponse{
			"test-server-sendcmd": {
				DeviceId: deviceID,
			},
		},
	}

	done := make(chan bool)
	err := m.Client().SendCommand().QueueSubscribe("test_module",
		func(msg *Msg, clientId string, req *mir_apiv1.SendCommandRequest) (*mir_apiv1.SendCommandResponse_CommandResponses, error) {
			done <- true
			return testCommands, nil
		},
	)
	assert.NilError(t, err)

	// Conflict with the cmd server running
	// We can only validate if the route work and not the reply
	_, _ = m.Client().SendCommand().Request(&mir_apiv1.SendCommandRequest{
		Targets: &mir_apiv1.DeviceTarget{
			Ids: []string{deviceID},
		},
	})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for command response")
	}
}
