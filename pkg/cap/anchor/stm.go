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
func (t Txn) WalkLongestSubpath(p Path) (a Anchor, _ error) {
	path := t.sched.root.bind(subpath(p))

	v, err := t.txn.LongestPrefix(t.sched.anchors, "id_prefix", path)
	if v != nil {
		a = v.(Anchor)
	}

	return a, err
}

// GetOrCreate returns the anchor along the supplied path, creating it if it
// does not exist. It does not attempt to create any missing anchors along p.
//
// This is potentially a write operation.
func (t Txn) GetOrCreate(p Path) (Anchor, error) {
	path := t.sched.root.bind(subpath(p))
	v, err := t.txn.First(t.sched.anchors, "id", path)
	if err != nil {
		return Anchor{}, err
	}

	if v != nil {
		return v.(Anchor), nil
	}

	return t.createAnchor(path)
}

func (t Txn) createAnchor(path Path) (a Anchor, err error) {
	a = Anchor{
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
		return !parent.IsChild(v.(Anchor).Path())
	})
}

type index struct{}

func (index) FromObject(obj interface{}) (bool, []byte, error) {
	if a, ok := obj.(Anchor); ok {
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
