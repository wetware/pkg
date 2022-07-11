package anchor

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-memdb"
	"github.com/wetware/ww/pkg/stm"
)

var schema = memdb.TableSchema{
	Name: "anchor",
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: index{},
		},
	},
}

type Scheduler struct {
	sched   stm.Scheduler
	anchors stm.TableRef
	root    Path
}

func NewScheduler(root Path) Scheduler {
	var (
		f       stm.Factory
		anchors = f.Register("anchor", &schema)
	)

	// Error is always nil since the scheduler is freshly instantiated.
	sched, _ := f.NewScheduler()

	return Scheduler{
		sched:   sched,
		anchors: anchors,
		root:    root,
	}
}

// Parent returns the scheduler scoped to the parent path.
func (s Scheduler) Parent() Scheduler {
	return Scheduler{
		sched:   s.sched,
		anchors: s.anchors,
		root:    s.root.bind(parent),
	}
}

// WithSubpath returns the scheduler scoped to the supplied subpath.
func (s Scheduler) WithSubpath(path Path) Scheduler {
	return Scheduler{
		sched:   s.sched,
		anchors: s.anchors,
		root:    s.root.bind(subpath(path)),
	}
}

func (s Scheduler) Txn(write bool) Txn {
	return Txn{
		sched: s,
		txn:   s.sched.Txn(write),
	}
}

type Txn struct {
	sched Scheduler
	txn   stm.Txn
}

func (t Txn) Commit() { t.txn.Commit() }
func (t Txn) Abort()  { t.txn.Abort() }

func (t Txn) Finish() {
	// caller is expected to call Commit(), at which point
	// Abort() becomes a nop.
	t.Abort()
}

// Parent returns the current transaction scoped to the parent path.
func (t Txn) Parent() Txn {
	return Txn{
		sched: t.sched.Parent(),
		txn:   t.txn,
	}
}

// IsOrphan returns true if the anchor at the transaction's root path
// (1) has no children and (2) has no value. Callers MUST NOT rely on
// IsOrphan() to determine if a path has any lingering references.
//
// This is a read-only transaction.
func (t Txn) IsOrphan() bool {
	// root node?
	if t.sched.root.IsRoot() {
		return false
	}

	// has children?
	if it, _ := t.Children(); it.Next() != nil {
		return false
	}

	// has value?
	_, ok := t.LoadValue() // TODO:  implement
	return !ok
}

// LoadValue returns the contents of the anchor located at the current
// path, if any.
func (t Txn) LoadValue() (any, bool) {
	return nil, false // TODO(soon)
}

// Scrub removes the anchor at the transaction's root path from the
// tree.   Any value associated with this path is unreachable after
// Scrub returns, but is not deallocated.
//
// This is a write operation.
func (t Txn) Scrub() error {
	v, err := t.txn.First(t.sched.anchors, "id", t.sched.root)
	if v == nil {
		return err
	}

	if err = t.txn.Delete(t.sched.anchors, v); err != nil {
		return err
	}

	if px := t.Parent(); px.IsOrphan() {
		err = px.Scrub()
	}

	return err
}

// Children returns an iterator over the children of the anchor identified
// by the path parameter.  This is a read-only operation.
func (t Txn) Children() (memdb.ResultIterator, error) {
	it, err := t.txn.Get(t.sched.anchors, "id_prefix", t.sched.root)
	return children(it, t.sched.root), err
}

// WalkLongestSubpath returns the anchor located at the longest subpath
// of p that currently exists, along with the remaining subpath.   If p
// does not match any existing prefix, a zero-value anchor is returned.
//
// This is a read-only operation.
func (t Txn) WalkLongestSubpath(p Path) (a AnchorServer, _ error) {
	path := t.sched.root.bind(subpath(p))

	v, err := t.txn.LongestPrefix(t.sched.anchors, "id_prefix", path)
	if v != nil {
		a = v.(AnchorServer)
	}

	return a, err
}

// GetOrCreate returns the anchor along the supplied path, creating it if it
// does not exist. It does not attempt to create any missing anchors along p.
//
// This is potentially a write operation.
func (t Txn) GetOrCreate(p Path) (AnchorServer, error) {
	path := t.sched.root.bind(subpath(p))
	v, err := t.txn.First(t.sched.anchors, "id", path)
	if err != nil {
		return AnchorServer{}, err
	}

	if v != nil {
		return v.(AnchorServer), nil
	}

	return t.createAnchor(path)
}

func (t Txn) createAnchor(path Path) (a AnchorServer, err error) {
	a = AnchorServer{
		sched:  t.sched.WithSubpath(path),
		anchor: &a,
	}

	err = t.txn.Insert(t.sched.anchors, a)
	return
}

func children(it memdb.ResultIterator, parent Path) *memdb.FilterIterator {
	return memdb.NewFilterIterator(it, func(v interface{}) bool {
		// NOTE:  filter is unusual in that it removes elements
		//        for which the function returns *true*.
		return !parent.IsChild(v.(AnchorServer).Path())
	})
}

type index struct{}

func (index) FromObject(obj interface{}) (bool, []byte, error) {
	if a, ok := obj.(AnchorServer); ok {
		return true, a.sched.root.index(), nil
	}

	return false, nil, errType(obj)
}

func (index) FromArgs(args ...interface{}) ([]byte, error) {
	path, err := argsToPath(args...)
	return path.index(), err
}

func (index) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	path, err := argsToPath(args...)
	return path.index(), err
}

func argsToPath(args ...any) (Path, error) {
	if len(args) != 1 {
		return Path{}, errNArgs(args)
	}

	if path, ok := args[0].(Path); ok {
		return path, path.Err()
	}

	return Path{}, errNArgs(args)
}

func errType(v any) error {
	return fmt.Errorf("invalid type: %s", reflect.TypeOf(v))
}

func errNArgs(args []any) error {
	return fmt.Errorf("expected one argument (got %d)", len(args))
}
