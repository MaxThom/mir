package mirv2

import (
	"fmt"
	"os"
	"testing"
	"time"

	cmd_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/cmd_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/nats-io/nats.go"
	"gotest.tools/assert"
)

var m *Mir

func TestMain(t *testing.M) {
	// Setup
	var err error
	m, err = Connect("test-client", nats.DefaultURL)
	if err != nil {
		panic(err)
	}

	// Run
	exitVal := t.Run()

	// Teardown
	err = m.Disconnect()
	if err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func TestEventRoutes_NewSubject(t *testing.T) {
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
			expected: "event.*.test.v1.func1",
		},
		{
			module:   "app",
			version:  "v2",
			function: "func2",
			extra:    []string{"extra1", "extra2"},
			expected: "event.*.app.v2.func2.extra1.extra2",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s-%s", tt.module, tt.version, tt.function), func(t *testing.T) {
			subject := m.Event().NewSubject(tt.module, tt.version, tt.function, tt.extra...)
			assert.Equal(t, tt.expected, subject.String())
		})
	}
}

func TestDeviceOnlineEvent(t *testing.T) {
	deviceID := "test-device-1"
	testDevice := mir_models.Device{
		Meta: mir_models.Meta{
			Name:      "Test_Device",
			Namespace: "default",
		},
		Spec: mir_models.Spec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := m.Event().DeviceOnline().Subscribe(func(msg *Msg, serverId string, device mir_models.Device) {
		received <- device
	})

	err = m.Event().DeviceOnline().Publish(m.GetInstanceName(), testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceOfflineEvent(t *testing.T) {
	deviceID := "test-device-1"
	testDevice := mir_models.Device{
		Meta: mir_models.Meta{
			Name:      "Test_Device",
			Namespace: "default",
		},
		Spec: mir_models.Spec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := m.Event().DeviceOffline().Subscribe(func(msg *Msg, serverId string, device mir_models.Device) {
		received <- device
	})

	err = m.Event().DeviceOffline().Publish(m.GetInstanceName(), testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceCreatedEvent(t *testing.T) {
	deviceID := "test-device-1"
	testDevice := mir_models.Device{
		Meta: mir_models.Meta{
			Name:      "Test_Device",
			Namespace: "default",
		},
		Spec: mir_models.Spec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := m.Event().DeviceCreate().Subscribe(func(msg *Msg, serverId string, device mir_models.Device) {
		received <- device
	})

	err = m.Event().DeviceCreate().Publish(m.GetInstanceName(), testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceUpdateEvent(t *testing.T) {
	deviceID := "test-device-1"
	testDevice := mir_models.Device{
		Meta: mir_models.Meta{
			Name:      "Test_Device",
			Namespace: "default",
		},
		Spec: mir_models.Spec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := m.Event().DeviceUpdate().Subscribe(func(msg *Msg, serverId string, device mir_models.Device) {
		received <- device
	})

	err = m.Event().DeviceUpdate().Publish(m.GetInstanceName(), testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestDeviceDeleteEvent(t *testing.T) {
	deviceID := "test-device-1"
	testDevice := mir_models.Device{
		Meta: mir_models.Meta{
			Name:      "Test_Device",
			Namespace: "default",
		},
		Spec: mir_models.Spec{
			DeviceId: deviceID,
		},
	}

	// Channel for test synchronization
	received := make(chan mir_models.Device)

	err := m.Event().DeviceDelete().Subscribe(func(msg *Msg, serverId string, device mir_models.Device) {
		received <- device
	})

	err = m.Event().DeviceDelete().Publish(m.GetInstanceName(), testDevice)
	if err != nil {
		t.Error(err)
	}

	select {
	case receivedDevice := <-received:
		assert.Equal(t, testDevice.Meta.Name, receivedDevice.Meta.Name)
		assert.Equal(t, testDevice.Spec.DeviceId, receivedDevice.Spec.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestCommandEvent(t *testing.T) {
	resp := cmd_apiv1.SendCommandResponse_CommandResponse{
		DeviceId: "0xTest",
	}

	// Channel for test synchronization
	received := make(chan *cmd_apiv1.SendCommandResponse_CommandResponse)

	err := m.Event().Command().Subscribe(func(msg *Msg, serverId string, cmd *cmd_apiv1.SendCommandResponse_CommandResponse) {
		received <- cmd
	})

	err = m.Event().Command().Publish(m.GetInstanceName(), &resp)
	if err != nil {
		t.Error(err)
	}

	select {
	case cmd := <-received:
		assert.Equal(t, resp.DeviceId, cmd.DeviceId)
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestEventQueueSubscribe(t *testing.T) {
	// Test data
	subject := m.Event().NewSubject("test", "v1", "queue-test")
	queueName := "test-queue"
	messageCount := 10
	rCount1 := 0
	rCount2 := 0

	// Subscribe both clients to the same queue group
	err := m.Event().QueueSubscribe(queueName, subject, func(msg *Msg, clientID string) {
		rCount1 += 1
	})
	if err != nil {
		t.Error(err)
	}

	err = m.Event().QueueSubscribe(queueName, subject, func(msg *Msg, clientID string) {
		rCount2 += 1
	})
	if err != nil {
		t.Error(err)
	}

	// Publish messages
	for i := 0; i < messageCount; i++ {
		err := m.Event().Publish(subject, m.GetInstanceName(), []byte{})
		if err != nil {
			t.Error(err)
		}
	}

	// Wait for all messages to be processed
	time.Sleep(1 * time.Second)

	// Verify that messages were distributed between subscribers
	assert.Equal(t, messageCount, rCount1+rCount2)
	assert.Equal(t, rCount1 > 0 && rCount2 > 0, true)
}
