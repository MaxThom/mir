package eventstore_srv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/swarm"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/maxthom/mir/pkgs/module/mir"
	"github.com/nats-io/nats.go"
	"github.com/surrealdb/surrealdb.go"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("eventstore")
var db *surrealdb.DB
var b *bus.BusConn
var sub *nats.Subscription
var mSdk *mir.Mir
var busUrl = "nats://127.0.0.1:4222"

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error

	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir_testing")
	b = test_utils.SetupNatsConPanic(busUrl)
	mSdk, err = mir.Connect("test_eventstore", busUrl)
	if err != nil {
		panic(err)
	}
	store := mng.NewSurrealMirStore(db)
	coreSrv, err := core_srv.NewCore(log, mSdk, store)
	if err := coreSrv.Serve(); err != nil {
		panic(err)
	}
	eventSrv, err := NewEventStore(log, mSdk, store)
	if err := eventSrv.Serve(); err != nil {
		panic(err)
	}
	fmt.Println(" -> bus")
	fmt.Println(" -> db")
	fmt.Println(" -> core")
	time.Sleep(1 * time.Second)
	// Clear data
	if _, err = store.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{
				"event_testing",
			},
		},
	}); err != nil {
		panic(err)
	}
	if _, err = store.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{
				"default",
			},
		},
	}); err != nil {
		panic(err)
	}
	if _, err = store.DeleteDevice(&core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.Targets{
			Namespaces: []string{
				"event_testing",
			},
		},
	}); err != nil {
		panic(err)
	}
	fmt.Println(" -> ready")

	// Tests
	exitVal := m.Run()

	// Teardown
	fmt.Println("Test Teardown")
	if _, err = store.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{
				"event_testing",
			},
		},
	}); err != nil {
		panic(err)
	}
	if _, err = store.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{
				"default",
			},
		},
	}); err != nil {
		panic(err)
	}
	if _, err = store.DeleteDevice(&core_apiv1.DeleteDeviceRequest{
		Targets: &core_apiv1.Targets{
			Namespaces: []string{
				"event_testing",
			},
		},
	}); err != nil {
		panic(err)
	}
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	b.Drain()
	coreSrv.Shutdown()
	eventSrv.Shutdown()
	b.Close()
	db.Close()
	fmt.Println(" -> core")
	fmt.Println(" -> nats")
	fmt.Println(" -> db")

	os.Exit(exitVal)
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
	sbj := mir.NewEventSubject("event_test", "v1", "list_req").WithId("0xf86")
	name := "list_request_test"
	namespace := "event_testing_store_normal"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_models.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_models.Meta{
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

	events, err := mSdk.Server().ListEvents().Request(
		mir_models.EventTarget{
			ObjectTarget: mir_models.ObjectTarget{
				Namespaces: []string{
					namespace,
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	// TODO list with mir_models.events
	testEvent := mir_models.Event{}
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
	sbj := mir.NewEventSubject("event_test", "v1", "list_req").WithId("0xf86")
	name := "list_request_test_default"
	namespace := "default"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_models.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_models.Meta{
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

	events, err := mSdk.Server().ListEvents().Request(
		mir_models.EventTarget{
			ObjectTarget: mir_models.ObjectTarget{
				Namespaces: []string{
					namespace,
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	// TODO list with mir_models.events
	testEvent := mir_models.Event{}
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
	s := swarm.NewSwarm(b)
	_, err := s.AddDevice(&core_apiv1.CreateDeviceRequest{
		Meta: &core_apiv1.Meta{
			Namespace: "event_testing",
		},
		Spec: &core_apiv1.Spec{
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

	dResp, err := mSdk.Server().ListDevice().Request(&core_apiv1.ListDeviceRequest{
		Targets:       s.ToTarget(),
		IncludeEvents: true,
	})

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
	sbj := mir.NewEventSubject("event_test", "v1", "list_req").WithId("0xf86")
	name := "list_request_delete_test"
	namespace := "event_testing"
	triggerChain := []string{"pizza", "toppings"}
	msg := mir.NewMsg(sbj.String())
	msg.AddToTriggerChain(triggerChain...)
	event := mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "device_online",
		Message: "device 'carrot' is online",
		Payload: j,
		RelatedObject: mir_models.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_models.Meta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}
	target := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
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
	time.Sleep(1 * time.Second)

	eventPresent, err := mSdk.Server().ListEvents().Request(target)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	eventDeleted, err := mSdk.Server().DeleteEvents().Request(target)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	eventGone, err := mSdk.Server().ListEvents().Request(target)
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
