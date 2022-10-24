package core

import (
	"bytes"
	"context"
	"encoding"
	"errors"
	"fmt"
	"sync"
	"time"
)

type (
	// DB is a genric interface for kv store databases
	DB interface {
		BatchGet() DBBatch
		Size() (uint64, error)
		TxnW(callback func(txn DBTxn) error) error
		TxnR(callback func(txn DBTxn) error) error
		Close() error
	}

	// DBBatch is provides methods to perform batch db operations.
	DBBatch interface {
		Cancel()
		ObjPut(DBK, DBObj, time.Duration) error
		ObjDel(DBK) error
		Flush() error
	}

	// DBTxn is the workhorse of db activity (everything wrapped in this)
	DBTxn interface {
		ObjExists(DBK) (bool, error)
		ObjGet(DBK, DBObj) error
		ObjPut(DBK, DBObj, time.Duration) error

		ObjDel(DBK) error
		ObjDelByKBytes(key []byte) error
		ObjDelAll(func(k, v []byte) (bool, error)) error
		KIterGet(prefix []byte, fwd bool, prefetchSize int, unmarshaller func([]byte) error) DBKeyIter
		KVIterGet(prefix []byte, fwd bool, prefetchSize int, unmarshaller func([]byte) error) DBStringIter
	}

	// DBKeyIter is an iterator for keys
	DBKeyIter interface {
		Next() (key []byte, ok bool, err error)
		Seek(key []byte)
		Close()
	}

	// DBStringIter is an iterator for keys and values
	DBStringIter interface {
		Next() (key []byte, value []byte, ok bool, err error)
		Seek(key []byte)
		Close()
	}

	// DBIndexKeyGen is a function that generates the key of an index for the DBType
	DBIndexKeyGen func(dbType DBObj) []byte

	// DBIndexKeyGenMap maps index names to DBIndexKeyGen functions
	DBIndexKeyGenMap map[string]DBIndexKeyGen

	// DBTypeRegistry tracks all types to make sure pks dont have the same prefix
	DBTypeRegistry struct {
		sync.Mutex
		Types [][]byte
	}
)

var (
	// ErrDBItemNotFound indicates the query returned no results
	ErrDBItemNotFound = errors.New("Not found")

	// DBTypeRegistryGlobal is the global registry of database types
	DBTypeRegistryGlobal = &DBTypeRegistry{
		Types: make([][]byte, 0),
	}
)

// Register adds a type to the DBTypeRegistry. Since using duplicate type keys would cause data
// corruption, registering a duplicate type key causes a panic.
func (d *DBTypeRegistry) Register(typeKey []byte) {
	defer d.Unlock()
	d.Lock()
	for _, registeredType := range d.Types {
		if bytes.Equal(registeredType, typeKey) {
			panic("type already registered")
		}
	}
	d.Types = append(d.Types, typeKey)
}

//
//
//
// DBK Interface for Primary Keys
type DBK interface {
	fmt.Stringer
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// This method makes this interface stronger for type checking
	IsADBK() bool
}

//
//
//
// DBObj is an interface for objects that can be persisted with DB
type DBObj interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Indexes() DBIndexKeyGenMap
}

// DBReq is a GET/PUT/DEL request to the db
type DBReq struct {
	pk DBK
	ob DBObj
}

type nilDBObj struct{ data []byte }

func (d *nilDBObj) MarshalBinary() (out []byte, err error) {
	return d.data, nil
}

func (d *nilDBObj) UnmarshalBinary(in []byte) (err error) {
	return nil
}

func (d *nilDBObj) Indexes() DBIndexKeyGenMap {
	return make(map[string]DBIndexKeyGen)
}

// NilDBObj marshalls and unmarshals nothing
var NilDBObj = &nilDBObj{}

// For stuffing DBTxns in contexts
type ctxKey int

const (
	dbTxnCtxValueKey ctxKey = iota
)

// DBTxnFromCtx gets the database transaction from the context
func DBTxnFromCtx(ctx context.Context) DBTxn {
	return ctx.Value(dbTxnCtxValueKey).(DBTxn)
}

// CtxWithDbTxn puts a database txn into the context
func CtxWithDbTxn(ctx context.Context, dbTxn DBTxn) context.Context {
	return context.WithValue(ctx, dbTxnCtxValueKey, dbTxn)
}
