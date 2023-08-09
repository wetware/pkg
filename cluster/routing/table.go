package routing

import (
	"sync"
	"time"

	"github.com/wetware/ww/util/stm"
)

type clock struct {
	time time.Time
	mu   sync.RWMutex
}

type Table struct {
	clock   *clock
	records stm.TableRef
	sched   stm.Scheduler
}

func (c *clock) Load() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.time
}

func (c *clock) Store(val time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time = val
}

func New(t0 time.Time) Table {
	var (
		f     stm.Factory
		clock = &clock{
			time: t0,
		}
	)

	records := f.Register("record", &schema)
	sched, err := f.NewScheduler() // no err since f is freshly instantiated
	if err != nil {
		panic(err)
	}

	return Table{
		clock:   clock,
		records: records,
		sched:   sched,
	}
}

func (table Table) Snapshot() Snapshot {
	return &query{
		records: table.records,
		tx:      table.sched.Txn(false),
	}
}

// Advance the state of the routing table to the current time.
// Expired entries will be evicted from the table.
func (table Table) Advance(t time.Time) {
	if old := table.clock.Load(); t.After(old) {
		defer table.clock.Store(t)

		// Most ticks will not have expired entries, so avoid locking.
		if rx := table.sched.Txn(false); table.expiredRecords(rx, t) {
			wx := table.sched.Txn(true)
			defer wx.Commit()

			table.dropExpired(wx, t)
		}
	}
}

func (table Table) expiredRecords(tx stm.Txn, t time.Time) bool {
	it, err := tx.ReverseLowerBound(table.records, "ttl", t)
	if err != nil {
		panic(err)
	}
	return it != nil && it.Next() != nil
}

func (table Table) dropExpired(wx stm.Txn, t time.Time) {
	it, err := wx.ReverseLowerBound(table.records, "ttl", t)
	if err != nil {
		panic(err)
	}

	for r := it.Next(); r != nil; r = it.Next() {
		if err = wx.Delete(table.records, r); err != nil {
			panic(err)
		}

	}
}

// Upsert inserts a record in the routing table, updating it
// if it already exists.  Returns false if rec is stale.
func (table Table) Upsert(rec Record) bool {
	// Some records are stale, so avoid locking until we're
	// sure to write.
	if rx := table.sched.Txn(false); table.valid(rx, rec) {
		wx := table.sched.Txn(true)
		defer wx.Commit()

		table.upsert(wx, rec)
		return true
	}

	return false
}

func (table Table) valid(tx stm.Txn, rec Record) bool {
	v, err := tx.First(table.records, "id", rec)
	if v == nil {
		return err == nil
	}

	old := v.(Record)

	// Same instance?  Prefer most recent record.
	if old.Server() == rec.Server() {
		return old.Seq() < rec.Seq()
	}

	// Different instance?  Prefer youngest record.  If seq
	// is same, prefer newly-received record.
	return old.Seq() > rec.Seq()
}

func (table Table) upsert(wx stm.Txn, rec Record) {
	err := wx.Insert(table.records, table.withDeadline(rec))
	if err != nil {
		panic(err)
	}
}

// record wraps a Record and provides a stable deadline, calculated
// upon instantiation of the struct.  This is required in order for
// memdb to compute a consistent TTL index.
type record struct {
	Record
	Deadline time.Time
}

func (table Table) withDeadline(rec Record) *record {
	return &record{
		Record:   rec,
		Deadline: table.clock.Load().Add(rec.TTL()),
	}
}

func (r record) PeerBytes() ([]byte, error) {
	return r.Record.(PeerIndex).PeerBytes()
}

func (r record) HostBytes() ([]byte, error) {
	return r.Record.(HostIndex).HostBytes()
}
