package stm

import "github.com/hashicorp/go-memdb"

// Txn is a transaction against a Scheduler instance.
// This can be a read or write transaction.
type Txn struct {
	txn *memdb.Txn
}

// TrackChanges enables change tracking for the transaction. If called at any
// point before commit, subsequent mutations will be recorded and can be
// retrieved using ChangeSet. Once this has been called on a transaction it
// can't be unset. As with other Txn methods it's not safe to call this from a
// different goroutine than the one making mutations or committing the
// transaction.
func (t Txn) TrackChanges() {
	t.txn.TrackChanges()
}

// Abort is used to cancel this transaction.
// This is a noop for read transactions,
// already aborted or committed transactions.
func (t Txn) Abort() {
	t.txn.Abort()
}

// Commit is used to finalize this transaction.
// This is a noop for read transactions,
// already aborted or committed transactions.
func (t Txn) Commit() {
	t.txn.Commit()
}

// Insert is used to add or update an object into the given table.
//
// When updating an object, the obj provided should be a copy rather
// than a value updated in-place. Modifying values in-place that are already
// inserted into MemDB is not supported behavior.
func (t Txn) Insert(table TableRef, v any) error {
	return t.txn.Insert(table.name, v)
}

// Delete is used to delete a single object from the given table.
// This object must already exist in the table.
func (t Txn) Delete(table TableRef, v any) error {
	return t.txn.Delete(table.name, v)
}

// DeletePrefix is used to delete an entire subtree based on a prefix.
// The given index must be a prefix index, and will be used to perform
// a scan and enumerate the set of objects to delete.  These will be
// removed from all other indexes, and then a special prefix operation
// will delete the objects from the given index in an efficient subtree
// delete operation.
//
// This is useful when you have a very large number of objects indexed
// by the given index, along with a much smaller number of entries in
// the other indexes for those objects.
func (t Txn) DeletePrefix(table TableRef, prefix_index string, prefix string) (bool, error) {
	return t.txn.DeletePrefix(table.name, prefix_index, prefix)
}

// DeleteAll is used to delete all the objects in a given table
// matching the constraints on the index
func (t Txn) DeleteAll(table TableRef, index string, args ...any) (int, error) {
	return t.txn.DeleteAll(table.name, index, args...)
}

// FirstWatch is used to return the first matching object for
// the given constraints on the index along with the watch channel.
//
// Note that all values read in the transaction form a consistent snapshot
// from the time when the transaction was created.
//
// The watch channel is closed when a subsequent write transaction
// has updated the result of the query. Since each read transaction
// operates on an isolated snapshot, a new read transaction must be
// started to observe the changes that have been made.
//
// If the value of index ends with "_prefix", FirstWatch will perform a prefix
// match instead of full match on the index. The registered indexer must implement
// PrefixIndexer, otherwise an error is returned.
func (t Txn) FirstWatch(table TableRef, index string, args ...any) (<-chan struct{}, any, error) {
	return t.txn.FirstWatch(table.name, index, args...)
}

// LastWatch is used to return the last matching object for
// the given constraints on the index along with the watch channel.
//
// Note that all values read in the transaction form a consistent snapshot
// from the time when the transaction was created.
//
// The watch channel is closed when a subsequent write transaction
// has updated the result of the query. Since each read transaction
// operates on an isolated snapshot, a new read transaction must be
// started to observe the changes that have been made.
//
// If the value of index ends with "_prefix", LastWatch will perform a prefix
// match instead of full match on the index. The registered indexer must implement
// PrefixIndexer, otherwise an error is returned.
func (t Txn) LastWatch(table TableRef, index string, args ...any) (<-chan struct{}, any, error) {
	return t.txn.LastWatch(table.name, index, args...)
}

// First is used to return the first matching object for
// the given constraints on the index.
//
// Note that all values read in the transaction form a consistent snapshot
// from the time when the transaction was created.
func (t Txn) First(table TableRef, index string, args ...any) (any, error) {
	return t.txn.First(table.name, index, args...)
}

// Last is used to return the last matching object for
// the given constraints on the index.
//
// Note that all values read in the transaction form a consistent snapshot
// from the time when the transaction was created.
func (t Txn) Last(table TableRef, index string, args ...any) (any, error) {
	return t.txn.Last(table.name, index, args...)
}

// LongestPrefix is used to fetch the longest prefix match for the given
// constraints on the index. Note that this will not work with the memdb
// StringFieldIndex because it adds null terminators which prevent the
// algorithm from correctly finding a match (it will get to right before the
// null and fail to find a leaf node). This should only be used where the prefix
// given is capable of matching indexed entries directly, which typically only
// applies to a custom indexer. See the unit test for an example.
//
// Note that all values read in the transaction form a consistent snapshot
// from the time when the transaction was created.
func (t Txn) LongestPrefix(table TableRef, index string, args ...any) (any, error) {
	return t.txn.LongestPrefix(table.name, index, args...)
}

// Get is used to construct a ResultIterator over all the rows that match the
// given constraints of an index. The index values must match exactly (this
// is not a range-based or prefix-based lookup) by default.
//
// Prefix lookups: if the named index implements PrefixIndexer, you may perform
// prefix-based lookups by appending "_prefix" to the index name. In this
// scenario, the index values given in args are treated as prefix lookups. For
// example, a StringFieldIndex will match any string with the given value
// as a prefix: "mem" matches "memdb".
//
// See the documentation for ResultIterator to understand the behaviour of the
// returned ResultIterator.
func (t Txn) Get(table TableRef, index string, args ...any) (memdb.ResultIterator, error) {
	return t.txn.Get(table.name, index, args...)
}

// GetReverse is used to construct a Reverse ResultIterator over all the
// rows that match the given constraints of an index.
// The returned ResultIterator's Next() will return the next Previous value.
//
// See the documentation on Get for details on arguments.
//
// See the documentation for ResultIterator to understand the behaviour of the
// returned ResultIterator.
func (t Txn) GetReverse(table TableRef, index string, args ...any) (memdb.ResultIterator, error) {
	return t.txn.GetReverse(table.name, index, args...)
}

// LowerBound is used to construct a ResultIterator over all the the range of
// rows that have an index value greater than or equal to the provide args.
// Calling this then iterating until the rows are larger than required allows
// range scans within an index. It is not possible to watch the resulting
// iterator since the radix tree doesn't efficiently allow watching on lower
// bound changes. The WatchCh returned will be nill and so will block forever.
//
// If the value of index ends with "_prefix", LowerBound will perform a prefix match instead of
// a full match on the index. The registered index must implement PrefixIndexer,
// otherwise an error is returned.
//
// See the documentation for ResultIterator to understand the behaviour of the
// returned ResultIterator.
func (t Txn) LowerBound(table TableRef, index string, args ...any) (memdb.ResultIterator, error) {
	return t.txn.LowerBound(table.name, index, args...)
}

// ReverseLowerBound is used to construct a Reverse ResultIterator over all the
// the range of rows that have an index value less than or equal to the
// provide args.  Calling this then iterating until the rows are lower than
// required allows range scans within an index. It is not possible to watch the
// resulting iterator since the radix tree doesn't efficiently allow watching
// on lower bound changes. The WatchCh returned will be nill and so will block
// forever.
//
// See the documentation for ResultIterator to understand the behaviour of the
// returned ResultIterator.
func (t Txn) ReverseLowerBound(table TableRef, index string, args ...any) (memdb.ResultIterator, error) {
	return t.txn.ReverseLowerBound(table.name, index, args...)
}

// Changes returns the set of object changes that have been made in the
// transaction so far. If change tracking is not enabled it wil always return
// nil. It can be called before or after Commit. If it is before Commit it will
// return all changes made so far which may not be the same as the final
// Changes. After abort it will always return nil. As with other Txn methods
// it's not safe to call this from a different goroutine than the one making
// mutations or committing the transaction. Mutations will appear in the order
// they were performed in the transaction but multiple operations to the same
// object will be collapsed so only the effective overall change to that object
// is present. If transaction operations are dependent (e.g. copy object X to Y
// then delete X) this might mean the set of mutations is incomplete to verify
// history, but it is complete in that the net effect is preserved (Y got a new
// value, X got removed).
func (t Txn) Changes() memdb.Changes {
	return t.txn.Changes()
}

// Defer is used to push a new arbitrary function onto a stack which
// gets called when a transaction is committed and finished. Deferred
// functions are called in LIFO order, and only invoked at the end of
// write transactions.
func (t Txn) Defer(fn func()) {
	t.txn.Defer(fn)
}

// Snapshot creates a snapshot of the current state of the transaction.
// Returns a new read-only transaction or nil if the transaction is
// already aborted or committed.
func (t Txn) Snapshot() Txn {
	return Txn{
		txn: t.txn.Snapshot(),
	}
}
