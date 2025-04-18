package mir

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/nats-io/nats.go"
)

type Store struct {
	props map[string]propsValue
	db    *badger.DB
	opts  StoreOptions
}

type StoreOptions struct {
	FolderPath string
	InMemory   bool
	Msgs       StoreMsgOptions
}

// MsgStorageType represents the storage mechanism
type MsgStorageType string
type StoreBucket string

const (
	StorageTypeNone MsgStorageType = "none"
	// StorageTypeNoStorage will not store messages
	StorageTypeNoStorage MsgStorageType = "nostorage"
	// StorageTypeOnlyIfOffline will keep messages only if device is d/c
	StorageTypeOnlyIfOffline MsgStorageType = "ifoffline"
	// StorageTypePersistent will keep all msgs
	StorageTypePersistent MsgStorageType = "persistent"

	msgPendingBucket    StoreBucket = "msgs.pending"
	msgPersistentBucket StoreBucket = "msgs.persistent"
	propertiesBucket    StoreBucket = "properties"
)

func (b StoreBucket) ToPrefix() []byte {
	return []byte(b + ":")
}

func (b StoreBucket) ToPrefixWithKey(key string) []byte {
	return []byte(string(b) + ":" + key)
}

func (b StoreBucket) ToKeyWithoutPrefix(key []byte) []byte {
	return bytes.TrimPrefix(key, []byte(b+":"))
}

type StoreMsgOptions struct {
	// Timelimit to store messages. If over, will start cycling messages
	// Default to 0 for infinite
	RententionLimit JsonReadableDuration
	// Cannot write messages to store if disk space left is above the pourcentage limit
	// Default to 85%, if disk space is at more then 85%, will start cycle the messages
	DiskSpaceLimit uint
	MsgStorageType MsgStorageType
}

type JsonReadableDuration time.Duration

func (d JsonReadableDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
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
	var err error
	path := ""
	if !s.opts.InMemory {
		path = s.opts.FolderPath
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("error creating directory: %w", err)
		}
	}

	opts := badger.DefaultOptions(path).
		WithInMemory(s.opts.InMemory).WithLoggingLevel(badger.WARNING)
	s.db, err = badger.Open(opts)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Load properties from the database
	if err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := propertiesBucket.ToPrefix()
		val := []byte{}
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := propertiesBucket.ToKeyWithoutPrefix(item.Key())
			val, err = item.ValueCopy(val)
			if err != nil {
				return err
			}
			propsValue := propsValue{}
			if err := json.Unmarshal(val, &propsValue); err != nil {
				return fmt.Errorf("error unmarshalling props from store: %w", err)
			}
			s.props[string(key)] = propsValue
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error loading properties from store: %w", err)
	}

	return nil
}

func (s *Store) GetProps(name string) (propsValue, bool) {
	props, ok := s.props[name]
	return props, ok
}

// Update the properties if the new value is different from the old one
// Returns the true if it was updated, false otherwise
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

	if err := s.db.Update(func(txn *badger.Txn) error {
		val, err := json.Marshal(localProp)
		if err != nil {
			return fmt.Errorf("error marshalling props to store: %w", err)
		}
		return txn.Set(propertiesBucket.ToPrefixWithKey(name), val)
	}); err != nil {
		return false, fmt.Errorf("error updating properties in store: %w", err)
	}

	// if !s.opts.InMemory {
	// 	if err := s.db.Update(func(tx *bolt.Tx) error {
	// 		b := tx.Bucket([]byte(propertiesBucket))

	// 		val, err := json.Marshal(localProp)
	// 		if err != nil {
	// 			return fmt.Errorf("error marshalling props to store: %w", err)
	// 		}

	// 		if err = b.Put([]byte(name), val); err != nil {
	// 			return fmt.Errorf("error writing props to store: %w", err)
	// 		}

	// 		return nil
	// 	}); err != nil {
	// 		return true, err
	// 	}
	// }
	return true, nil

}

func (s *Store) Close() error {
	return s.db.Close()
}

type entryItem struct {
	key []byte
	val []byte
	ttl time.Duration
}

func (s *Store) SwapMsgByBatch(bucketFrom StoreBucket, bucketTo StoreBucket, size int, h func([]nats.Msg) error) error {
	var errs error
	prefix := bucketFrom.ToPrefix()
	done := false
	iOpts := badger.DefaultIteratorOptions
	iOpts.PrefetchSize = size

	for !done {
		msgs := []nats.Msg{}
		batchStartKey := bucketFrom.ToPrefix()
		itemMarkForSwap := []entryItem{}

		if err := s.db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(iOpts)
			defer it.Close()

			count := 0
			val := []byte{}
			var err error
			for it.Seek(batchStartKey); it.ValidForPrefix(prefix) && count < size; it.Next() {
				count += 1
				item := it.Item()
				val, err = item.ValueCopy(val)
				if err != nil {
					return err
				}
				itemMarkForSwap = append(itemMarkForSwap, entryItem{
					key: item.KeyCopy(nil),
					val: bytes.Clone(val),
					ttl: time.Duration(item.ExpiresAt()),
				})

				msg := nats.Msg{}
				err := json.Unmarshal(val, &msg)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to unmarshal message: %w", err))
					continue
				}
				msgs = append(msgs, msg)
			}
			if !it.ValidForPrefix(prefix) {
				done = true
			} else {
				it.Next()
				if !it.ValidForPrefix(prefix) {
					done = true
				} else {
					batchStartKey = it.Item().Key()
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("error reading messages from store: %w", err)
		}

		if len(msgs) > 0 {
			err := h(msgs)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle batch messages: %w", err))
				continue
			}
		}

		if err := s.db.Update(func(txn *badger.Txn) error {
			for _, k := range itemMarkForSwap {
				errs = errors.Join(errs, txn.Delete(k.key))

				newKey := bucketTo.ToPrefixWithKey(string(bucketFrom.ToKeyWithoutPrefix(k.key)))
				newItem := badger.NewEntry(newKey, k.val).WithTTL(k.ttl)
				if err := txn.SetEntry(newItem); err != nil {
					errs = errors.Join(errs, err)
					continue
				}
			}
			return errs
		}); err != nil {
			return fmt.Errorf("error deleting messages from store: %w", err)
		}
	}

	return errs
}

func (s *Store) DeleteMsgByBatch(bucket StoreBucket, size int, h func([]nats.Msg) error) error {
	var errs error
	prefix := bucket.ToPrefix()
	done := false
	iOpts := badger.DefaultIteratorOptions
	iOpts.PrefetchSize = size

	for !done {
		msgs := []nats.Msg{}
		batchStartKey := bucket.ToPrefix()
		itemMarkForSwap := []entryItem{}

		if err := s.db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(iOpts)
			defer it.Close()

			count := 0
			val := []byte{}
			var err error
			for it.Seek(batchStartKey); it.ValidForPrefix(prefix) && count < size; it.Next() {
				count += 1
				item := it.Item()
				val, err = item.ValueCopy(val)
				if err != nil {
					return err
				}
				itemMarkForSwap = append(itemMarkForSwap, entryItem{
					key: item.KeyCopy(nil),
					val: bytes.Clone(val),
					ttl: time.Duration(item.ExpiresAt()),
				})

				msg := nats.Msg{}
				err := json.Unmarshal(val, &msg)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to unmarshal message: %w", err))
					continue
				}
				msgs = append(msgs, msg)
			}
			if !it.ValidForPrefix(prefix) {
				done = true
			} else {
				it.Next()
				if !it.ValidForPrefix(prefix) {
					done = true
				} else {
					batchStartKey = it.Item().Key()
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("error reading messages from store: %w", err)
		}

		if len(msgs) > 0 {
			err := h(msgs)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle batch messages: %w", err))
				continue
			}
		}

		if err := s.db.Update(func(txn *badger.Txn) error {
			for _, k := range itemMarkForSwap {
				errs = errors.Join(errs, txn.Delete(k.key))
			}
			return errs
		}); err != nil {
			return fmt.Errorf("error deleting messages from store: %w", err)
		}
	}
	return errs
}

// Read by batch, send the batch, delete/swap the batch
// Pass a handler to do by batch in one transaction

func (s *Store) SaveMsgToPending(msg nats.Msg) error {
	return s.saveMsg(msgPendingBucket, msg)
}

func (s *Store) SaveMsgToPermanent(msg nats.Msg) error {
	return s.saveMsg(msgPersistentBucket, msg)
}

func (s *Store) saveMsg(bucket StoreBucket, msg nats.Msg) error {
	// Save the message to the database
	if err := s.db.Update(func(txn *badger.Txn) error {
		key := bucket.ToPrefixWithKey(time.Now().UTC().Format(time.RFC3339Nano))
		msgData, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		entry := badger.NewEntry(key, msgData).WithTTL(time.Duration(s.opts.Msgs.RententionLimit))
		return txn.SetEntry(entry)
	}); err != nil {
		return fmt.Errorf("error storing message in store: %w", err)
	}
	return nil
	// return s.db.Update(func(tx *bolt.Tx) error {
	// 	pendingBucket := tx.Bucket([]byte(bucket))
	// 	key := []byte(time.Now().UTC().Format(time.RFC3339Nano))

	// 	msgData, err := json.Marshal(msg)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to marshal message: %w", err)
	// 	}

	// 	// Data is too small for compression
	// 	// d, err := zstd.CompressData(msgData)
	// 	// if err != nil {
	// 	// 	return fmt.Errorf("failted to compress message: %w", err)
	// 	// }

	// 	// Store the message in the pending bucket
	// 	if err := pendingBucket.Put(key, msgData); err != nil {
	// 		return fmt.Errorf("failed to store message: %w", err)
	// 	}

	// 	return nil
	// })
}
