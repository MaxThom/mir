package surreal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/libs/resync"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
)

var ErrDatabaseDisconnected = fmt.Errorf("database disconnected")

type ConnectionStatus int

const (
	// Connection is connected
	StatusConnected ConnectionStatus = iota
	// Connection is disconnected and not trying to reconnect
	StatusDisconnected
	// Connection is attempting to reconnect
	StatusReconnecting
	// Connection cannot authentified itself to the db
	StatusNotAuthenticated
	// Connection is purposfully closed
	StatusClosed
)

func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "Disconnected"
	case StatusConnected:
		return "Connected"
	case StatusNotAuthenticated:
		return "NotAuthenticated"
	case StatusClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

type AutoReconnDB struct {
	*surrealdb.DB
	dbMu        sync.RWMutex
	ctx         context.Context
	log         zerolog.Logger
	ConnStatus  ConnectionStatus
	once        resync.Once
	isConn      bool
	connHandler ConnHandler
	Url         string
	User        string
	Password    string
	Namespace   string
	Database    string
}

type ConnHandler struct {
	FnConnected       func(url string)
	FnFailedReconnect func(url string, nextAttempt time.Duration)
	FnDisconnected    func(url string)
}

func Connect(ctx context.Context, url, namespace, database, user, password string, h ConnHandler) (*AutoReconnDB, error) {
	db, err := connect(ctx, url, namespace, database, user, password, h)
	if err != nil {
		// Check for connection errors
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "context deadline exceeded") {
			db.monitorAndReconnect()
		}
		return db, err
	}
	return db, nil
}

func connect(ctx context.Context, url, namespace, database, user, password string, h ConnHandler) (*AutoReconnDB, error) {
	db, err := surrealdb.Connect(ctx, url)
	if err != nil {
		d := &AutoReconnDB{
			DB:          db,
			ctx:         ctx,
			ConnStatus:  StatusDisconnected,
			connHandler: h,
			Url:         url,
			User:        user,
			Password:    password,
			Namespace:   namespace,
			Database:    database,
		}
		return d, err
	}

	if _, err = db.SignIn(ctx, &surrealdb.Auth{
		Username: user,
		Password: password,
	}); err != nil {
		d := &AutoReconnDB{
			DB:          db,
			ctx:         ctx,
			ConnStatus:  StatusNotAuthenticated,
			connHandler: h,
			Url:         url,
			User:        user,
			Password:    password,
			Namespace:   namespace,
			Database:    database,
		}
		return d, err
	}

	if err = db.Use(ctx, namespace, database); err != nil {
		return &AutoReconnDB{
			DB:          db,
			ctx:         ctx,
			ConnStatus:  StatusConnected,
			connHandler: h,
			isConn:      true,
			Url:         url,
			User:        user,
			Password:    password,
			Namespace:   namespace,
			Database:    database,
		}, err
	}

	return &AutoReconnDB{
		DB:          db,
		ctx:         ctx,
		ConnStatus:  StatusConnected,
		connHandler: h,
		isConn:      true,
		Url:         url,
		User:        user,
		Password:    password,
		Namespace:   namespace,
		Database:    database,
	}, nil
}

func (db *AutoReconnDB) Close() error {
	db.dbMu.RLock()
	defer db.dbMu.RUnlock()
	if db.DB == nil {
		return nil
	}
	return db.DB.Close(db.ctx)
}

// TODO, if error is of network disconnect
//  - [x] update status
//  - [x] start monitoring process
//    - [x] check connection every few seconds
//    - [x] recreate the db if needed
//    - [x] set status to connected
//  - [x] discard coming request as it is disconnected
//  - [x] add status changes handler

func (db *AutoReconnDB) monitorAndReconnect() {
	db.isConn = false
	fnM := func() {
		go func() {
			if db.connHandler.FnDisconnected != nil {
				db.connHandler.FnDisconnected(db.Url)
			}
			t := 5 * time.Second
			ticker := time.NewTicker(t)
			defer ticker.Stop()
			for {
				select {
				case <-db.ctx.Done():
					return
				case <-ticker.C:
					d, err := connect(db.ctx, db.Url, db.Namespace, db.Database, db.User, db.Password, db.connHandler)
					db.dbMu.Lock()
					db.DB = d.DB
					db.ConnStatus = d.ConnStatus
					db.dbMu.Unlock()
					if err != nil {
						// Mean still trying to connect
						if db.connHandler.FnFailedReconnect != nil {
							db.connHandler.FnFailedReconnect(db.Url, t)
						}
						continue
					}
					db.isConn = true
					if db.connHandler.FnConnected != nil {
						db.connHandler.FnConnected(db.Url)
					}
					db.once.Reset()
					return
				}
			}
		}()
	}
	db.once.Do(fnM)
}

func Create[T any, TWhat surrealdb.TableOrRecord](db *AutoReconnDB, what TWhat, data any) (*T, error) {
	if !db.isConn {
		return nil, ErrDatabaseDisconnected
	}
	ctxT, cancel := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancel()
	db.dbMu.RLock()
	respDb, err := surrealdb.Create[T](ctxT, db.DB, what, data)
	db.dbMu.RUnlock()
	if err != nil {
		// Check for connection errors
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "context deadline exceeded") {
			db.monitorAndReconnect()
		}
		return nil, err
	}

	return respDb, err
}

func Query[T any](db *AutoReconnDB, query string, vars map[string]any) (T, error) {
	var empty T
	if !db.isConn {
		return empty, ErrDatabaseDisconnected
	}

	ctxT, cancel := context.WithTimeout(db.ctx, 5*time.Second)
	defer cancel()
	db.dbMu.RLock()
	result, err := surrealdb.Query[T](ctxT, db.DB, query, vars)
	db.dbMu.RUnlock()
	if err != nil {
		// Check for connection errors
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "broken pipe") || strings.Contains(err.Error(), "context deadline exceeded") {
			db.monitorAndReconnect()
		}
		return empty, err
	}

	res := *result
	if len(res) == 0 {
		return empty, nil
	}

	return res[0].Result, nil
}
