package mng

import (
	"os"
	"testing"

	"github.com/maxthom/mir/internal/libs/test_utils"
	"github.com/maxthom/mir/pkgs/api/proto/v1alpha/core_api"
)

var store surrealDeviceStore

func TestMain(m *testing.M) {

	db := test_utils.SetupSurrealDbConnsPanic("ws://127.0.0.1:8000/rpc", "root", "root", "global", "mir")
	store = *NewSurrealDeviceStore(db)

	store.DeleteDevice(&core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Labels: map[string]string{
				"testing": "mng_store",
			},
		},
	})

	exitVal := m.Run()

	store.DeleteDevice(&core_api.DeleteDeviceRequest{
		Targets: &core_api.Targets{
			Labels: map[string]string{
				"testing": "mng_store",
			},
		},
	})
	os.Exit(exitVal)
}

func TestUpdateRequest(t *testing.T) {
	// Arrange
	_, err := store.CreateDevice(&core_api.CreateDeviceRequest{
		DeviceId:  "TestUpdateFreeRequest",
		Name:      "bob",
		Namespace: "integration_test",
		Disabled:  false,
		Labels: map[string]string{
			"testing": "mng_store",
			"hero":    "larzuk",
		},
	})
	if err != nil {
		t.Error(err)
	}

	// Act

	// Assert
}

func strRef(s string) *string {
	return &s
}
