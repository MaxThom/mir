package mng

import (
	"encoding/json"
	"regexp"

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
	ListDevice(t mir_models.DeviceTarget, includeEvents bool) ([]mir_models.Device, error)
	CreateDevice(d mir_models.Device) (mir_models.Device, error)
	UpdateDevice(t mir_models.DeviceTarget, d mir_models.Device) ([]mir_models.Device, error)
	MergeDevice(t mir_models.DeviceTarget, patch json.RawMessage, op UpdateType) ([]mir_models.Device, error)
	DeleteDevice(t mir_models.DeviceTarget) ([]mir_models.Device, error)

	ListEvent(t mir_models.EventTarget) ([]mir_models.Event, error)
	CreateEvent(e mir_models.Event) (mir_models.Event, error)
	UpdateEvent(t mir_models.ObjectTarget, upd mir_models.EventUpdate) ([]mir_models.Event, error)
	MergeEvent(t mir_models.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_models.Event, error)
	DeleteEvent(t mir_models.EventTarget) ([]mir_models.Event, error)
}

type surrealMirStore struct {
	db *surrealdb.DB
}

func NewSurrealMirStore(db *surrealdb.DB) *surrealMirStore {
	return &surrealMirStore{
		db: db,
	}
}
