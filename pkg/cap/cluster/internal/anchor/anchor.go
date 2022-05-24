package anchor

import (
	"fmt"
	"reflect"

	"capnproto.org/go/capnp/v3"
	"github.com/hashicorp/go-memdb"
	"github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/stm"
)

var anchorSchema = memdb.TableSchema{
	Name: "anchor",
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: index{},
		},
	},
}

type RootAnchor struct {
	stm.Scheduler
	anchors stm.TableRef
}

func NewRootAnchor() RootAnchor {
	var (
		f       stm.Factory
		anchors = f.Register("anchor", &anchorSchema)
	)

	sched, err := f.NewScheduler()
	if err != nil {
		panic(err)
	}

	return RootAnchor{
		Scheduler: sched,
		anchors:   anchors,
	}
}

func (root RootAnchor) Txn(write bool) Txn {
	return Txn{
		Txn:     root.Scheduler.Txn(write),
		anchors: root.anchors,
	}
}

func (root RootAnchor) Snapshot() RootAnchor {
	return RootAnchor{
		Scheduler: root.Scheduler.Snapshot(),
		anchors:   root.anchors,
	}
}

type ChildAllocator interface {
	NewChildren(int32) (capnp.StructList[cluster.Anchor_Child], error)
}

type Txn struct {
	stm.Txn
	anchors stm.TableRef
}

func (t Txn) Finish() {
	// caller is expected to call Commit(), at which point
	// Abort() becomes a nop.
	t.Abort()
}

// Walk the path to the specified anchor, creating new anchors
// along the way, as needed.  The path argument MUST be valid,
// and in canonical form.
func (t Txn) Walk(path Path) (cluster.Anchor, error) {
	// XXX: remember to call AddRef()
	panic("NOT IMPLEMENTED")
}

func (t Txn) BindChildren(a ChildAllocator) error {
	// XXX: remember to call AddRef() for each
	panic("NOT IMPLEMENTED")
}

type index struct{}

func (index) FromObject(obj interface{}) (bool, []byte, error) {
	path, err := argsToPath(obj)
	if err != nil {
		return false, nil, errType(obj)
	}

	return true, path.index(), nil
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
		return path, nil
	}

	return Path{}, errNArgs(args)
}

func errType(v any) error {
	return fmt.Errorf("invalid type: %s", reflect.TypeOf(v))
}

func errNArgs(args []any) error {
	return fmt.Errorf("expected one argument (got %d)", len(args))
}
