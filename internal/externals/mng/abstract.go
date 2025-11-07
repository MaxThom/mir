package mng

import (
	"encoding/json"
	"regexp"

	"github.com/maxthom/mir/internal/libs/external"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/pkgs/mir_v1"
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
	UpdateDeviceHello(updates map[mir_v1.DeviceId]mir_v1.DeviceHello) ([]mir_v1.Device, error)
	MergeDevice(t mir_v1.DeviceTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Device, error)
	DeleteDevice(t mir_v1.DeviceTarget) ([]mir_v1.Device, error)

	ListEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error)
	CreateEvent(e mir_v1.Event) (mir_v1.Event, error)
	CreateEvents(e []mir_v1.Event) ([]mir_v1.Event, error)
	UpdateEvent(t mir_v1.ObjectTarget, upd mir_v1.EventUpdate) ([]mir_v1.Event, error)
	MergeEvent(t mir_v1.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Event, error)
	DeleteEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error)

	Status() external.ConnectionStatus
	StatusSubscribe() <-chan external.ConnectionStatus
}

type surrealMirStore struct {
	db *surreal.AutoReconnDB
}

func NewSurrealMirStore(db *surreal.AutoReconnDB) *surrealMirStore {
	return &surrealMirStore{
		db: db,
	}
}

func (s *surrealMirStore) Status() external.ConnectionStatus {
	return s.db.ConnStatus
}

func (s *surrealMirStore) StatusSubscribe() <-chan external.ConnectionStatus {
	return s.db.StatusSubscribe()
}
