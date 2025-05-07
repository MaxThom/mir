package eventstore_srv

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/externals/mng"
	bus "github.com/maxthom/mir/internal/libs/external/natsio"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/internal/servers/core_srv"
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
		Payload: map[string]any{
			"key1": "val1",
			"key2": map[string]any{
				"key3": "val3",
			},
		},
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/device",
			Meta: mir_models.Meta{
				Name:      name,
				Namespace: namespace,
			},
		},
	}

	// Act
	err := mSdk.Event().Publish(sbj, event, msg)
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
		Payload: map[string]any{
			"key1": "val1",
			"key2": map[string]any{
				"key3": "val3",
			},
		},
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/device",
			Meta: mir_models.Meta{
				Name: name,
			},
		},
	}

	// Act
	err := mSdk.Event().Publish(sbj, event, msg)
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
