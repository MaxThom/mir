package mng

import (
	"encoding/json"
	"regexp"

	core_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/v1/core_api"
	"github.com/maxthom/mir/pkgs/mir_models"
	"github.com/surrealdb/surrealdb.go"
)

var (
	nullRegEx = regexp.MustCompile(`([:,{\[]\s*)null`)
)

type UpdateType string

const (
	JsonPatch  UpdateType = "json-patch"
	MergePatch UpdateType = "merge-patch"
)

type MirStore interface {
	ListDevice(req *core_apiv1.ListDeviceRequest) ([]mir_models.Device, error)
	CreateDevice(req *core_apiv1.CreateDeviceRequest) (mir_models.Device, error)
	UpdateDevice(req *core_apiv1.UpdateDeviceRequest) ([]mir_models.Device, error)
	MergeDevice(targets *core_apiv1.Targets, patch json.RawMessage, op UpdateType) ([]mir_models.Device, error)
	DeleteDevice(req *core_apiv1.DeleteDeviceRequest) ([]mir_models.Device, error)

	ListEvent(t mir_models.ObjectTarget) ([]mir_models.Event, error)
	CreateEvent(e mir_models.Event) (mir_models.Event, error)
	UpdateEvent(t mir_models.ObjectTarget, upd mir_models.EventUpdate) ([]mir_models.Event, error)
	MergeEvent(t mir_models.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_models.Event, error)
	DeleteEvent(t mir_models.ObjectTarget) ([]mir_models.Event, error)
}

type surrealMirStore struct {
	db *surrealdb.DB
}

func NewSurrealMirStore(db *surrealdb.DB) *surrealMirStore {
	return &surrealMirStore{
		db: db,
	}
}
