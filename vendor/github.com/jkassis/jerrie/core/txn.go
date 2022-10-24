package core

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// Txn is a goroutine control structure that allows subgoroutines to commit, cancel, or expire
// as a group
type Txn struct {
	sync.Mutex
	doneCh  chan struct{}
	wg      *sync.WaitGroup
	status  Status
	reports []string
}

// Status is a return type for the Txn object
type Status int

const (
	// StatusNil indicates the txn is not resolved
	StatusNil Status = iota

	// StatusCommitted indicates the txn was committed
	StatusCommitted Status = iota

	// StatusCancelled indicates the txn was cancelled
	StatusCancelled Status = iota

	// StatusExpired indicates the txn expired
	StatusExpired Status = iota
)

// TxnMake returns a new empty txn
func TxnMake() *Txn {
	return &Txn{
		doneCh:  make(chan struct{}),
		wg:      &sync.WaitGroup{},
		status:  StatusNil,
		reports: make([]string, 0),
	}
}

// ReportGet returns the complete report for the transactions
func (t *Txn) ReportGet() string {
	t.Lock()
	sort.Strings(t.reports)
	result := strings.Join(t.reports, "\n")
	t.Unlock()
	return "TXN Status:\n" + result
}

// Report adds a message to the reports
func (t *Txn) Report(message string) {
	t.Lock()
	t.reports = append(t.reports, message)
	t.Unlock()
}

// Commit causes Status to return TxnStatusCommitted
func (t *Txn) Commit() {
	if t.status != StatusNil {
		return
	}
	t.status = StatusCommitted
	close(t.doneCh)
}

// Cancel causes Status to return TxnStatusCanceled
func (t *Txn) Cancel() {
	if t.status != StatusNil {
		return
	}
	t.status = StatusCancelled
	close(t.doneCh)
}

// Expire causes Status to return TxnStatusCommitted
func (t *Txn) Expire() {
	if t.status != StatusNil {
		return
	}
	t.status = StatusExpired
	close(t.doneCh)
}

// Block adds to the internal waitgroup to block the transaction until the corresponding UnBlock
func (t *Txn) Block() {
	t.wg.Add(1)
}

// Unblock releases the txn
func (t *Txn) Unblock() {
	t.wg.Done()
}

// Status blocks until the transaction is resolved
func (t *Txn) Status() Status {
	return t.status
}

// Done blocks until the transaction is resolved
func (t *Txn) Done() chan struct{} {
	return t.doneCh
}

// Wait waits for controlled goroutines to settle the transaction
func (t *Txn) Wait() {
	t.wg.Wait()
}

//
// The rest of this just makes it easy to bind a txn to a context
//
const txnContextKey contextKey = 0

// TxnWithContext returns a context with an embedded txn
func TxnWithContext(ctx context.Context) (ctxWithTxn context.Context, t *Txn) {
	// make the txn
	t = TxnMake()
	ctxWithTxn = context.WithValue(ctx, txnContextKey, t)

	// listen for my done or context done
	go func() {
		select {
		case <-ctxWithTxn.Done():
			if ctxWithTxn.Err() == context.DeadlineExceeded {
				t.Expire()
			} else {
				t.Cancel()
			}
		case <-t.doneCh:
		}
	}()
	return ctxWithTxn, t
}

// FromContext pulls the Txn out of the context
func FromContext(ctx context.Context) *Txn {
	txn := ctx.Value(txnContextKey)
	return txn.(*Txn)
}
