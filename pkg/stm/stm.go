package stm

import (
	"github.com/hashicorp/go-memdb"
)

// TableRef is a token that allows the holder to designate a table
// in the Scheduler.   It is effectively an object capability that
// confers access to a given table.  The table's name is kept in a
// private field to prevent unauthorized code from designating the
// table (i.e. unauthorized users have no way to pass in a table).
type TableRef struct {
	name string
}

// Factory populates a memdb schema and generates secure TableRefs.
type Factory struct {
	schema *memdb.DBSchema
}

// Register a table schema for the scheduler.  The returned TableRef
// is required for operations against the table defined by t.  If a
// schema already exists with the supplied name, Register will panic.
func (f *Factory) Register(name string, t *memdb.TableSchema) TableRef {
	// initialize the db schema?
	if f.schema == nil {
		f.schema = &memdb.DBSchema{
			Tables: make(map[string]*memdb.TableSchema, 1),
		}
	}

	if _, exists := f.schema.Tables[name]; exists {
		panic("schema collision")
	}

	// create a cryptographically-secure key for the table.
	f.schema.Tables[name] = t

	// return a TableRef, which can be used to designate the table.
	return TableRef{name: name}
}

// NewScheduler returns a Scheduler, initialized with the tables
// that have have been registered to the Factory before the call.
// Successive calls to NewScheduler produce independent instances
// of Scheduler that do not share any additional state beyond the
// underlying schema and TableRefs.
func (f *Factory) NewScheduler() (s Scheduler, err error) {
	s.db, err = memdb.NewMemDB(f.schema)
	return
}

// Scheduler provides transactions that guarantee the ACID properties
// of Atomicity, Consistency and Isolation.
//
// Objects managed by the scheduler are updated through MVCC.  These
// objects are not copied, and MUST NOT be modified in-place after they
// have been inserted. For the avoidance of doubt, said objects MUST NOT
// be updated after they have been deleted from the scheduler since they
// may still be present in snapshots held by other goroutines.
type Scheduler struct {
	db *memdb.MemDB
}

// Txn is used to start a new transaction in either read or write mode.
// There can only be a single concurrent writer, but any number of readers.
func (s Scheduler) Txn(write bool) Txn {
	return Txn{
		txn: s.db.Txn(write),
	}
}

// Snapshot is used to capture a point-in-time snapshot  of the database that
// will not be affected by any write operations to the existing Scheduler.
//
// If the Scheduler is storing reference-based values (pointers, maps, slices,
// etc.), the Snapshot will not deep copy those values. Therefore, it is still
// unsafe to modify any inserted values in either DB.
func (s Scheduler) Snapshot() Scheduler {
	return Scheduler{
		db: s.db.Snapshot(),
	}
}
