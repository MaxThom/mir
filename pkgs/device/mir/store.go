package mir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Store struct {
	ctx   context.Context
	props map[string]propsValue
	db    *bolt.DB
}

type StoreOptions struct {
	path string
}

type propsValue struct {
	LastUpdate time.Time `json:"lastUpdate"`
	Value      []byte    `json:"value"`
}

func NewStore(opts StoreOptions) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(opts.path), 0755); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}
	db, err := bolt.Open(opts.path, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, err
	}
	return &Store{
		props: make(map[string]propsValue),
		db:    db,
	}, nil
}

func (s *Store) Load(ctx context.Context) error {
	s.ctx = ctx

	go func() {
		<-ctx.Done()
		s.Close()
	}()

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
	return true, nil

}

func (s *Store) Close() error {
	return s.db.Close()
}
