package mir

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nats-io/nats.go"
	bolt "go.etcd.io/bbolt"
)

type Store struct {
	props map[string]propsValue
	db    *bolt.DB
	opts  StoreOptions
}

type StoreOptions struct {
	Path     string
	InMemory bool
	Msgs     StoreMsgOptions
}

// MsgStorageType represents the storage mechanism
type MsgStorageType int

const (
	StorageTypeNone MsgStorageType = iota
	// StorageTypeNoStorage will not store messages
	StorageTypeNoStorage
	// StorageTypeOnlyIfOffline will keep messages only if device is d/c
	StorageTypeOnlyIfOffline
	// StorageTypeAlways will keep all msgs
	StorageTypeAlways
)

type StoreMsgOptions struct {
	// Timelimit to store messages. If over, will start cycling messages
	// Default to 0 for infinite
	RententionLimit time.Duration
	// Cannot write messages to store if disk space left is above the pourcentage limit
	// Default to 85%, if disk space is at more then 85%, will start cycle the messages
	DiskSpaceLimit uint
	MsgStorageType MsgStorageType
}

type propsValue struct {
	LastUpdate time.Time `json:"lastUpdate"`
	Value      []byte    `json:"value"`
}

func NewStore(opts StoreOptions) (*Store, error) {
	return &Store{
		props: make(map[string]propsValue),
		db:    nil,
		opts:  opts,
	}, nil
}

func (s *Store) Load() error {
	if !s.opts.InMemory {
		var err error
		if err := os.MkdirAll(filepath.Dir(s.opts.Path), 0755); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
		s.db, err = bolt.Open(s.opts.Path, 0600, &bolt.Options{Timeout: 3 * time.Second})
		if err != nil {
			return err
		}

		if err := s.db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte("msgs.pending"))
			if err != nil {
				return fmt.Errorf("failed to create msgs.pending bucket: %w", err)
			}
			_, err = tx.CreateBucketIfNotExists([]byte("msgs.sent"))
			if err != nil {
				return fmt.Errorf("failed to create msgs.sent bucket: %w", err)
			}

			b, err := tx.CreateBucketIfNotExists([]byte("properties"))
			if err != nil {
				return fmt.Errorf("error creating properties bucket: %w", err)
			}
			if err = b.ForEach(func(k, v []byte) error {
				propsValue := propsValue{}
				if err := json.Unmarshal(v, &propsValue); err != nil {
					return fmt.Errorf("error unmarshalling props from store: %w", err)
				}
				s.props[string(k)] = propsValue
				return nil
			}); err != nil {
				return fmt.Errorf("error loading store: %w", err)
			}

			return nil
		}); err != nil {
			return fmt.Errorf("error loading store: %w", err)
		}
	}
	return nil
}

func (s *Store) GetProps(name string) (propsValue, bool) {
	props, ok := s.props[name]
	return props, ok
}

// Update the properties if the new value is different from the old one
// Returns the new value if it was updated, nil otherwise
func (s *Store) UpdatePropsIfNew(name string, prop propsValue) (bool, error) {
	localProp, ok := s.GetProps(name)
	if !ok {
		// Never had that config before
		localProp = propsValue{
			LastUpdate: prop.LastUpdate,
			Value:      prop.Value,
		}
		s.props[name] = localProp
	} else if prop.LastUpdate.After(localProp.LastUpdate) {
		// Config has changed
		localProp.LastUpdate = prop.LastUpdate
		localProp.Value = prop.Value
		s.props[name] = localProp
	} else {
		return false, nil
	}

	if !s.opts.InMemory {
		if err := s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("properties"))

			val, err := json.Marshal(localProp)
			if err != nil {
				return fmt.Errorf("error marshalling props to store: %w", err)
			}

			if err = b.Put([]byte(name), val); err != nil {
				return fmt.Errorf("error writing props to store: %w", err)
			}

			return nil
		}); err != nil {
			return true, err
		}
	}
	return true, nil

}

func (s *Store) Close() error {
	if s.opts.InMemory {
		return nil
	}
	return s.db.Close()
}

func (s *Store) SaveMsgNoStorage(msg nats.Msg) error {
	return nil
}

func (s *Store) SwapMsgFromPendingToSent(msg nats.Msg) error {
	// return s.swapMsg("msgs.pending", "msgs.sent", msg)
	return nil
}

func (s *Store) DeleteMsgByPatch(bucket string, size int) error {
	msgs := []nats.Msg{}
	key := []byte{}
	v := []byte{}
	count := 0

	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return errors.New("bucket '" + bucket + "' not found")
		}
		c := b.Cursor()
		if len(key) == 0 {
			key, _ = c.First()
		}

		for key, v = c.Seek(key); key != nil && count < size; key, v = c.Next() {
			count += 1
			fmt.Printf("%s %s\n", string(key), string(v))

			msg := nats.Msg{}
			err := json.Unmarshal(v, &msg)
			if err != nil {
				return fmt.Errorf("failed to unmarshal message: %w", err)
			}
			msgs = append(msgs, msg)
			fmt.Println(msg)
		}

		return nil
	})

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return errors.New("bucket '" + bucket + "' not found")
		}

		return nil
	})
}

// Read by batch, send the batch, delete/swap the batch
// Pass a handler to do by batch in one transaction

func (s *Store) SaveMsgToPending(msg nats.Msg) error {
	return s.saveMsg("msgs.pending", msg)
}

func (s *Store) SaveMsgToSent(msg nats.Msg) error {
	return s.saveMsg("msgs.sent", msg)
}

func (s *Store) saveMsg(bucket string, msg nats.Msg) error {
	// Save the message to the database
	return s.db.Update(func(tx *bolt.Tx) error {
		fmt.Println("CACA")
		pendingBucket := tx.Bucket([]byte(bucket))
		key := []byte(time.Now().UTC().Format(time.RFC3339Nano))

		msgData, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		// Store the message in the pending bucket
		if err := pendingBucket.Put(key, msgData); err != nil {
			return fmt.Errorf("failed to store message: %w", err)
		}

		return nil
	})
}

func (s *Store) SaveMsgAlways(msg nats.Msg) error {
	return nil
}
