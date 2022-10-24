package core

import (
	"sync"
)

// Publisher can be embedded
type Publisher struct {
	sync.Mutex
	subscribers []chan interface{}
}

// Init Set it up
func (o *Publisher) Init() {
	o.subscribers = make([]chan interface{}, 0)
}

// Sub adds a subscriber
func (o *Publisher) Sub(c chan interface{}) chan interface{} {
	defer o.Unlock()
	o.Lock()
	o.subscribers = append(o.subscribers, c)
	return c
}

// SubCount returns number of subscribers
func (o *Publisher) SubCount() int {
	defer o.Unlock()
	o.Lock()
	return len(o.subscribers)
}

// Unsub releases a subscriber
func (o *Publisher) Unsub(c chan interface{}) {
	defer o.Unlock()
	o.Lock()
	for i, v := range o.subscribers {
		if v == c {
			o.subscribers = append(o.subscribers[:i], o.subscribers[i+1:]...)
			return
		}
	}
}

// Publish sends an evt to all subscribers
func (o *Publisher) Publish(txn *Txn, evt interface{}) {
	defer o.Unlock()
	o.Lock()
	for _, v := range o.subscribers {
		select {
		case v <- evt:
			continue
		case <-txn.Done():
			break
		}
	}
}

// Reset kicks all channels
func (o *Publisher) Reset() {
	defer o.Unlock()
	o.Lock()
	for _, v := range o.subscribers {
		close(v)
	}
	o.subscribers = make([]chan interface{}, 0)
}
