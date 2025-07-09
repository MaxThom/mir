package mng

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/maxthom/mir/pkgs/mir_v1"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"github.com/surrealdb/surrealdb.go"
)

var (
	nullRegEx = regexp.MustCompile(`([:,{\[]\s*)null`)

	ErrorListingDevices        = errors.New("error listing devices from database")
	ErrorNoDeviceFound         = errors.New("no device found with current targets criteria")
	ErrorDeviceShouldBeCreated = errors.New("device should be created")

	ErrorListingEvents = errors.New("error listing events from database")
	ErrorNoEventFound  = errors.New("no events found with current targets criteria")
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

type natskvMirStore struct {
	js           jetstream.JetStream
	bucketDevice jetstream.KeyValue
	bucketEvent  jetstream.KeyValue
}

func NewNatsKVMirStore(js jetstream.JetStream) (*natskvMirStore, error) {
	bucketDevice, err := js.KeyValue(context.Background(), bucketStrDevice)
	if err != nil {
		if err == jetstream.ErrBucketNotFound {
			bucketDevice, err = js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: bucketStrDevice})
			if err != nil {
				return nil, fmt.Errorf("error creating storage bucket for device: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error retrieving storage bucket for device: %w", err)
		}
	}
	bucketEvent, err := js.KeyValue(context.Background(), bucketStrEvent)
	if err != nil {
		if err == jetstream.ErrBucketNotFound {
			bucketDevice, err = js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{Bucket: bucketStrEvent})
			if err != nil {
				return nil, fmt.Errorf("error creating storage bucket for event: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error retrieving storage bucket for event: %w", err)
		}
	}

	return &natskvMirStore{
		js:           js,
		bucketDevice: bucketDevice,
		bucketEvent:  bucketEvent,
	}, nil
}
