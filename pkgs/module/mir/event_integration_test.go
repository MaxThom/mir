package mir

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go"
	"gotest.tools/assert"
)

var m *Mir

func TestMain(t *testing.M) {
	// Setup
	var err error
	m, err = Connect("test-client-modulesdk", nats.DefaultURL)
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
		id       string
		module   string
		version  string
		function string
		extra    []string
		expected string
	}{
		{
			id:       "0xf86",
			module:   "test",
			version:  "v1",
			function: "func1",
			extra:    []string{},
			expected: "event.0xf86.test.v1.func1",
		},
		{
			id:       "0xf86",
			module:   "app",
			version:  "v2",
			function: "func2",
			extra:    []string{"extra1", "extra2"},
			expected: "event.0xf86.app.v2.func2.extra1.extra2",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s-%s-%s", tt.id, tt.module, tt.version, tt.function), func(t *testing.T) {
			subject := m.Event().NewSubject(tt.id, tt.module, tt.version, tt.function, tt.extra...)
			assert.Equal(t, tt.expected, subject.String())
		})
	}
}

func TestEventQueueSubscribe(t *testing.T) {
	// Test data
	subject := m.Event().NewSubject("test-queue-event", "test", "v1", "queue-test")
	queueName := "test-queue"
	messageCount := 10
	rCount1 := 0
	rCount2 := 0

	// Subscribe both clients to the same queue group
	err := m.Event().QueueSubscribe(queueName, func(msg *Msg, subjectId string, evt mir_v1.EventSpec, err error) {
		if subjectId == "test-queue-event" {
			rCount1 += 1
		}
	})
	if err != nil {
		t.Error(err)
	}

	err = m.Event().QueueSubscribe(queueName, func(msg *Msg, subjectId string, evt mir_v1.EventSpec, err error) {
		if subjectId == "test-queue-event" {
			rCount2 += 1
		}
	})
	if err != nil {
		t.Error(err)
	}

	// Publish messages
	for i := 0; i < messageCount; i++ {
		err := m.Event().Publish(subject, mir_v1.EventSpec{}, nil)
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

func TestEventQueueSubscribeSubject(t *testing.T) {
	// Test data
	subject := m.Event().NewSubject("test-queue-event-sbj", "test", "v1", "queue-test")
	queueName := "test-queue-sbj"
	messageCount := 10
	rCount1 := 0
	rCount2 := 0

	// Subscribe both clients to the same queue group
	err := m.Event().QueueSubscribeSubject(queueName, subject, func(msg *Msg, subjectId string, evt mir_v1.EventSpec, err error) {
		rCount1 += 1
	})
	if err != nil {
		t.Error(err)
	}

	err = m.Event().QueueSubscribeSubject(queueName, subject, func(msg *Msg, subjectId string, evt mir_v1.EventSpec, err error) {
		rCount2 += 1
	})
	if err != nil {
		t.Error(err)
	}

	// Publish messages
	for i := 0; i < messageCount; i++ {
		err := m.Event().Publish(subject, mir_v1.EventSpec{}, nil)
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
