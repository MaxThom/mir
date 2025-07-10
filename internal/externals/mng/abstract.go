package mng

import (
	"encoding/json"
	"regexp"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/maxthom/surrealdb.go"
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
	ListDevice(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error)
	CreateDevice(d mir_v1.Device) (mir_v1.Device, error)
	UpdateDevice(t mir_v1.DeviceTarget, d mir_v1.Device) ([]mir_v1.Device, error)
	MergeDevice(t mir_v1.DeviceTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Device, error)
	DeleteDevice(t mir_v1.DeviceTarget) ([]mir_v1.Device, error)

	ListEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error)
	CreateEvent(e mir_v1.Event) (mir_v1.Event, error)
	UpdateEvent(t mir_v1.ObjectTarget, upd mir_v1.EventUpdate) ([]mir_v1.Event, error)
	MergeEvent(t mir_v1.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Event, error)
	DeleteEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error)
}

type surrealMirStore struct {
	db *surrealdb.DB
}

func NewSurrealMirStore(db *surrealdb.DB) *surrealMirStore {
	return &surrealMirStore{
		db: db,
	}
}
