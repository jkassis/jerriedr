package core

import (
	"errors"
	"sync"
)

// LazyLoader publishes the object it is waiting for when it is Set
type LazyLoader struct {
	Publisher
	sync.Mutex
	isSet bool
	obj   interface{}
}

// Init allows a LazyLoader to be inited with an object
func (lazyLoader *LazyLoader) Init(obj interface{}) {
	lazyLoader.Publisher.Init()

	defer lazyLoader.Unlock()
	lazyLoader.Lock()
	lazyLoader.obj = obj
	lazyLoader.isSet = obj != nil
}

// ObjGet just returns the current value of the object or error if not set
func (lazyLoader *LazyLoader) ObjGet() (interface{}, error) {
	if !lazyLoader.isSet {
		return nil, errors.New("lazyLoader has not be set")
	}
	return lazyLoader.obj, nil
}

// ObjSet registers the DocType and notifies blocked subscribers
func (lazyLoader *LazyLoader) ObjSet(txn *Txn, obj interface{}) {
	// set the object
	lazyLoader.Lock()
	original := lazyLoader.obj
	if obj == nil {
		Log.Error("lazyLoader setting to nil")
	}
	lazyLoader.obj = obj
	lazyLoader.isSet = true
	lazyLoader.Unlock()

	// block the txn since this might rollback
	txn.Block()

	// go wait for the txn to complete
	go func() {
		<-txn.Done()
		// should we commit?
		if txn.Status() != StatusCommitted {
			// no. rollback.
			lazyLoader.Lock()
			lazyLoader.obj = original
			lazyLoader.Unlock()
		}
		txn.Unblock()
	}()

	// Notify subscribers
	lazyLoader.Publish(txn, obj)
	lazyLoader.Reset()
}

// ObjGetOrWait blocks until an DocType is registered
func (lazyLoader *LazyLoader) ObjGetOrWait(txn *Txn) (interface{}, error) {
	// get the object
	lazyLoader.Lock()
	obj := lazyLoader.obj

	// was it set?
	if obj != nil {
		// yes. return it
		lazyLoader.Unlock()
		return obj, nil
	}

	// wasn't set... make a subscription
	subscription := lazyLoader.Sub(make(chan interface{}, 1))
	lazyLoader.Unlock()

	// block until the object is set or the txn is cancelled
	select {
	case val := <-subscription:
		return val, nil
	case <-txn.Done():
		return nil, errors.New("cancelled")
	}
}

// IsSet is true if the object is loaded
func (lazyLoader *LazyLoader) IsSet() bool {
	lazyLoader.Lock()
	defer lazyLoader.Unlock()
	return lazyLoader.isSet
}
