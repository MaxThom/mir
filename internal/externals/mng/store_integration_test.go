package mng

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/internal/libs/jsonyaml"
	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/pkgs/mir_v1"
	surrealdbModels "github.com/surrealdb/surrealdb.go/pkg/models"
	"gotest.tools/assert"
)

var log = test_utils.TestLogger("mirstore")
var db *surreal.AutoReconnDB
var mirStore *surrealMirStore

func TestMain(m *testing.M) {
	// Setup
	fmt.Println("> Test Setup")
	db = test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir_testing")
	mirStore = NewSurrealMirStore(db)
	if err := dataCleanUp(); err != nil {
		panic(err)
	}
	time.Sleep(1 * time.Second)

	// Tests
	fmt.Println("> Test Run")
	exitVal := m.Run()
	time.Sleep(1 * time.Second)

	// Teardown
	fmt.Println("> Test Teardown")
	db.Close()
	time.Sleep(1 * time.Second)

	os.Exit(exitVal)
}

func dataCleanUp() error {
	if _, err := mirStore.DeleteEvent(mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Labels: map[string]string{
				"mirstore": "testing",
			},
		},
	}); err != nil {
		return err
	}
	if _, err := mirStore.DeleteDevice(mir_v1.DeviceTarget{
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}); err != nil {
		return err
	}

	return nil
}

func TestPublishStoreDeviceCreate(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "create_dev",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev",
		Disabled: boolPtr(true),
	})
	// Act
	dResp, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, d.ApiVersion, dResp.ApiVersion)
	assert.Equal(t, d.Kind, dResp.Kind)
	assert.Equal(t, d.Meta.Name, dResp.Meta.Name)
	assert.Equal(t, d.Meta.Namespace, dResp.Meta.Namespace)
	assert.Equal(t, d.Meta.Labels["mirstore"], dResp.Meta.Labels["mirstore"])
	assert.Equal(t, d.Meta.Annotations["info"], dResp.Meta.Annotations["info"])
	assert.Equal(t, d.Spec.DeviceId, dResp.Spec.DeviceId)
	assert.Equal(t, *d.Spec.Disabled, *dResp.Spec.Disabled)
}

func TestPublishStoreDeviceCreateMissingNameNamespace(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev_missing_namens",
	})
	// Act
	dResp, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, d.ApiVersion, dResp.ApiVersion)
	assert.Equal(t, d.Kind, dResp.Kind)
	assert.Equal(t, d.Spec.DeviceId, dResp.Meta.Name)
	assert.Equal(t, "default", dResp.Meta.Namespace)
}

func TestPublishStoreDeviceCreateMissingId(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{})
	// Act
	_, err := mirStore.CreateDevice(d)

	// Assert
	assert.ErrorContains(t, err, "device id is missing")
}

func TestPublishStoreDeviceCreateAlreadyExistId(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev_already_exist_id",
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore2",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev_already_exist_id",
	})
	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)

	// Assert
	assert.ErrorContains(t, err, "device create_dev_already_exist_id/mirstore2 with deviceId create_dev_already_exist_id already exist")
}

func TestPublishStoreDeviceCreateAlreadyExistNameNs(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "already_exist_name",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev_already_exist_name_0",
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "already_exist_name",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "create_dev_already_exist_name_1",
	})
	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)

	// Assert
	assert.ErrorContains(t, err, "device already_exist_name/mirstore with deviceId create_dev_already_exist_name_1 already exist")
}

func TestPublishStoreDeviceCreateSameNameDifferentNs(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "same_name_diff_ns",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "same_name_diff_ns_0",
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "same_name_diff_ns",
		Namespace: "mirstore2",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "same_name_diff_ns_1",
	})
	// Act
	_, err0 := mirStore.CreateDevice(d0)
	_, err1 := mirStore.CreateDevice(d1)

	// Assert
	assert.NilError(t, err0)
	assert.NilError(t, err1)
}

func TestPublishStoreDeviceListByIds(t *testing.T) {
	// Arrange
	ids := []string{"list_dev_0", "list_dev_1"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	l := mir_v1.DeviceTarget{
		Ids: ids,
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(lResp), 2)
}

func TestPublishStoreDeviceListByName(t *testing.T) {
	// Arrange
	ids := []string{"list_dev_0_name", "list_dev_1_name", "list_dev_2_name"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[2],
	})
	l := mir_v1.DeviceTarget{
		Names: ids[:2],
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(lResp), 2)
}

func TestPublishStoreDeviceListByLabel(t *testing.T) {
	// Arrange
	ids := []string{"list_dev_0_lbl", "list_dev_1_lbl", "list_dev_2_lbl"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"lbl":      "yiha",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"lbl":      "yiha",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[2],
	})
	l := mir_v1.DeviceTarget{
		Labels: map[string]string{
			"lbl": "yiha",
		},
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(lResp), 2)
}

func TestPublishStoreDeviceListByNamespace(t *testing.T) {
	// Arrange
	ids := []string{"list_dev_0_namespace", "list_dev_1_namespace"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore_list",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore_list",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	l := mir_v1.DeviceTarget{
		Namespaces: []string{"mirstore_list"},
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(lResp), 2)
}

func TestPublishStoreDeviceDeleteByIds(t *testing.T) {
	// Arrange
	ids := []string{"del_dev_0", "del_dev_1"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	l := mir_v1.DeviceTarget{
		Ids: ids,
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteDevice(l)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dResp), 2)
	assert.Equal(t, len(lResp), 0)
}

func TestPublishStoreDeviceDeleteByName(t *testing.T) {
	// Arrange
	ids := []string{"del_dev_0_name", "del_dev_1_name", "del_dev_2_name"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[2],
	})
	l := mir_v1.DeviceTarget{
		Names: ids[:2],
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteDevice(l)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dResp), 2)
	assert.Equal(t, len(lResp), 0)
}

func TestPublishStoreDeviceDeleteByLabel(t *testing.T) {
	// Arrange
	ids := []string{"del_dev_0_lbl", "del_dev_1_lbl", "del_dev_2_lbl"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"lbl":      "yiha_del",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"lbl":      "yiha_del",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[2],
	})
	l := mir_v1.DeviceTarget{
		Labels: map[string]string{
			"lbl": "yiha_del",
		},
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteDevice(l)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dResp), 2)
	assert.Equal(t, len(lResp), 0)
}

func TestPublishStoreDeviceDeleteByNamespace(t *testing.T) {
	// Arrange
	ids := []string{"del_dev_0_namespace", "del_dev_1_namespace"}
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore_del",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[0],
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore_del",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: ids[1],
	})
	l := mir_v1.DeviceTarget{
		Namespaces: []string{"mirstore_del"},
	}

	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteDevice(l)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(l, false)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(dResp), 2)
	assert.Equal(t, len(lResp), 0)
}

func TestPublishStoreDeviceUpdateMeta(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev",
		Disabled: boolPtr(true),
	})
	// Act
	_, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	d.Meta.Name = "update_dev_post"
	d.Meta.Namespace = "mirstore_post"
	d.Meta.Labels["test"] = "pizza"
	d.Meta.Labels["test2"] = ""
	d.Meta.Labels["food"] = "gross"
	d.Meta.Annotations["food"] = "gross"
	d.Meta.Annotations["info"] = ""
	uResp, err := mirStore.UpdateDevice(mir_v1.DeviceTarget{Ids: []string{d.Spec.DeviceId}}, d)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, d.ApiVersion, uResp[0].ApiVersion)
	assert.Equal(t, d.Kind, uResp[0].Kind)
	assert.Equal(t, d.Meta.Name, uResp[0].Meta.Name)
	assert.Equal(t, d.Meta.Namespace, uResp[0].Meta.Namespace)
	assert.Equal(t, d.Meta.Labels["mirstore"], uResp[0].Meta.Labels["mirstore"])
	assert.Equal(t, d.Meta.Labels["test"], uResp[0].Meta.Labels["test"])
	assert.Equal(t, d.Meta.Labels["food"], uResp[0].Meta.Labels["food"])
	_, ok := uResp[0].Meta.Labels["test2"]
	assert.Equal(t, false, ok)
	assert.Equal(t, d.Meta.Annotations["food"], uResp[0].Meta.Annotations["food"])
	_, ok = uResp[0].Meta.Annotations["info"]
	assert.Equal(t, false, ok)
}

func TestPublishStoreDeviceUpdateSpec(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_spec",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_spec",
		Disabled: boolPtr(true),
	})
	dUpd := mir_v1.NewDevice().WithSpec(mir_v1.DeviceSpec{
		DeviceId: "bob",
		Disabled: boolPtr(false),
	})
	// Act
	_, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}
	uResp, err := mirStore.UpdateDevice(d.ToTarget(), dUpd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, "bob", uResp[0].Spec.DeviceId)
	assert.Equal(t, false, *uResp[0].Spec.Disabled)
}

func TestPublishStoreDeviceUpdateProps(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_props",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_props",
	})
	dUpd := mir_v1.NewDevice()
	dUpd.Properties.Desired = map[string]any{
		"test": "pizza",
		"test2": map[string]any{
			"inner": true,
		},
	}
	dUpd.Properties.Reported = map[string]any{
		"test": "pizza",
		"test2": map[string]any{
			"inner": []int{1, 2, 3},
		},
	}
	dUpd2 := mir_v1.NewDevice()
	dUpd2.Properties.Desired = map[string]any{
		"test":  "bob",
		"test2": nil,
		"test3": "bobby",
	}
	dUpd2.Properties.Reported = map[string]any{
		"test":  "bob",
		"test3": "bobby",
		"test2": map[string]any{
			"inner": []int{1, 2, 3, 4, 5},
		},
	}

	// Act
	_, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}
	uResp1, err := mirStore.UpdateDevice(d.ToTarget(), dUpd)
	if err != nil {
		t.Error(err)
	}
	uResp2, err := mirStore.UpdateDevice(d.ToTarget(), dUpd2)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, uResp1[0].Properties.Desired["test"], "pizza")
	assert.Equal(t, uResp1[0].Properties.Desired["test2"].(map[string]any)["inner"], true)
	assert.Equal(t, uResp2[0].Properties.Desired["test"], "bob")
	assert.Equal(t, uResp2[0].Properties.Desired["test3"], "bobby")
	_, ok := uResp2[0].Properties.Desired["test2"]
	assert.Equal(t, false, ok)

	assert.Equal(t, uResp1[0].Properties.Reported["test"], "pizza")
	assert.Equal(t, len(uResp1[0].Properties.Reported["test2"].(map[string]any)["inner"].([]any)), 3)
	assert.Equal(t, uResp2[0].Properties.Reported["test"], "bob")
	assert.Equal(t, uResp2[0].Properties.Reported["test3"], "bobby")
	assert.Equal(t, len(uResp2[0].Properties.Reported["test2"].(map[string]any)["inner"].([]any)), 5)
}

func TestPublishStoreDeviceUpdateStatus(t *testing.T) {
	// Arrange
	d := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_st",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_st",
	})
	dUpd := mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
		Online:         boolPtr(true),
		LastHearthbeat: surrealTimePtr(time.Date(1992, 10, 14, 14, 0, 0, 0, time.UTC)),
		Schema: mir_v1.Schema{
			CompressedSchema: []byte{0x12},
			PackageNames:     []string{"bob"},
			LastSchemaFetch:  surrealTimePtr(time.Date(1992, 10, 14, 14, 0, 0, 0, time.UTC)),
		},
		Properties: mir_v1.PropertiesTime{
			Desired: map[string]surrealdbModels.CustomDateTime{
				"test": {Time: time.Date(1992, 10, 14, 14, 0, 0, 0, time.UTC)},
			},
			Reported: map[string]surrealdbModels.CustomDateTime{
				"test": {Time: time.Date(1992, 10, 14, 14, 0, 0, 0, time.UTC)},
			},
		},
	})
	dUpd2 := mir_v1.NewDevice().WithStatus(mir_v1.DeviceStatus{
		Properties: mir_v1.PropertiesTime{
			Desired: map[string]surrealdbModels.CustomDateTime{
				"test": {},
			},
			Reported: map[string]surrealdbModels.CustomDateTime{
				"test": {},
			},
		},
	})

	// Act
	_, err := mirStore.CreateDevice(d)
	if err != nil {
		t.Error(err)
	}
	uResp, err := mirStore.UpdateDevice(d.ToTarget(), dUpd)
	if err != nil {
		t.Error(err)
	}
	uResp1, err := mirStore.UpdateDevice(d.ToTarget(), dUpd2)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, dUpd.Status.LastHearthbeat.Time.UTC(), uResp[0].Status.LastHearthbeat.UTC())
	assert.Equal(t, *dUpd.Status.Online, *uResp[0].Status.Online)
	assert.Equal(t, len(dUpd.Status.Schema.CompressedSchema), len(uResp[0].Status.Schema.CompressedSchema))
	assert.Equal(t, dUpd.Status.Schema.PackageNames[0], uResp[0].Status.Schema.PackageNames[0])
	assert.Equal(t, dUpd.Status.Schema.LastSchemaFetch.UTC(), uResp[0].Status.Schema.LastSchemaFetch.UTC())
	assert.Equal(t, dUpd.Status.Properties.Desired["test"].UTC(), uResp[0].Status.Properties.Desired["test"].UTC())
	assert.Equal(t, dUpd.Status.Properties.Reported["test"].UTC(), uResp[0].Status.Properties.Reported["test"].UTC())
	_, ok := uResp1[0].Status.Properties.Desired["test"]
	assert.Equal(t, false, ok)
	_, ok = uResp1[0].Status.Properties.Reported["test"]
	assert.Equal(t, false, ok)
}

func TestPublishStoreDeviceUpdateMetaNameAlreadyExist(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_al0",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_al0",
		Disabled: boolPtr(true),
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_al1",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_al1",
		Disabled: boolPtr(true),
	})
	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	d0.Meta.Name = d1.Meta.Name
	_, err = mirStore.UpdateDevice(d0.ToTarget(), d0)

	// Assert
	assert.ErrorContains(t, err, "")
}

func TestPublishStoreDeviceUpdateMetaIdAlreadyExist(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id0",
		Disabled: boolPtr(true),
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id1",
		Disabled: boolPtr(true),
	})
	// Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	d0.Spec.DeviceId = d1.Spec.DeviceId
	_, err = mirStore.UpdateDevice(mir_v1.DeviceTarget{Ids: []string{d0.Spec.DeviceId}}, d0)

	// Assert
	assert.ErrorContains(t, err, "")
}

func TestPublishStoreDeviceUpdateMetaMultipleOnlyNamespace(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_id0_onlyname",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id0_onlyname_1",
		Disabled: boolPtr(true),
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_id0_onlyname",
		Namespace: "mirstore2",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id0_onlyname_2",
		Disabled: boolPtr(true),
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Namespace: "mirstore3",
	}) // Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.UpdateDevice(mir_v1.ToTargets(d0, d1), d2)

	// Assert
	assert.ErrorContains(t, err, "cannot update device as multiple device will have the same name 'update_dev_id0_onlyname' in namespace 'mirstore3'")
}

func TestPublishStoreDeviceUpdateMetaMultipleOnlyName(t *testing.T) {
	// Arrange
	d0 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_id0_onlynamens",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id0_onlynamens_1",
		Disabled: boolPtr(true),
	})
	d1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "update_dev_id1_onlynamens",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "hotdog",
			"test2":    "ham",
		},
		Annotations: map[string]string{
			"info": "supra",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "update_dev_id0_onlynamens_2",
		Disabled: boolPtr(true),
	})
	d2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name: "clash_name_b",
	}) // Act
	_, err := mirStore.CreateDevice(d0)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateDevice(d1)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.UpdateDevice(mir_v1.ToTargets(d0, d1), d2)

	// Assert
	assert.ErrorContains(t, err, "cannot update device as multiple device will have the same name 'clash_name_b' in namespace 'mirstore'")
}

func TestMergeDeviceBasic(t *testing.T) {
	// Arrange
	device := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_basic",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
		Annotations: map[string]string{
			"info": "original",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_basic",
		Disabled: boolPtr(false),
	})

	// Create initial device
	_, err := mirStore.CreateDevice(device)
	if err != nil {
		t.Error(err)
	}

	// Prepare merge patch
	mergePatch := map[string]interface{}{
		"meta": map[string]interface{}{
			"labels": map[string]string{
				"test":     "updated",
				"new_test": "added",
			},
			"annotations": map[string]string{
				"info":     "updated",
				"new_info": "added",
			},
		},
	}
	patchBytes, err := json.Marshal(mergePatch)
	if err != nil {
		t.Error(err)
	}

	// Act
	mergedDevices, err := mirStore.MergeDevice(device.ToTarget(), patchBytes, MergePatch)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(mergedDevices), 1)
	assert.Equal(t, "merge_device_basic", mergedDevices[0].Meta.Name)
	assert.Equal(t, "mirstore", mergedDevices[0].Meta.Namespace)
	assert.Equal(t, "updated", mergedDevices[0].Meta.Labels["test"])
	assert.Equal(t, "added", mergedDevices[0].Meta.Labels["new_test"])
	assert.Equal(t, "updated", mergedDevices[0].Meta.Annotations["info"])
	assert.Equal(t, "added", mergedDevices[0].Meta.Annotations["new_info"])
	assert.Equal(t, "merge_device_basic", mergedDevices[0].Spec.DeviceId)
	assert.Equal(t, false, *mergedDevices[0].Spec.Disabled)
}

func TestMergeDeviceSpec(t *testing.T) {
	// Arrange
	device := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_spec",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_spec",
		Disabled: boolPtr(false),
	})

	// Create initial device
	_, err := mirStore.CreateDevice(device)
	if err != nil {
		t.Fatal(err)
	}

	// Prepare merge patch
	mergePatch := map[string]interface{}{
		"spec": map[string]interface{}{
			"disabled": true,
		},
	}
	patchBytes, err := json.Marshal(mergePatch)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	mergedDevices, err := mirStore.MergeDevice(device.ToTarget(), patchBytes, MergePatch)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(mergedDevices), 1)
	assert.Equal(t, "merge_device_spec", mergedDevices[0].Meta.Name)
	assert.Equal(t, true, *mergedDevices[0].Spec.Disabled)
}

func TestMergeDeviceProperties(t *testing.T) {
	// Arrange
	device := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_props",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_props",
	})

	initialProps := mir_v1.DeviceProperties{
		Desired: map[string]any{
			"config": map[string]any{
				"feature1": true,
				"feature2": "enabled",
				"nested": map[string]any{
					"setting1": 123,
					"setting2": "value",
				},
			},
		},
		Reported: map[string]any{
			"status": "online",
			"metrics": map[string]any{
				"memory": 1024,
				"cpu":    0.5,
			},
		},
	}
	device.Properties = initialProps

	// Create initial device
	_, err := mirStore.CreateDevice(device)
	if err != nil {
		t.Error(err)
	}

	// Prepare merge patch
	mergePatch := map[string]interface{}{
		"properties": map[string]interface{}{
			"desired": map[string]interface{}{
				"config": map[string]interface{}{
					"feature2": "disabled",
					"feature3": "new",
					"nested": map[string]interface{}{
						"setting1": 456,
						"setting3": "new",
					},
				},
			},
			"reported": map[string]interface{}{
				"metrics": map[string]interface{}{
					"memory": 2048,
					"disk":   75,
				},
			},
		},
	}
	patchBytes, err := json.Marshal(mergePatch)
	if err != nil {
		t.Error(err)
	}

	// Act
	mergedDevices, err := mirStore.MergeDevice(device.ToTarget(), patchBytes, MergePatch)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(mergedDevices), 1)

	// Check desired properties
	config := mergedDevices[0].Properties.Desired["config"].(map[string]any)
	assert.Equal(t, true, config["feature1"])
	assert.Equal(t, "disabled", config["feature2"])
	assert.Equal(t, "new", config["feature3"])

	nested := config["nested"].(map[string]any)
	assert.Equal(t, uint64(456), nested["setting1"])
	assert.Equal(t, "value", nested["setting2"])
	assert.Equal(t, "new", nested["setting3"])

	// Check reported properties
	assert.Equal(t, "online", mergedDevices[0].Properties.Reported["status"])

	metrics := mergedDevices[0].Properties.Reported["metrics"].(map[string]any)
	assert.Equal(t, uint64(2048), metrics["memory"])
	assert.Equal(t, float64(0.5), metrics["cpu"])
	assert.Equal(t, uint64(75), metrics["disk"])
}

func TestMergeDeviceInvalidJson(t *testing.T) {
	// Arrange
	device := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_invalid",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_invalid",
	})

	// Create initial device
	_, err := mirStore.CreateDevice(device)
	if err != nil {
		t.Fatal(err)
	}

	// Prepare invalid JSON patch
	invalidJson := []byte(`{"meta": {"name": "new_name", "invalid_field": true}`)

	// Act
	_, err = mirStore.MergeDevice(device.ToTarget(), invalidJson, MergePatch)

	// Assert
	assert.ErrorContains(t, err, "unknown fields in json patch")
}

func TestMergeDeviceNoTarget(t *testing.T) {
	// Arrange
	emptyTarget := mir_v1.DeviceTarget{}
	patch := []byte(`{"meta": {"labels": {"test": "value"}}}`)

	// Act
	_, err := mirStore.MergeDevice(emptyTarget, patch, MergePatch)

	// Assert
	assert.ErrorContains(t, err, mir_v1.ErrorNoDeviceTargetProvided.Error())
}

func TestMergeDeviceUniqueConstraintViolation(t *testing.T) {
	// Arrange
	// Create first device
	device1 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_unique1",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_unique1",
	})

	// Create second device
	device2 := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "merge_device_unique2",
		Namespace: "mirstore",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.DeviceSpec{
		DeviceId: "merge_device_unique2",
	})

	// Create both devices
	_, err := mirStore.CreateDevice(device1)
	if err != nil {
		t.Error(err)
	}

	_, err = mirStore.CreateDevice(device2)
	if err != nil {
		t.Error(err)
	}

	// Prepare patch that would violate uniqueness constraint
	// by changing device2's name to device1's name
	mergePatch := map[string]interface{}{
		"meta": map[string]interface{}{
			"name": "merge_device_unique1",
		},
	}
	patchBytes, err := json.Marshal(mergePatch)
	if err != nil {
		t.Error(err)
	}

	// Act
	_, err = mirStore.MergeDevice(device2.ToTarget(), patchBytes, MergePatch)

	// Assert
	assert.ErrorContains(t, err, "cannot update device as name 'merge_device_unique1' is already in use in namespace 'mirstore'")
}

func TestPublishEventStoreCreateRequest(t *testing.T) {
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
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "create_event",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}).WithStatus(mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	})

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
}

func TestPublishEventStoreNotUnique(t *testing.T) {
	// Arrange
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "create_event_unique",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "create_event_unique",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateEvent(m2)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Assert(t, err != nil, true)
	assert.Assert(t, strings.Contains(err.Error(), "already exist"), true)
}

func TestPublishEventStoreListName(t *testing.T) {
	// Arrange
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Names: []string{"list_event_1", "list_event_2"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"events_list_test"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1_ns",
		Namespace: "events_list_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2_ns",
		Namespace: "events_list_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Labels: map[string]string{"test": "list_labels"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_labels",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_labels",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Labels: map[string]string{"test": "list_limit"},
		},
		Limit: 2,
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	})
	m3 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_3_limit",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := mirStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"store_test_time"},
		},
		DateFilter: mir_v1.DateFilter{
			From: time.Date(2025, 05, 7, 0, 0, 0, 0, time.UTC),
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 7, 13, 0, 0, 0, time.UTC)),
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC)),
	})
	m3 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_3_lbl",
		Namespace: "store_test_time",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 5, 0, 0, 0, 0, time.UTC)),
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := mirStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"store_test_time_to_from"},
		},
		DateFilter: mir_v1.DateFilter{
			From: time.Date(2025, 05, 7, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC),
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_1_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 7, 13, 0, 0, 0, time.UTC)),
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_2_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 7, 12, 0, 0, 0, time.UTC)),
	})
	m3 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_event_3_lbl",
		Namespace: "store_test_time_to_from",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "list_limit",
		},
	}).WithStatus(mir_v1.EventStatus{
		FirstAt: surrealTimePtr(time.Date(2025, 05, 5, 0, 0, 0, 0, time.UTC)),
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	mResp3, err := mirStore.CreateEvent(m3)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Names: []string{"delete_event_1", "delete_event_2"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Namespaces: []string{"events_delete_test"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_1_ns",
		Namespace: "events_delete_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_2_ns",
		Namespace: "events_delete_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.EventTarget{
		ObjectTarget: mir_v1.ObjectTarget{
			Labels: map[string]string{"test": "delete_labels"},
		},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_1_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "delete_labels",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "delete_event_2_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"test":     "delete_labels",
		},
	})
	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	dResp, err := mirStore.DeleteEvent(tar)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListEvent(tar)
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
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_meta_lbl"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_lbl",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
			"key3":     "test",
		},
		Annotations: map[string]string{
			"mirstore": "testing",
			"key3":     "test",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}).WithStatus(mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	})
	upd := mir_v1.EventUpdate{
		Meta: &mir_v1.MetaUpdate{
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
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := mirStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Meta.Labels["mirstore"], m.Meta.Labels["mirstore"])
	assert.Equal(t, uResp[0].Meta.Labels["caca_mou"], *upd.Meta.Labels["caca_mou"])
	_, ok := uResp[0].Meta.Labels["key3"]
	assert.Equal(t, false, ok)
	assert.Equal(t, uResp[0].Meta.Annotations["mirstore"], m.Meta.Annotations["mirstore"])
	assert.Equal(t, uResp[0].Meta.Annotations["caca_mou"], *upd.Meta.Annotations["caca_mou"])
	_, ok = uResp[0].Meta.Annotations["key3"]
	assert.Equal(t, false, ok)
}

func TestPublishEventStoreUpdateNameRequest(t *testing.T) {
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
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_meta_name"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_name",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}).WithStatus(mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	})
	upd := mir_v1.EventUpdate{
		Meta: &mir_v1.MetaUpdate{
			Name: strPtr("update_event_new_name"),
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := mirStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Meta.Name, *upd.Meta.Name)
}

func TestPublishEventStoreUpdateNameRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_meta_name_1"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_name_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_name_2",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	upd := mir_v1.EventUpdate{
		Meta: &mir_v1.MetaUpdate{
			Name: strPtr("update_event_meta_name_2"),
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = mirStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "is already in use in namespace"))
}

func TestPublishEventStoreUpdateNamespaceRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_meta_namespace_1"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_namespace_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_namespace_1",
		Namespace: "test_ns",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	upd := mir_v1.EventUpdate{
		Meta: &mir_v1.MetaUpdate{
			Namespace: strPtr("test_ns"),
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = mirStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "cannot update object as multiple device will have the same name"))
}

func TestPublishEventStoreUpdateNameNamespaceRequestDuplicate(t *testing.T) {
	// Arrange
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_meta_namens_1"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_namens_1",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_meta_namens_2",
		Namespace: "test_ns",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	})
	upd := mir_v1.EventUpdate{
		Meta: &mir_v1.MetaUpdate{
			Name:      strPtr("update_event_meta_namens_2"),
			Namespace: strPtr("test_ns"),
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	_, err = mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1 * time.Second)

	_, err = mirStore.UpdateEvent(tar, upd)

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, true, strings.Contains(err.Error(), "cannot update object has"))
}

func TestPublishEventStoreUpdateSpecRequest(t *testing.T) {
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
	jUpdMap := map[string]any{
		"caca_mou": "bien_mou",
	}
	jUpd, err := json.Marshal(jUpdMap)
	if err != nil {
		t.Error(err)
	}
	jRaw := jsonyaml.RawMessage(jUpd)
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_spec"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_spec",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}).WithStatus(mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	})
	upd := mir_v1.EventUpdate{
		Spec: &mir_v1.EventUpdateSpec{
			Type:    strPtr(mir_v1.EventTypeWarning),
			Reason:  strPtr("pizza"),
			Message: strPtr("test_de_la_mort"),
			Payload: &jRaw,
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(2 * time.Second)

	uResp, err := mirStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	uPayload := make(map[string]any)
	err = json.Unmarshal(uResp[0].Spec.Payload, &uPayload)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Spec.Reason, *upd.Spec.Reason)
	assert.Equal(t, uResp[0].Spec.Message, *upd.Spec.Message)
	assert.Equal(t, uResp[0].Spec.Type, *upd.Spec.Type)
	assert.Equal(t, uPayload["caca_mou"], jUpdMap["caca_mou"])
	_, ok := uPayload["key3"]
	assert.Equal(t, false, ok)
}

func TestPublishEventStoreUpdateStatusRequest(t *testing.T) {
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
	tar := mir_v1.ObjectTarget{
		Names: []string{"update_event_status"},
	}
	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "update_event_status",
		Namespace: "store_test",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}).WithStatus(mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	})
	upd := mir_v1.EventUpdate{
		Status: &mir_v1.EventUpdateStatus{
			Count:   intPtr(3),
			FirstAt: surrealTimePtr(time.Date(2014, 10, 14, 5, 5, 5, 5, time.UTC)),
			LastAt:  surrealTimePtr(time.Date(2014, 10, 14, 5, 5, 5, 5, time.UTC)),
		},
	}

	// Act
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}

	uResp, err := mirStore.UpdateEvent(tar, upd)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, uResp[0].Status.Count, *upd.Status.Count)
	assert.Equal(t, uResp[0].Status.LastAt.UTC(), upd.Status.LastAt.UTC())
	assert.Equal(t, uResp[0].Status.FirstAt.UTC(), upd.Status.FirstAt.UTC())
}

func TestPulishListDeviceWithEvents(t *testing.T) {
	// Arrange
	tar := mir_v1.DeviceTarget{
		Ids: []string{"peanut_butter"},
	}
	dev := mir_v1.NewDevice().WithMeta(mir_v1.Meta{
		Name:      "peanut_butter",
		Namespace: "eventstore_testing",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithId("peanut_butter")

	m := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_dev_with_event_1",
		Namespace: "eventstore_testing",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:   mir_v1.EventTypeNormal,
		Reason: "PEANUT",
		RelatedObject: mir_v1.Object{
			Meta: mir_v1.Meta{
				Name:      "peanut_butter",
				Namespace: "eventstore_testing",
			},
		},
	})
	m2 := mir_v1.NewEvent().WithMeta(mir_v1.Meta{
		Name:      "list_dev_with_event_2",
		Namespace: "eventstore_testing",
		Labels: map[string]string{
			"mirstore": "testing",
		},
	}).WithSpec(mir_v1.EventSpec{
		Type:   mir_v1.EventTypeNormal,
		Reason: "PEANUT",
		RelatedObject: mir_v1.Object{
			Meta: mir_v1.Meta{
				Name:      "peanut_butter",
				Namespace: "eventstore_testing",
			},
		},
	})
	// Act
	dResp, err := mirStore.CreateDevice(dev)
	if err != nil {
		t.Error(err)
	}
	mResp, err := mirStore.CreateEvent(m)
	if err != nil {
		t.Error(err)
	}
	mResp2, err := mirStore.CreateEvent(m2)
	if err != nil {
		t.Error(err)
	}
	lResp, err := mirStore.ListDevice(tar, true)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, mResp.Meta.Name, m.Meta.Name)
	assert.Equal(t, mResp2.Meta.Name, m2.Meta.Name)
	assert.Equal(t, dResp.Meta.Name, dev.Meta.Name)
	assert.Equal(t, 1, len(lResp))
	assert.Equal(t, len(lResp[0].Status.Events), 2)
	assert.Equal(t, lResp[0].Status.Events[0].Reason, "PEANUT")
	assert.Equal(t, lResp[0].Status.Events[1].Reason, "PEANUT")

}

func TestPublishEventStoreCreateBatchRequest(t *testing.T) {
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
	spec := mir_v1.EventSpec{
		Type:    mir_v1.EventTypeNormal,
		Reason:  "integration_test",
		Message: "a simple test",
		RelatedObject: mir_v1.Object{
			ApiVersion: "mir/v1alpha",
			Kind:       "device",
			Meta: mir_v1.Meta{
				Name:      "device1",
				Namespace: "store_test",
			},
		},
		Payload: j,
	}
	status := mir_v1.EventStatus{
		Count:   1,
		FirstAt: surrealTimePtr(time.Now().UTC()),
		LastAt:  surrealTimePtr(time.Now().UTC()),
	}
	count := 10
	events := make([]mir_v1.Event, count)
	for i := range count {
		events[i] = mir_v1.NewEvent().WithMeta(mir_v1.Meta{
			Name:      "create_event_batch_" + strconv.Itoa(i),
			Namespace: "store_test",
			Labels: map[string]string{
				"mirstore": "testing",
			},
		}).WithSpec(spec).WithStatus(status)
	}

	// Act

	mResp, err := mirStore.CreateEvents(events)
	if err != nil {
		t.Error(err)
	}

	// Assert
	assert.Equal(t, len(mResp), count)
}

func strPtr(s string) *string {
	return &s
}

func surrealTime(s time.Time) surrealdbModels.CustomDateTime {
	return surrealdbModels.CustomDateTime{Time: s}
}

func surrealTimePtr(s time.Time) *surrealdbModels.CustomDateTime {
	return &surrealdbModels.CustomDateTime{Time: s}
}

func intPtr(s int) *int {
	return &s
}

func boolPtr(s bool) *bool {
	return &s
}
