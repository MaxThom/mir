package eventstore_srv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/mir/pkgs/module/mir"
	"gotest.tools/assert"
)

var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"
var log = test_utils.TestLogger("event")

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	var err error

	mSdk, err = mir.Connect("test_eventstore", busUrl)
	if err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)
	if err := dataCleanUp(); err != nil {
		panic(err)
	}
	// Tests
	fmt.Println("> Test Run")
	exitVal := m.Run()
	time.Sleep(1 * time.Second)

	// Teardown
	fmt.Println("> Test Teardown")
	if err := mSdk.Disconnect(); err != nil {
		fmt.Println(err)
	}
	time.Sleep(1 * time.Second)

	os.Exit(exitVal)
}

func dataCleanUp() error {
	// Clear data
	if _, err := mSdk.Client().DeleteEvents().Request(mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"event_testing"},
		},
	}); err != nil {
		return err
	}
	if _, err := mSdk.Client().DeleteEvents().Request(mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"default"},
		},
	}); err != nil {
		return err
	}
	if _, err := mSdk.Client().DeleteDevice().Request(mir_v1.DeviceTarget{
		Namespaces: []string{"event_testing"},
	}); err != nil {
		return err
	}
	return nil
}

func TestPublishEventStoreNormal(t *testing.T) {
	// Arrange
	j, err := json.Marshal(map[string]any{
		"key":  "value",
		"key2": "value2",
		"key3": map[string]any{
			"key3": "value3",
			"key4": "value4",
		},
	})
	if err != nil {
		t.Error(err)
	}
	sbj := mSdk.Event().NewSubject("0xf86", "event_test", "v1", "list_req")
	name := "list_request_test"
	namespace := "event_testing_store_normal"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}

	// Act
	err = mSdk.Event().Publish(sbj, event, msg)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	events, err := mSdk.Client().ListEvents().Request(
		mir_v1.EventTarget{
			ObjectTarget: mir_v1.ObjectTarget{
				Namespaces: []string{
					namespace,
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	// TODO list with mir_v1.events
	testEvent := mir_v1.Event{}
	for _, event := range events {
		if event.Spec.RelatedObject.Meta.Name == name {
			testEvent = event
		}
	}

	// Assert
	assert.Equal(t, event.Message, testEvent.Spec.Message)
	assert.Equal(t, strings.Contains(testEvent.Meta.Name, name), true)
	assert.Equal(t, namespace, testEvent.Object.Meta.Namespace)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderTrigger], "pizza,toppings,test_eventstore-"), true)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderRoute], sbj.String()), true)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderSubject], sbj.GetId()), true)
}

func TestPublishEventStoreNsDefault(t *testing.T) {
	// Arrange
	j, err := json.Marshal(map[string]any{
		"key":  "value",
		"key2": "value2",
		"key3": map[string]any{
			"key3": "value3",
			"key4": "value4",
		},
	})
	if err != nil {
		t.Error(err)
	}
	sbj := mSdk.Event().NewSubject("0xf86", "event_test", "v1", "list_req")
	name := "list_request_test_default"
	namespace := "default"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name: name,
			},
		},
	}

	// Act
	err = mSdk.Event().Publish(sbj, event, msg)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	events, err := mSdk.Client().ListEvents().Request(
		mir_v1.EventTarget{
			ObjectTarget: mir_v1.ObjectTarget{
				Namespaces: []string{
					namespace,
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	// TODO list with mir_v1.events
	testEvent := mir_v1.Event{}
	for _, event := range events {
		if event.Spec.RelatedObject.Meta.Name == name {
			testEvent = event
		}
	}

	// Assert
	assert.Equal(t, event.Message, testEvent.Spec.Message)
	assert.Equal(t, strings.Contains(testEvent.Meta.Name, name), true)
	assert.Equal(t, namespace, testEvent.Object.Meta.Namespace)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderTrigger], "pizza,toppings,test_eventstore-"), true)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderRoute], sbj.String()), true)
	assert.Equal(t, strings.Contains(testEvent.Object.Meta.Annotations[mir.HeaderSubject], sbj.GetId()), true)
}

func TestPublishListDeviceRequestWithEvents(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	s := swarm.NewSwarm(mSdk.Bus)
	_, err := s.AddDevice(&mir_apiv1.CreateDeviceRequest{
		Meta: &mir_apiv1.Meta{
			Namespace: "event_testing",
		},
		Spec: &mir_apiv1.DeviceSpec{
			DeviceId: "jam_n_butter",
		},
	}).Incubate()
	if err != nil {
		t.Error(err)
	}

	// Act
	time.Sleep(1 * time.Second)
	wgs, err := s.Deploy(ctx)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	dResp, err := mSdk.Client().ListDevice().Request(s.ToTarget(), true)

	// Assert
	assert.Equal(t, len(dResp), 1)
	assert.Equal(t, len(dResp[0].Status.Events) > 0, true)

	cancel()
	for _, wg := range wgs {
		wg.Wait()
	}
}

func TestPublishDeleteEventsRequest(t *testing.T) {
	// Arrange
	j, err := json.Marshal(map[string]any{
		"key":  "value",
		"key2": "value2",
		"key3": map[string]any{
			"key3": "value3",
			"key4": "value4",
		},
	})
	if err != nil {
		t.Error(err)
	}
	sbj := mSdk.Event().NewSubject("0xf86", "event_test", "v1", "list_req")
	name := "list_request_delete_test"
	namespace := "event_testing"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
	target := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Names: []string{
				name,
			},
			Namespaces: []string{
				namespace,
			},
		},
	}

	// Act
	err = mSdk.Event().Publish(sbj, event, msg)
	if err != nil {
		t.Error(err)
	}
	// Here we need bigger timer has the event srv
	// is processing so many events for the other tests
	time.Sleep(10 * time.Second)

	eventPresent, err := mSdk.Client().ListEvents().Request(target)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	eventDeleted, err := mSdk.Client().DeleteEvents().Request(target)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	eventGone, err := mSdk.Client().ListEvents().Request(target)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(eventPresent), 1)
	assert.Equal(t, strings.Contains(eventPresent[0].Meta.Name, name), true)
	assert.Equal(t, len(eventDeleted), 1)
	assert.Equal(t, strings.Contains(eventDeleted[0].Meta.Name, name), true)
	assert.Equal(t, len(eventGone), 0)
}
