package mng

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/maxthom/mir/internal/libs/external"
	"github.com/maxthom/mir/internal/libs/external/surreal"
	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/rs/zerolog"
)

var (
	nullRegEx = regexp.MustCompile(`([:,{\[]\s*)null`)
	l         zerolog.Logger
)

type UpdateType string

const (
	JsonPatch  UpdateType = "json-patch"
	MergePatch UpdateType = "merge-patch"
)

type MirStore interface {
	ListDevice(t mir_v1.DeviceTarget, includeEvents bool) ([]mir_v1.Device, error)
	CreateDevice(d mir_v1.Device) (mir_v1.Device, error)
	CreateDevices(d []mir_v1.Device) ([]mir_v1.Device, error)
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

func NewSurrealMirStore(log zerolog.Logger, db *surreal.AutoReconnDB) (*surrealMirStore, error) {
	l = log.With().Str("srv", "surreal_mir_store").Logger()
	st := &surrealMirStore{
		db: db,
	}

	// Subscribe to db status changes and initialize schema on reconnect
	go func() {
		for status := range db.StatusSubscribe() {
			if status == external.StatusConnected {
				err := st.initSchema()
				if err != nil {
					l.Error().Err(err).Msg("error initializing surreal schema on db reconnect")
				}
				l.Debug().Msg("db reconnected, schema initialized")
			}
		}
	}()

	err := st.initSchema()
	if err != nil {
		if db.ConnStatus == external.StatusConnected {
			l.Error().Err(err).Msg("error initializing surreal schema")
			return nil, err
		} else {
			l.Warn().Err(err).Msg("error initializing surreal schema, it will be created on connect")
		}
	}

	return st, nil
}

func (s *surrealMirStore) initSchema() error {
	q := `
		DEFINE TABLE IF NOT EXISTS devices SCHEMALESS;
		DEFINE TABLE IF NOT EXISTS events SCHEMALESS;
	`
	_, err := surreal.Query[any](s.db, q, nil)
	if err != nil {
		return fmt.Errorf("%w for schema init: %w", mir_v1.ErrorDbExecutingQuery, err)
	}

	return nil
}

func (s *surrealMirStore) Status() external.ConnectionStatus {
	return s.db.ConnStatus
}

func (s *surrealMirStore) StatusSubscribe() <-chan external.ConnectionStatus {
	return s.db.StatusSubscribe()
}
