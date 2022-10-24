package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	// Default DBBadger discardRatio. It represents the discard ratio for the
	// DBBadger GC.
	//
	// Ref: https://godoc.org/github.com/dgraph-io/badger#DB.RunValueLogGC
	badgerDiscardRatio = 0.5

	// Default DBBadger GC interval
	badgerGCInterval = 10 * time.Minute

	// Badget Transactions
	// https://blog.dgraph.io/post/badger-txn/
)

// DBBadger maintains information An implementation of DB backed by Badger DB
type DBBadger struct {
	DB         *badger.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
	logger     *logrus.Logger
}

// BadgerBatch wraps badger.WriteBatch
type BadgerBatch struct {
	WriteBatch *badger.WriteBatch
}

var dbBadgerExportMetricsComplete bool

func dbBadgerExportMetrics() {
	if dbBadgerExportMetricsComplete {
		return
	}
	exports := map[string]*prometheus.Desc{}
	metricnames := []string{
		"badger_disk_reads_total",
		"badger_disk_writes_total",
		"badger_read_bytes",
		"badger_written_bytes",
		"badger_lsm_level_gets_total",
		"badger_lsm_bloom_hits_total",
		"badger_gets_total",
		"badger_puts_total",
		"badger_blocked_puts_total",
		"badger_memtable_gets_total",
		"badger_lsm_size_bytes",
		"badger_vlog_size_bytes",
		"badger_pending_writes_total",
	}
	for _, name := range metricnames {
		exports[name] = prometheus.NewDesc(
			name,
			"badger db metric "+name,
			nil, nil,
		)
	}
	collector := prometheus.NewExpvarCollector(exports)
	PromRegisterCollector(collector)

}

// NewDBBadger returns a DB backed by DBBadger
// If the database cannot be initialized, an error will be returned.
func NewDBBadger(opts *badger.Options, logger *logrus.Logger) *DBBadger {
	dbBadgerExportMetrics()
	if err := os.MkdirAll(opts.ValueDir, 0774); err != nil {
		Log.Error("NewDBBadger: " + err.Error())
		os.Exit(1)
	}
	if err := os.MkdirAll(opts.Dir, 0774); err != nil {
		Log.Error("NewDBBadger: " + err.Error())
		os.Exit(1)
	}

	dbBadger, err := badger.Open(*opts)
	if err != nil {
		Log.Error(err)
		os.Exit(1)
	}

	bdb := &DBBadger{
		DB:     dbBadger,
		logger: logger,
	}
	bdb.ctx, bdb.cancelFunc = context.WithCancel(context.Background())

	go bdb.runGC()

	// counter for redirect rate
	return bdb
}

// DBBadgerTxn is a database transaction context
type DBBadgerTxn struct {
	badgerTxn *badger.Txn
}

// TxnW gets a new read-write transaction context
func (bdb *DBBadger) TxnW(callback func(txn DBTxn) error) error {
	return bdb.DB.Update(func(badgerTxn *badger.Txn) error {
		return callback(&DBBadgerTxn{badgerTxn: badgerTxn})
	})
}

// TxnR gets a new read-only transaction context
func (bdb *DBBadger) TxnR(callback func(txn DBTxn) error) error {
	return bdb.DB.View(func(badgerTxn *badger.Txn) error {
		return callback(&DBBadgerTxn{badgerTxn: badgerTxn})
	})
}

// Size returns size of the database, but badger provides no good way to do that.
func (bdb *DBBadger) Size() (uint64, error) {
	return 0, errors.New("not implemented")
	// var total uint64
	// tables := bdb.DB.Tables(true)
	// for _, table := range tables {
	// 	total += table.KeyCount
	// }

	// return total, nil
}

// BatchGet returns a BadgerBatch for batch operations
func (bdb *DBBadger) BatchGet() DBBatch {
	return &BadgerBatch{
		WriteBatch: bdb.DB.NewWriteBatch(),
	}
}

// Flush commits the BadgerBatch
func (bb *BadgerBatch) Flush() error {
	return bb.WriteBatch.Flush()
}

// Cancel should be called even after flush (doesn't hurt)
func (bb *BadgerBatch) Cancel() {
	bb.WriteBatch.Cancel()
}

// ObjPut inserts or updates an node to the DB
func (bb *BadgerBatch) ObjPut(k DBK, v DBObj, ttl time.Duration) error {
	kb, err := k.MarshalBinary()
	if err != nil {
		return err
	}
	Log.Tracef("DBBadgerTxn.ObjPut: PUTTING  -->%s<--", k.String())
	val, err := v.MarshalBinary()
	if err != nil {
		return err
	}

	// ttl requires a different call
	if ttl > 0 {
		err = bb.WriteBatch.SetEntry(badger.NewEntry(kb, val).WithTTL(ttl))
	} else {
		err = bb.WriteBatch.Set(kb, val)
	}
	if err != nil {
		return err
	}

	// Now write indexes
	if v != nil {
		indexes := v.Indexes()
		if indexes != nil {
			for _, dbIndexKeyGen := range indexes {
				indexKey := dbIndexKeyGen(v)
				indexEntryKey := make([]byte, len(kb)+len(indexKey))
				copy(indexEntryKey[:], indexKey)
				copy(indexEntryKey[len(indexKey):], kb)
				err := bb.WriteBatch.Set(indexEntryKey, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ObjDel removes an object by k
func (bb *BadgerBatch) ObjDel(k DBK) error {
	kb, err := k.MarshalBinary()
	if err != nil {
		return err
	}
	Log.Debugf("BadgerBatch.ObjDel: DELTING  -->%s<--", k.String())

	err = bb.WriteBatch.Delete(kb)
	if err != nil {
		return err
	}

	// indexes := obj.Indexes()
	// if indexes != nil {
	// 	for _, dbIndexKeyGen := range indexes {
	// 		indexKey := dbIndexKeyGen(obj)
	// 		indexEntryKey := make([]byte, len(pkb)+len(indexKey))
	// 		copy(indexEntryKey[:], indexKey)
	// 		copy(indexEntryKey[len(indexKey):], pkb)
	// 		err := bb.WriteBatch.Set(indexEntryKey, nil)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	return nil
}

// ObjGet hydrates an node from the DB by the K data already set in the node
func (txn *DBBadgerTxn) ObjGet(k DBK, v DBObj) error {
	// https://groups.google.com/forum/#!topic/golang-nuts/wnH302gBa4I/discussion
	if InterfaceIsNil(k) {
		return errors.New("ObjGet: k is nil... must provide a key for lookup")
	}
	if InterfaceIsNil(v) {
		return errors.New("ObjGet: v is nil... must provide an empty value")
	}
	kb, err := k.MarshalBinary()
	if err != nil {
		return err
	}
	Log.Debugf("DBBadgerTxn.ObjGet: GETTING  -->%s<--", k.String())

	item, err := txn.badgerTxn.Get(kb)
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return ErrDBItemNotFound
		case nil:
			return err
		}
	}
	err = item.Value(func(val []byte) error {
		return v.UnmarshalBinary(val)
	})
	return err
}

// ObjPut inserts or updates an node to the DB
func (txn *DBBadgerTxn) ObjPut(k DBK, v DBObj, ttl time.Duration) error {
	kb, err := k.MarshalBinary()
	if err != nil {
		return err
	}
	Log.Debugf("DBBadgerTxn.ObjPut: PUTTING  -->%s<--", k.String())
	var val []byte
	if v != nil {
		val, err = v.MarshalBinary()
		if err != nil {
			return err
		}
	}

	// Set the value
	if ttl > 0 {
		err = txn.badgerTxn.SetEntry(badger.NewEntry(kb, val).WithTTL(ttl))
	} else {
		err = txn.badgerTxn.Set(kb, val)
	}
	if err != nil {
		return err
	}

	// Set indexes
	if v != nil {
		indexes := v.Indexes()
		if indexes != nil {
			for _, dbIndexKeyGen := range indexes {
				indexKey := dbIndexKeyGen(v)
				indexEntryKey := make([]byte, len(kb)+len(indexKey))
				copy(indexEntryKey[:], indexKey)
				copy(indexEntryKey[len(indexKey):], kb)
				err := txn.badgerTxn.Set(indexEntryKey, nil)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ObjDel inserts or updates an node to the DB
func (txn *DBBadgerTxn) ObjDel(k DBK) error {
	kb, err := k.MarshalBinary()
	if err != nil {
		return err
	}
	return txn.ObjDelByKBytes(kb)
}

// ObjDelByKBytes inserts or updates an node to the DB
func (txn *DBBadgerTxn) ObjDelByKBytes(k []byte) error {
	var err error
	Log.Debugf("DBBadgerTxn.ObjDel: DELTING  -->%s<--", Bytes2Base64(k))

	err = txn.badgerTxn.Delete(k)
	if err != nil {
		return err
	}
	// indexes := obj.Indexes()
	// if indexes != nil {
	// 	for _, dbIndexKeyGen := range indexes {
	// 		indexKey := dbIndexKeyGen(obj)
	// 		indexEntryKey := make([]byte, len(pk)+len(indexKey))
	// 		copy(indexEntryKey[:], indexKey)
	// 		copy(indexEntryKey[len(indexKey):], pk)
	// 		err := txn.badgerTxn.Set(indexEntryKey, nil)
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	return nil
}

// ObjDelAll deletes all objects in db
func (txn *DBBadgerTxn) ObjDelAll(filter func(k, v []byte) (bool, error)) (err error) {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	opts.PrefetchSize = 1000
	it := txn.badgerTxn.NewIterator(opts)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		k := it.Item().KeyCopy(nil)
		if filter != nil {
			v, err := it.Item().ValueCopy(nil)
			if err != nil {
				return err
			}
			doIt, err := filter(k, v)
			if err != nil {
				return err
			}
			if !doIt {
				continue
			}
		}
		if Log.IsLevelEnabled(logrus.DebugLevel) {
			Log.Debugf("DBBadgerTxn.ObjDelAll: DELTING  -->%s<--", Bytes2Base64(k))
		}
		err = txn.badgerTxn.Delete(k)
		if err != nil {
			return err
		}
	}

	return nil
}

// ObjExists returns true if the DB has the node by primary key
func (txn *DBBadgerTxn) ObjExists(k DBK) (ok bool, err error) {
	kb, err := k.MarshalBinary()
	if err != nil {
		return false, err
	}
	_, err = txn.badgerTxn.Get(kb)
	switch err {
	case badger.ErrKeyNotFound:
		ok, err = false, nil
	case nil:
		ok, err = true, nil
	}

	return
}

// DBBadgerKeyIter is an iterator fr badger db
type DBBadgerKeyIter struct {
	unmarshaller func([]byte) error
	prefix       []byte
	badger.Iterator
}

// Next returns the next key from a DBBadgerKeyIter
func (it *DBBadgerKeyIter) Next() ([]byte, bool, error) {
	if !it.ValidForPrefix(it.prefix) {
		return nil, false, nil
	}
	item := it.Item()
	key := item.KeyCopy(nil)

	// err := item.Value(func(v []byte) error {
	// if Log.IsLevelEnabled(logrus.InfoLevel) {
	// 	Log.Info("Iterating on this key " + string(it.prefix))
	// 	Log.Info("Found key             " + string(key))
	// 	Log.Info("Found value           " + string(v))
	// }
	// 	return nil
	// })
	// if err != nil {
	// 	return nil, false, err
	// }

	if it.unmarshaller != nil {
		err := item.Value(it.unmarshaller)
		if err != nil {
			return nil, false, err
		}
	}
	it.Iterator.Next()
	return key, true, nil
}

// Close closes a DBBadgerKeyIter
func (it *DBBadgerKeyIter) Close() {
	it.Iterator.Close()
}

// KIterGet returns a KeyIter for the given prefix
func (txn *DBBadgerTxn) KIterGet(prefix []byte, fwd bool, prefetchSize int, unmarshaller func([]byte) error) DBKeyIter {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = unmarshaller != nil
	opts.Reverse = !fwd
	opts.PrefetchSize = prefetchSize
	it := txn.badgerTxn.NewIterator(opts)
	it.Seek(prefix)

	return &DBBadgerKeyIter{
		unmarshaller: unmarshaller,
		prefix:       prefix,
		Iterator:     *it,
	}
}

// Seek move to that prefix
func (it *DBBadgerKeyIter) Seek(prefix []byte) {
	it.Iterator.Seek(prefix)
}

// DBBadgerKVIter is an iterator for badger db
type DBBadgerKVIter struct {
	unmarshaller func([]byte) error
	prefix       []byte
	badger.Iterator
}

// Next returns the next key from a DBBadgerKeyIter
func (it *DBBadgerKVIter) Next() ([]byte, []byte, bool, error) {
	if !it.ValidForPrefix(it.prefix) {
		return nil, nil, false, nil
	}
	item := it.Item()
	key := item.KeyCopy(nil)
	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, nil, false, err
	}

	// err := item.Value(func(v []byte) error {
	// if Log.IsLevelEnabled(logrus.InfoLevel) {
	// 	Log.Info("Iterating on this key " + string(it.prefix))
	// 	Log.Info("Found key             " + string(key))
	// 	Log.Info("Found value           " + string(v))
	// }
	// 	return nil
	// })
	// if err != nil {
	// 	return nil, false, err
	// }

	if it.unmarshaller != nil {
		err := item.Value(it.unmarshaller)
		if err != nil {
			return nil, nil, false, err
		}
	}
	it.Iterator.Next()
	return key, value, true, nil
}

// Seek move to that prefix
func (it *DBBadgerKVIter) Seek(prefix []byte) {
	it.Iterator.Seek(prefix)
}

// Close closes a DBBadgerKeyIter
func (it *DBBadgerKVIter) Close() {
	it.Iterator.Close()
}

// KVIterGet returns a KeyIter for the given prefix
func (txn *DBBadgerTxn) KVIterGet(prefix []byte, fwd bool, prefetchSize int, unmarshaller func([]byte) error) DBStringIter {
	opts := badger.DefaultIteratorOptions
	opts.PrefetchValues = true
	opts.PrefetchSize = prefetchSize
	opts.Reverse = !fwd
	it := txn.badgerTxn.NewIterator(opts)
	it.Seek(prefix)

	return &DBBadgerKVIter{
		unmarshaller: unmarshaller,
		prefix:       prefix,
		Iterator:     *it,
	}
}

// Close implements the DB interface. It closes the connection to the underlying
// DBBadger database as well as invoking the context's cancel function.
func (bdb *DBBadger) Close() error {
	bdb.cancelFunc()
	return bdb.DB.Close()
}

// runGC triggers the garbage collection for the DBBadger backend database. It
// should be run in a goroutine.
func (bdb *DBBadger) runGC() {
	defer SentryRecover("DBBadger.runGC")
	ticker := time.NewTicker(badgerGCInterval)
	for {
		select {
		case <-ticker.C:
			err := bdb.DB.RunValueLogGC(badgerDiscardRatio)
			if err != nil {
				// don't report error when GC didn't result in any cleanup
				if err == badger.ErrNoRewrite {
					bdb.logger.Debugf("no DBBadger GC occurred: %v", err)
				} else {
					bdb.logger.Errorf("failed to GC DBBadger: %v", err)
				}
			}

		case <-bdb.ctx.Done():
			return
		}
	}
}

var backupTimeMetrc, restoreTimeMetrc prometheus.Histogram

func init() {
	backupTimeMetrc = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "db_badger_snapshot_time",
			Help: "time to snapshot a badger db",
		})
	PromRegisterCollector(backupTimeMetrc)
}

// SnapshotMake makes a new snapshot of this database
func (bdb *DBBadger) SnapshotMake() *DBBadgerSnapshot {
	snapshot := &DBBadgerSnapshot{bdb: bdb, stream: nil}
	snapshot.Snap()
	return snapshot
}

// DBBadgerSnapshot a snapshot of the badger db
type DBBadgerSnapshot struct {
	bdb    *DBBadger
	stream *badger.Stream
}

// Snap freezes the snapshot
func (ss *DBBadgerSnapshot) Snap() error {
	// db.NewStreamAt(readTs) for managed mode.
	stream := ss.bdb.DB.NewStream()
	// stream.NumGo = 16 // Set number of goroutines to use for iteration.
	// stream.ChooseKey = nil
	// ss.stream.KeyToList = nil
	ss.stream = stream
	return nil
}

// Write writes a snapshot to writer
func (ss *DBBadgerSnapshot) Write(w io.Writer) error {
	start := time.Now()
	Log.Warnf("DBBadgerSnapshot.Backup: starting")

	// Run the stream (discard backup time)
	_, err := ss.stream.Backup(w, 1)
	if err != nil {
		err = fmt.Errorf("error doing badger db backup: %w", err)
		return err
	}

	duration := time.Since(start)
	Log.Warnf("DBBadgerSnapshot.Backup: took %s", duration.String())
	backupTimeMetrc.Observe(float64(duration))

	return nil
}

// Release frees up a snapshot
func (ss *DBBadgerSnapshot) Release() {
	return
}

// Read restores the snapshot
func (ss *DBBadgerSnapshot) Read(r io.Reader) error {
	start := time.Now()
	Log.Warn("DBBadgerSnapshot.Restore: starting")
	err := ss.bdb.DB.Load(r, 256)
	if err != nil {
		return err
	}
	duration := time.Since(start)
	Log.Warnf("DBBadgerSnapshot.Restore: took %s", duration.String())
	return nil
}
