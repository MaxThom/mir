package mng

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"gotest.tools/assert"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestDashboard(name, namespace string) mir_v1.Dashboard {
	return mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      name,
		Namespace: namespace,
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestDashboardCreate(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_create", "dashtest")

	// Act
	resp, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, resp.Meta.Name, d.Meta.Name)
	assert.Equal(t, resp.Meta.Namespace, d.Meta.Namespace)
	assert.Equal(t, resp.ApiVersion, d.ApiVersion)
	assert.Equal(t, resp.Kind, d.Kind)
	assert.Equal(t, resp.Meta.Labels["mirstore"], "testing")
	assert.Assert(t, !resp.Status.CreatedAt.IsZero())
	assert.Assert(t, !resp.Status.UpdatedAt.IsZero())
}

func TestDashboardCreateWithWidgets(t *testing.T) {
	// Arrange
	ri := 30
	tm := 60
	d := newTestDashboard("dash_create_widgets", "dashtest").WithSpec(mir_v1.DashboardSpec{
		Description:     "a dashboard with widgets",
		RefreshInterval: &ri,
		TimeMinutes:     &tm,
		Widgets: []mir_v1.DashboardWidget{
			{
				Id:    "w1",
				Type:  mir_v1.WidgetTypeTelemetry,
				Title: "Telemetry Widget",
				X:     0, Y: 0, W: 4, H: 3,
				Config: mir_v1.TelemetryWidgetConfig{
					Measurement: "temperature",
					Fields:      []string{"value"},
					TimeRange:   "1h",
				},
			},
			{
				Id:    "w2",
				Type:  mir_v1.WidgetTypeCommand,
				Title: "Command Widget",
				X:     4, Y: 0, W: 4, H: 3,
				Config: mir_v1.CommandWidgetConfig{},
			},
		},
	})

	// Act
	resp, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, resp.Meta.Name, d.Meta.Name)
	assert.Equal(t, resp.Spec.Description, "a dashboard with widgets")
	assert.Equal(t, *resp.Spec.RefreshInterval, ri)
	assert.Equal(t, *resp.Spec.TimeMinutes, tm)
	assert.Equal(t, len(resp.Spec.Widgets), 2)
	assert.Equal(t, resp.Spec.Widgets[0].Id, "w1")
	assert.Equal(t, resp.Spec.Widgets[1].Id, "w2")
}

func TestDashboardCreateAlreadyExist(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_create_dup", "dashtest")

	// Act
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d)

	// Assert
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "already exist"))
}

func TestDashboardCreateAutoTimestamps(t *testing.T) {
	// Arrange
	before := time.Now().UTC().Add(-time.Second)
	d := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_create_timestamps",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing"},
	}).WithSpec(mir_v1.DashboardSpec{})
	// Strip timestamps to verify they are auto-set
	d.Status.CreatedAt = time.Time{}
	d.Status.UpdatedAt = time.Time{}

	// Act
	resp, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Assert(t, resp.Status.CreatedAt.After(before))
	assert.Assert(t, resp.Status.UpdatedAt.After(before))
}

func TestDashboardCreateEmptyWidgetsList(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_create_empty_widgets", "dashtest")

	// Act
	resp, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	if resp.Spec.Widgets != nil {
		assert.Equal(t, len(resp.Spec.Widgets), 0)
	} else {
		assert.Assert(t, resp.Spec.Widgets == nil)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestDashboardListByName(t *testing.T) {
	// Arrange
	d1 := newTestDashboard("dash_list_name_1", "dashtest")
	d2 := newTestDashboard("dash_list_name_2", "dashtest")
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Names: []string{"dash_list_name_1", "dash_list_name_2"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 2)
	names := []string{resp[0].Meta.Name, resp[1].Meta.Name}
	assert.Assert(t, strings.Contains(strings.Join(names, ","), "dash_list_name_1"))
	assert.Assert(t, strings.Contains(strings.Join(names, ","), "dash_list_name_2"))
}

func TestDashboardListByNamespace(t *testing.T) {
	// Arrange
	d1 := newTestDashboard("dash_list_ns_1", "dashtest_ns")
	d2 := newTestDashboard("dash_list_ns_2", "dashtest_ns")
	d1.Meta.Labels["mirstore"] = "testing"
	d2.Meta.Labels["mirstore"] = "testing"
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Namespaces: []string{"dashtest_ns"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 2)
}

func TestDashboardListByLabel(t *testing.T) {
	// Arrange
	d1 := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_list_lbl_1",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing", "env": "list_label_test"},
	})
	d2 := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_list_lbl_2",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing", "env": "list_label_test"},
	})
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Labels: map[string]string{"env": "list_label_test"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 2)
}

func TestDashboardListSingleByName(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_list_single", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Names: []string{"dash_list_single"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Meta.Name, "dash_list_single")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDashboardDeleteByName(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_delete_name", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Act
	deleted, err := mirStore.DeleteDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_delete_name"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(deleted), 1)
	assert.Equal(t, deleted[0].Meta.Name, "dash_delete_name")

	remaining, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Names: []string{"dash_delete_name"},
	})
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, len(remaining), 0)
}

func TestDashboardDeleteByNamespace(t *testing.T) {
	// Arrange
	d1 := newTestDashboard("dash_delete_ns_1", "dashtest_delete_ns")
	d2 := newTestDashboard("dash_delete_ns_2", "dashtest_delete_ns")
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}

	// Act
	deleted, err := mirStore.DeleteDashboard(mir_v1.ObjectTarget{
		Namespaces: []string{"dashtest_delete_ns"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(deleted), 2)

	remaining, err := mirStore.ListDashboards(mir_v1.ObjectTarget{
		Namespaces: []string{"dashtest_delete_ns"},
	})
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, len(remaining), 0)
}

func TestDashboardDeleteByLabel(t *testing.T) {
	// Arrange
	d1 := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_delete_lbl_1",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing", "env": "delete_label_test"},
	})
	d2 := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_delete_lbl_2",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing", "env": "delete_label_test"},
	})
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}

	// Act
	deleted, err := mirStore.DeleteDashboard(mir_v1.ObjectTarget{
		Labels: map[string]string{"env": "delete_label_test"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(deleted), 2)
}

func TestDashboardDeleteNoTarget(t *testing.T) {
	// Act
	_, err := mirStore.DeleteDashboard(mir_v1.ObjectTarget{})

	// Assert
	assert.ErrorContains(t, err, "No device target provided")
}

func TestDashboardDeleteReturnsDeletedDashboards(t *testing.T) {
	// Arrange
	ri := 15
	d := newTestDashboard("dash_delete_return", "dashtest").WithSpec(mir_v1.DashboardSpec{
		Description:     "delete me",
		RefreshInterval: &ri,
	})
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Act
	deleted, err := mirStore.DeleteDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_delete_return"},
	})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(deleted), 1)
	assert.Equal(t, deleted[0].Spec.Description, "delete me")
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestDashboardUpdateMetaLabels(t *testing.T) {
	// Arrange
	d := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:      "dash_upd_meta_lbl",
		Namespace: "dashtest",
		Labels:    map[string]string{"mirstore": "testing", "old_key": "old_val"},
	})
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Meta: &mir_v1.MetaUpdate{
			Labels: map[string]*string{
				"new_key": strPtr("new_val"),
				"old_key": nil,
			},
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_meta_lbl"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Meta.Labels["new_key"], "new_val")
	_, ok := resp[0].Meta.Labels["old_key"]
	assert.Equal(t, false, ok)
}

func TestDashboardUpdateMetaAnnotations(t *testing.T) {
	// Arrange
	d := mir_v1.NewDashboard().WithMeta(mir_v1.Meta{
		Name:        "dash_upd_meta_anno",
		Namespace:   "dashtest",
		Labels:      map[string]string{"mirstore": "testing"},
		Annotations: map[string]string{"info": "original"},
	})
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Meta: &mir_v1.MetaUpdate{
			Annotations: map[string]*string{
				"info":    strPtr("updated"),
				"new_ann": strPtr("added"),
			},
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_meta_anno"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Meta.Annotations["info"], "updated")
	assert.Equal(t, resp[0].Meta.Annotations["new_ann"], "added")
}

func TestDashboardUpdateMetaName(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_name_old", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Meta: &mir_v1.MetaUpdate{
			Name: strPtr("dash_upd_name_new"),
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_name_old"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Meta.Name, "dash_upd_name_new")
}

func TestDashboardUpdateSpecDescription(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_spec_desc", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Spec: &mir_v1.DashboardUpdateSpec{
			Description: strPtr("updated description"),
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_spec_desc"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Spec.Description, "updated description")
}

func TestDashboardUpdateSpecRefreshInterval(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_spec_ri", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Spec: &mir_v1.DashboardUpdateSpec{
			RefreshInterval: intPtr(60),
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_spec_ri"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, *resp[0].Spec.RefreshInterval, 60)
}

func TestDashboardUpdateSpecTimeMinutes(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_spec_tm", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Spec: &mir_v1.DashboardUpdateSpec{
			TimeMinutes: intPtr(120),
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_spec_tm"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, *resp[0].Spec.TimeMinutes, 120)
}

func TestDashboardUpdateSpecWidgets(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_spec_widgets", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	widgets := []mir_v1.DashboardWidget{
		{Id: "w1", Type: mir_v1.WidgetTypeTelemetry, Title: "T1", X: 0, Y: 0, W: 4, H: 3},
		{Id: "w2", Type: mir_v1.WidgetTypeEvents, Title: "E1", X: 4, Y: 0, W: 4, H: 3},
	}
	upd := mir_v1.DashboardUpdate{
		Spec: &mir_v1.DashboardUpdateSpec{
			Widgets: widgets,
		},
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_spec_widgets"},
	}, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, len(resp[0].Spec.Widgets), 2)
	assert.Equal(t, resp[0].Spec.Widgets[0].Id, "w1")
	assert.Equal(t, resp[0].Spec.Widgets[1].Id, "w2")
}

func TestDashboardUpdateNoTarget(t *testing.T) {
	// Act
	_, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{}, mir_v1.DashboardUpdate{})

	// Assert
	assert.ErrorContains(t, err, "No device target provided")
}

func TestDashboardUpdateNameDuplicate(t *testing.T) {
	// Arrange
	d1 := newTestDashboard("dash_upd_dup_1", "dashtest")
	d2 := newTestDashboard("dash_upd_dup_2", "dashtest")
	_, err := mirStore.CreateDashboard(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDashboard(d2)
	if err != nil {
		t.Error(err)
	}
	upd := mir_v1.DashboardUpdate{
		Meta: &mir_v1.MetaUpdate{
			Name: strPtr("dash_upd_dup_2"),
		},
	}
	time.Sleep(500 * time.Millisecond)

	// Act
	_, err = mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_dup_1"},
	}, upd)

	// Assert
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "is already in use"))
}

func TestDashboardUpdateEmptyUpdateReturnsCurrentState(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_upd_empty", "dashtest")
	created, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.UpdateDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_upd_empty"},
	}, mir_v1.DashboardUpdate{})
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Meta.Name, created.Meta.Name)
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func TestDashboardMergeBasic(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_merge_basic", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	patch, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"description": "merged description",
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.MergeDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_merge_basic"},
	}, patch, MergePatch)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, resp[0].Spec.Description, "merged description")
}

func TestDashboardMergeRefreshAndTime(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_merge_ri_tm", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	patch, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"refreshInterval": 45,
			"timeMinutes":     90,
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Act
	resp, err := mirStore.MergeDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_merge_ri_tm"},
	}, patch, MergePatch)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(resp), 1)
	assert.Equal(t, *resp[0].Spec.RefreshInterval, 45)
	assert.Equal(t, *resp[0].Spec.TimeMinutes, 90)
}

func TestDashboardMergeInvalidJson(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_merge_invalid_json", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = mirStore.MergeDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_merge_invalid_json"},
	}, []byte(`{"unknown_field": "value"}`), MergePatch)

	// Assert
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "unknown fields"))
}

func TestDashboardMergeNoTarget(t *testing.T) {
	// Arrange
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"description": "x"}})

	// Act
	_, err := mirStore.MergeDashboard(mir_v1.ObjectTarget{}, patch, MergePatch)

	// Assert
	assert.ErrorContains(t, err, "No device target provided")
}

func TestDashboardMergeUnsupportedOperation(t *testing.T) {
	// Arrange
	d := newTestDashboard("dash_merge_unsupported_op", "dashtest")
	_, err := mirStore.CreateDashboard(d)
	if err != nil {
		t.Error(err)
	}
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"description": "x"}})

	// Act
	_, err = mirStore.MergeDashboard(mir_v1.ObjectTarget{
		Names: []string{"dash_merge_unsupported_op"},
	}, patch, JsonPatch)

	// Assert
	assert.Assert(t, err != nil)
}
