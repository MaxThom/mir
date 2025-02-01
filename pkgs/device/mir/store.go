package mir

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
