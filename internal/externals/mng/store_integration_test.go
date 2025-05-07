package mng

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/surrealdb/surrealdb.go"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("eventstore")
var db *surrealdb.DB
var eventStore *surrealMirStore

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("Test Setup")
	var err error

	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir_testing")
	eventStore = NewSurrealMirStore(db)
	fmt.Println(" -> db")
	time.Sleep(1 * time.Second)
	// Clear data
	if _, err = eventStore.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Labels: map[string]string{
				"eventstore": "testing",
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
	if _, err = eventStore.DeleteEvent(mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Labels: map[string]string{
				"eventstore": "testing",
			},
		},
	}); err != nil {
		panic(err)
	}
	fmt.Println(" -> cleaned up")
	time.Sleep(1 * time.Second)
	db.Close()
	fmt.Println(" -> db")

	os.Exit(exitVal)
}

func TestPublishEventStoreCreateRequest(t *testing.T) {
	// Arrange
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "create_event",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	}).WithSpec(mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/devices",
			Meta: mir_models.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: map[string]any{
			"key":  "value",
			"key2": "value2",
			"key3": map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
	}).WithStatus(mir_models.EventStatus{
		Count:   1,
		FirstAt: time.Now().UTC(),
		LastAt:  time.Now().UTC(),
	})

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
}

func TestPublishEventStoreNotUnique(t *testing.T) {
	// Arrange
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "create_event_unique",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "create_event_unique",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = eventStore.CreateEvent(m2)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Assert(t, err != nil, true)
	assert.Assert(t, strings.Contains(err.Error(), "already exist"), true)
}

func TestPublishEventStoreListName(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Names: []string{"list_event_1", "list_event_2"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 2)
	assert.Equal(t, lResp[0].Meta.Name == m.Meta.Name || lResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, lResp[1].Meta.Name == m.Meta.Name || lResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreListNamespace(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{"events_list_test"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1_ns",
		Namespace: "events_list_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2_ns",
		Namespace: "events_list_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 2)
	assert.Equal(t, lResp[0].Meta.Name == m.Meta.Name || lResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, lResp[1].Meta.Name == m.Meta.Name || lResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreListLabels(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Labels: map[string]string{"test": "list_labels"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_labels",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_labels",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 2)
	assert.Equal(t, lResp[0].Meta.Name == m.Meta.Name || lResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, lResp[1].Meta.Name == m.Meta.Name || lResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreListLimit(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Labels: map[string]string{"test": "list_limit"},
		},
		Limit: 2,
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	})
	m3 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_3_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := eventStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, mResp3.Meta.Name, m3.Meta.Name)
	assert.Equal(t, len(lResp), 2)
}

func TestPublishEventStoreListDateNow(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{"store_test_time"},
		},
		DateFilter: mir_models.DateFilter{
			From: time.Date(2025, 05, 7, 0, 0, 0, 0, time.UTC),
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 7, 13, 0, 0, 0, time.UTC),
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC),
	})
	m3 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_3_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 5, 0, 0, 0, 0, time.UTC),
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := eventStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, mResp3.Meta.Name, m3.Meta.Name)
	assert.Equal(t, len(lResp), 2)
}

func TestPublishEventStoreListDateToFrom(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{"store_test_time_to_from"},
		},
		DateFilter: mir_models.DateFilter{
			From: time.Date(2025, 05, 7, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC),
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 7, 13, 0, 0, 0, time.UTC),
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC),
	})
	m3 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "list_event_3_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "list_limit",
		},
	}).WithStatus(mir_models.EventStatus{
		FirstAt: time.Date(2025, 05, 5, 0, 0, 0, 0, time.UTC),
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := eventStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, mResp3.Meta.Name, m3.Meta.Name)
	assert.Equal(t, len(lResp), 1)
}

func TestPublishEventStoreDeleteName(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Names: []string{"delete_event_1", "delete_event_2"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := eventStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 0)
	assert.Equal(t, dResp[0].Meta.Name == m.Meta.Name || dResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, dResp[1].Meta.Name == m.Meta.Name || dResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreDeleteNamespace(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Namespaces: []string{"events_delete_test"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_1_ns",
		Namespace: "events_delete_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_2_ns",
		Namespace: "events_delete_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := eventStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 0)
	assert.Equal(t, dResp[0].Meta.Name == m.Meta.Name || dResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, dResp[1].Meta.Name == m.Meta.Name || dResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreDeleteLabels(t *testing.T) {
	// Arrange
	tar := mir_models.EventTarget{
		ObjectTarget: mir_models.ObjectTarget{
			Labels: map[string]string{"test": "delete_labels"},
		},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_1_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "delete_labels",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "delete_event_2_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"test":       "delete_labels",
		},
	})
	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := eventStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := eventStore.ListEvent(tar)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, len(lResp), 0)
	assert.Equal(t, dResp[0].Meta.Name == m.Meta.Name || dResp[0].Meta.Name == m2.Meta.Name, true)
	assert.Equal(t, dResp[1].Meta.Name == m.Meta.Name || dResp[1].Meta.Name == m2.Meta.Name, true)
}

func TestPublishEventStoreUpdateMetaLblAnnoRequest(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_meta_lbl"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
			"key3":       "test",
		},
		Annotations: map[string]string{
			"eventstore": "testing",
			"key3":       "test",
		},
	}).WithSpec(mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/devices",
			Meta: mir_models.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: map[string]any{
			"key":  "value",
			"key2": "value2",
			"key3": map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
	}).WithStatus(mir_models.EventStatus{
		Count:   1,
		FirstAt: time.Now().UTC(),
		LastAt:  time.Now().UTC(),
	})
	upd := mir_models.EventUpdate{
		Meta: &mir_models.MetaUpdate{
			Labels: map[string]*string{
				"caca_mou": strPtr("bien_mou"),
				"key3":     nil,
			},
			Annotations: map[string]*string{
				"caca_mou": strPtr("bien_mou"),
				"key3":     nil,
			},
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := eventStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Meta.Labels["eventstore"], m.Meta.Labels["eventstore"])
	assert.Equal(t, uResp[0].Meta.Labels["caca_mou"], *upd.Meta.Labels["caca_mou"])
	_, ok := uResp[0].Meta.Labels["key3"]
	assert.Equal(t, false, ok)
	assert.Equal(t, uResp[0].Meta.Annotations["eventstore"], m.Meta.Annotations["eventstore"])
	assert.Equal(t, uResp[0].Meta.Annotations["caca_mou"], *upd.Meta.Annotations["caca_mou"])
	_, ok = uResp[0].Meta.Annotations["key3"]
	assert.Equal(t, false, ok)
}

func TestPublishEventStoreUpdateNameRequest(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_meta_name"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_name",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	}).WithSpec(mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/devices",
			Meta: mir_models.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: map[string]any{
			"key":  "value",
			"key2": "value2",
			"key3": map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
	}).WithStatus(mir_models.EventStatus{
		Count:   1,
		FirstAt: time.Now().UTC(),
		LastAt:  time.Now().UTC(),
	})
	upd := mir_models.EventUpdate{
		Meta: &mir_models.MetaUpdate{
			Name: strPtr("update_event_new_name"),
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := eventStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Meta.Name, *upd.Meta.Name)
}

func TestPublishEventStoreUpdateNameRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_meta_name_1"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_name_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_name_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	upd := mir_models.EventUpdate{
		Meta: &mir_models.MetaUpdate{
			Name: strPtr("update_event_meta_name_2"),
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = eventStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "is already in use in namespace"))
}

func TestPublishEventStoreUpdateNamespaceRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_meta_namespace_1"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_namespace_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_namespace_1",
		Namespace: "test_ns",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	upd := mir_models.EventUpdate{
		Meta: &mir_models.MetaUpdate{
			Namespace: strPtr("test_ns"),
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = eventStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "cannot update object as multiple device will have the same name"))
}

func TestPublishEventStoreUpdateNameNamespaceRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_meta_namens_1"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_namens_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	m2 := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_meta_namens_2",
		Namespace: "test_ns",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	})
	upd := mir_models.EventUpdate{
		Meta: &mir_models.MetaUpdate{
			Name:      strPtr("update_event_meta_namens_2"),
			Namespace: strPtr("test_ns"),
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = eventStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = eventStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "cannot update object has"))
}

func TestPublishEventStoreUpdateSpecRequest(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_spec"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_spec",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	}).WithSpec(mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/devices",
			Meta: mir_models.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: map[string]any{
			"key":  "value",
			"key2": "value2",
			"key3": map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
	}).WithStatus(mir_models.EventStatus{
		Count:   1,
		FirstAt: time.Now().UTC(),
		LastAt:  time.Now().UTC(),
	})
	upd := mir_models.EventUpdate{
		Spec: &mir_models.EventUpdateSpec{
			Type:    strPtr(mir_models.EventTypeWarning),
			Reason:  strPtr("pizza"),
			Message: strPtr("test_de_la_mort"),
			Payload: map[string]any{
				"caca_mou": "bien_mou",
				"key3":     nil,
			},
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := eventStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Spec.Reason, *upd.Spec.Reason)
	assert.Equal(t, uResp[0].Spec.Message, *upd.Spec.Message)
	assert.Equal(t, uResp[0].Spec.Type, *upd.Spec.Type)
	assert.Equal(t, uResp[0].Spec.Payload["caca_mou"], upd.Spec.Payload["caca_mou"])
	_, ok := uResp[0].Spec.Payload["key3"]
	assert.Equal(t, false, ok)
}

func TestPublishEventStoreUpdateStatusRequest(t *testing.T) {
	// Arrange
	tar := mir_models.ObjectTarget{
		Names: []string{"update_event_status"},
	}
	m := mir_models.NewEvent().WithMeta(mir_models.Meta{
		Name:      "update_event_status",
		Namespace: "store_test",
		Labels: map[string]string{
			"eventstore": "testing",
		},
	}).WithSpec(mir_models.EventSpec{
		Type:    mir_models.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_models.Object{
			ApiVersion: "v1alpha",
			ApiName:    "mir/devices",
			Meta: mir_models.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: map[string]any{
			"key":  "value",
			"key2": "value2",
			"key3": map[string]any{
				"key3": "value3",
				"key4": "value4",
			},
		},
	}).WithStatus(mir_models.EventStatus{
		Count:   1,
		FirstAt: time.Now().UTC(),
		LastAt:  time.Now().UTC(),
	})
	upd := mir_models.EventUpdate{
		Status: &mir_models.EventUpdateStatus{
			Count:   intPtr(3),
			FirstAt: timePtr(time.Date(2014, 10, 14, 5, 5, 5, 5, time.UTC)),
			LastAt:  timePtr(time.Date(2014, 10, 14, 5, 5, 5, 5, time.UTC)),
		},
	}

	// Act
	mResp, err := eventStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := eventStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Status.Count, *upd.Status.Count)
	assert.Equal(t, uResp[0].Status.LastAt, *upd.Status.LastAt)
	assert.Equal(t, uResp[0].Status.FirstAt, *upd.Status.FirstAt)
}

func strPtr(s string) *string {
	return &s
}

func timePtr(s time.Time) *time.Time {
	return &s
}

func intPtr(s int) *int {
	return &s
}
