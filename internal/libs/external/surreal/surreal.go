package surreal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/maxthom/mir/internal/libs/external"
	"github.com/maxthom/mir/internal/libs/resync"
	"github.com/rs/zerolog"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

var ErrDatabaseDisconnected = fmt.Errorf("database disconnected")

const defaultTimeout = 30 * time.Second

type AutoReconnDB struct {
	*surrealdb.DB
	statusSubs  []chan external.ConnectionStatus
	statusMu    sync.RWMutex
	dbMu        sync.RWMutex
	ctx         context.Context
	log         zerolog.Logger
	ConnStatus  external.ConnectionStatus
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
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "connection reset by peer") ||
			strings.Contains(err.Error(), "unexpected EOF") {
			db.monitorAndReconnect()
		}
		return db, err
	}
	if db.connHandler.FnConnected != nil {
		db.connHandler.FnConnected(db.Url)
	}
	return db, nil
}

func connect(ctx context.Context, url, namespace, database, user, password string, h ConnHandler) (*AutoReconnDB, error) {
	db, err := surrealdb.Connect(ctx, url)
	if err != nil {
		d := &AutoReconnDB{
			DB:          db,
			ctx:         ctx,
			ConnStatus:  external.StatusDisconnected,
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
			ConnStatus:  external.StatusNotAuthenticated,
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
			ConnStatus:  external.StatusConnected,
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
		ConnStatus:  external.StatusConnected,
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
	db.dbMu.Lock()
	defer db.dbMu.Unlock()
	db.statusMu.Lock()
	defer db.statusMu.Unlock()

	db.ConnStatus = external.StatusClosed
	for _, ch := range db.statusSubs {
		ch <- db.ConnStatus
		close(ch)
	}

	if db.DB == nil {
		return nil
	}

	return db.DB.Close(db.ctx)
}

func (db *AutoReconnDB) monitorAndReconnect() {
	db.isConn = false
	fnM := func() {
		go func() {
			db.ConnStatus = external.StatusDisconnected
			for _, ch := range db.statusSubs {
				ch <- db.ConnStatus
			}
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
					if db.ConnStatus != d.ConnStatus {
						for _, ch := range db.statusSubs {
							ch <- d.ConnStatus
						}
					}
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

func (db *AutoReconnDB) StatusSubscribe() <-chan external.ConnectionStatus {
	ch := make(chan external.ConnectionStatus, 10) // Buffered to prevent blocking
	db.statusMu.Lock()
	db.statusSubs = append(db.statusSubs, ch)
	db.statusMu.Unlock()
	return ch
}

func Create[T any, TWhat surrealdb.TableOrRecord](db *AutoReconnDB, what TWhat, data any) (*T, error) {
	if !db.isConn {
		return nil, ErrDatabaseDisconnected
	}
	ctxT, cancel := context.WithTimeout(db.ctx, defaultTimeout)
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

func Insert[TResult any](db *AutoReconnDB, what string, data any) (*[]TResult, error) {
	if !db.isConn {
		return nil, ErrDatabaseDisconnected
	}
	ctxT, cancel := context.WithTimeout(db.ctx, defaultTimeout)
	defer cancel()
	db.dbMu.RLock()
	respDb, err := surrealdb.Insert[TResult](ctxT, db.DB, models.Table(what), data)
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

	ctxT, cancel := context.WithTimeout(db.ctx, defaultTimeout)
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

func QueryMultiple[T any](db *AutoReconnDB, query string, vars map[string]any) ([]T, error) {
	var empty []T
	if !db.isConn {
		return empty, ErrDatabaseDisconnected
	}

	ctxT, cancel := context.WithTimeout(db.ctx, defaultTimeout)
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

	objs := make([]T, len(res))
	for _, r := range res {
		objs = append(objs, r.Result)
	}

	return objs, nil
}
