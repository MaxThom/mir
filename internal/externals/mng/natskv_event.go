package mng

import (
	"encoding/json"

	"github.com/maxthom/mir/pkgs/mir_v1"
)

var (
	kindEvent      string = "event"
	bucketStrEvent string = scope + "." + apiVersionV1Alpha + "." + kindEvent
)

func (s *natskvMirStore) ListEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error) {
	return []mir_v1.Event{}, nil
}

func (s *natskvMirStore) CreateEvent(e mir_v1.Event) (mir_v1.Event, error) {
	return e, nil
}

// This method is too OP
// Maybe it need to be divided into Upsert and Patch
// Upsert is for apply and edit
// Patch is for patch
func (s *natskvMirStore) UpdateEvent(t mir_v1.ObjectTarget, upd mir_v1.EventUpdate) ([]mir_v1.Event, error) {
	return nil, nil
}

func (s *natskvMirStore) MergeEvent(t mir_v1.ObjectTarget, patch json.RawMessage, op UpdateType) ([]mir_v1.Event, error) {
	return nil, nil
}

func (s *natskvMirStore) DeleteEvent(t mir_v1.EventTarget) ([]mir_v1.Event, error) {
	return nil, nil
}
