package stm

import (
	"capnproto.org/go/capnp/v3"
	"github.com/hashicorp/go-memdb"
	"github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/stm"
)

var anchorSchema = memdb.TableSchema{
	Name:    "anchor",
	Indexes: map[string]*memdb.IndexSchema{
		// "id": {
		// 	Name:    "id",
		// 	Unique:  true,
		// 	Indexer: anchorIndexer{},
		// },
		// "name": {
		// 	Name:    "name",
		// 	Indexer: nameIndexer{},
		// },
		// "path": {
		// 	Name:    "path",
		// 	Unique:  true,
		// 	Indexer: pathIndexer{},
		// },
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

func (sched RootAnchor) Txn(write bool) Txn {
	return Txn{
		Txn:     sched.Scheduler.Txn(write),
		anchors: sched.anchors,
	}
}

func (sched RootAnchor) Snapshot() RootAnchor {
	return RootAnchor{
		Scheduler: sched.Scheduler.Snapshot(),
		anchors:   sched.anchors,
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
// along the way, as needed.
func (t Txn) Walk(path PathIterator) (cluster.Anchor, error) {
	// XXX: remember to call AddRef()
	panic("NOT IMPLEMENTED")
}

func (t Txn) BindChildren(a ChildAllocator) error {
	// XXX: remember to call AddRef() for each
	panic("NOT IMPLEMENTED")
}
