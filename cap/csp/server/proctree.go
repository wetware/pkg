package csp_server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	api "github.com/wetware/pkg/api/process"
)

const INIT_PID = 1

// ProcTree represents the process tree of an executor.
// It is represented a binary tree, in which the left branch of a node
// represents a child process, while the right branch represents a
// sibling process (shares the same parent).
type ProcTree struct {
	// TODO move context out of tree
	Ctx context.Context
	// PIDC is a couter that increases to assign new PIDs.
	PIDC AtomicCounter
	// TPC keeps track of the number of processes in the tree.
	TPC AtomicCounter
	// Root of the process tree.
	Root *ProcNode
	// Map of processes associated to their PIDs. MUST be initialized.
	// Map map[uint32]api.Process_Server
	Map *sync.Map
	// Mutex to ensure thread safety.
	Mut *sync.RWMutex
}

// NewProcTree is the default constuctor for ProcTree, but it may
// also be maually constructed.
func NewProcTree(ctx context.Context) ProcTree {
	return ProcTree{
		Ctx:  ctx,
		PIDC: NewAtomicCounter(INIT_PID),
		TPC:  NewAtomicCounter(1),
		Root: &ProcNode{Pid: INIT_PID},
		Map:  &sync.Map{},
		Mut:  &sync.RWMutex{},
	}
}

// ppidOrInit checks for a process with pid=ppid and returns
// ppid if found, INIT_PID otherwise.
func (pt *ProcTree) PpidOrInit(ppid uint32) uint32 {
	if ppid == 0 {
		return INIT_PID
	} else {
		// Default INIT_PID as a parent.
		if _, ok := pt.Load(ppid); !ok {
			return INIT_PID
		}
	}
	return ppid
}

// NextPid returns the next avaiable PID and ensures it does not collide
// with any existing processes.
func (pt *ProcTree) NextPid() uint32 {
	pid := pt.PIDC.Inc()
	_, col := pt.Load(pid)
	for col {
		pid := pt.PIDC.Inc()
		_, col = pt.Load(pid)
	}
	return pid
}

// Kill recursively kills a process and it's children
func (pt *ProcTree) Kill(pid uint32) {
	pt.Mut.Lock()
	defer pt.Mut.Unlock()
	// Can't kill root process.
	if pid == pt.Root.Pid {
		return
	}

	n := pop(pt.Root, pid)
	p, ok := pt.Load(pid)
	if ok && p != nil {
		pt.TPC.Dec()
		stop(pt.Ctx, p)
		pt.Delete(pid)
	}

	// Kill all subprocesses.
	if n != nil {
		pt.kill(n.Left)
	}
}

// kill recursively kills process n, its siblings and children
func (pt *ProcTree) kill(n *ProcNode) {
	if n == nil {
		return
	}
	p, ok := pt.Load(n.Pid)
	if ok && p != nil {
		pt.TPC.Dec()
		stop(pt.Ctx, p)
		pt.Delete(n.Pid)
	}
	pt.kill(n.Left)
	pt.kill(n.Right)
}

// stop a process in a specific way based on its implementation type.
func stop(ctx context.Context, p api.Process_Server) {
	// *process p calls this function from its Kill implementation
	// thus we must avoid infinite recursivity. The process is
	// killed with p.cancel() instead.
	if ps, ok := p.(*process); ok {
		fmt.Printf("killing process %d\n", p.(*process).Pid)
		ps.cancel()
	} else {
		// Generic implementation.
		p.Kill(ctx, api.Process_kill{})
	}
}

// Pop removes the node with PID=pid and replaces it with a sibling
// in the process tree.
func (pt ProcTree) Pop(pid uint32) *ProcNode {
	pt.Mut.Lock()
	defer pt.Mut.Unlock()
	return pop(pt.Root, pid)
}

func pop(n *ProcNode, pid uint32) *ProcNode {
	// Root proc.
	if pid == n.Pid {
		return nil
	}

	// Find the parent.
	parent, _ := findParent(n, pid)

	// Orphaned node.
	if parent == nil {
		return find(n, pid)
	}

	child := parent.Left
	// This case should never occur if FindParent is correct.
	if child == nil {
		return nil
	}

	// Child is immediate left branch.
	if child.Pid == pid {
		result := child
		parent.Left = child.Right
		return result
	}

	// Descend throught the rightest branch.
	sibling := child.Right
	for sibling != nil && sibling.Pid != pid {
		child, sibling = sibling, sibling.Right
	}

	// Bridge left and right siblings.
	if sibling != nil {
		child.Right = sibling.Right
	}

	return sibling
}

// Find returns a node in the process tree with PID=pid. nil if not found.
func (pt ProcTree) Find(pid uint32) *ProcNode {
	pt.Mut.RLock()
	defer pt.Mut.RUnlock()
	return find(pt.Root, pid)
}

// FindParent returns the parent of the process with PID=pid. nil if not found.
func (pt ProcTree) FindParent(pid uint32) *ProcNode {
	pt.Mut.RLock()
	defer pt.Mut.RUnlock()
	n, _ := findParent(pt.Root, pid)
	return n
}

// Insert creates a node with PID=pid as a child of PID=ppid.
func (pt ProcTree) Insert(pid, ppid uint32) error {
	pt.Mut.Lock()
	err := insert(pt.Root, pid, ppid)
	pt.Mut.Unlock()
	if err == nil {
		pt.TPC.Inc()
	}
	return err
}

// find performs an In-Order Depth First Search of the tree.
func find(n *ProcNode, pid uint32) *ProcNode {
	// Corner case.
	if n == nil || n.Pid == pid {
		return n
	}

	// Explore left branch.
	x := find(n.Left, pid)
	if x != nil {
		return x
	}

	// Explore right branch.
	return find(n.Right, pid)
}

// findParent does a Depth First Search of the tree until
// finding the node with PID=pid, then returns it's parent node.
func findParent(n *ProcNode, pid uint32) (*ProcNode, bool) {
	// Corner case, defaults to being the right-branch node.
	if n == nil || n.Pid == pid {
		return nil, n != nil
	}

	// Explore left branch.
	if n.Left != nil {
		x, childInRight := findParent(n.Left, pid)
		// Node was a children or grandchildren.
		if x != nil {
			return x, false
		} else {
			// Node was a sibling of right.
			if childInRight {
				return n, false
			}
		}
	}

	// Explore immediate sibling.
	return findParent(n.Right, pid)
}

// Insert adds a new node PID to root as a child of PPID.
// If PPID has no children PID will be the immediate child.
// Otherwise it will iterate over the siblings and add it at the end of the chain.
func insert(root *ProcNode, pid, ppid uint32) error {
	n := &ProcNode{
		Pid: pid,
	}

	parent := find(root, ppid)
	if parent == nil {
		return fmt.Errorf(
			"could not insert (pid=%d), parent (ppid=%d) no longer alive",
			pid,
			ppid,
		)
	}
	if parent.Left == nil {
		parent.Left = n
		return nil
	}

	next := parent.Left
	for next.Right != nil {
		next = next.Right
	}
	next.Right = &ProcNode{
		Pid: pid,
	}
	return nil
}

// Trim all orphaned branches.
func (pt ProcTree) Trim(ctx context.Context) {
	for pid := range pt.MapSnapshot() {
		if pt.Find(pid) == nil {
			pt.Kill(pid)
		}
	}
}

func (pt ProcTree) AddToMap(pid uint32, p api.Process_Server) {
	pt.Store(pid, p)
}

// Load a process from the map.
func (pt ProcTree) Load(pid uint32) (api.Process_Server, bool) {
	v, ok := pt.Map.Load(pid)
	if !ok {
		return nil, ok
	}
	return v.(api.Process_Server), ok
}

// Store a process on the map.
func (pt ProcTree) Store(pid uint32, p api.Process_Server) {
	pt.Map.Store(pid, p)
}

// Delete a process from the map.
func (pt ProcTree) Delete(pid uint32) {
	pt.Map.Delete(pid)
}

// MapSnapshots returns a snapshot of Map as a native map.
func (pt ProcTree) MapSnapshot() map[uint32]api.Process_Server {
	snapshot := make(map[uint32]api.Process_Server)
	pt.Map.Range(func(key, value any) bool {
		snapshot[key.(uint32)] = value.(api.Process_Server)
		return true
	})
	return snapshot
}

// ProcNode represents a process in the process tree.
type ProcNode struct {
	// Pid contais the Process ID.
	Pid uint32
	// Left contains a child process.
	Left *ProcNode
	// Right contains a sibling process.
	Right *ProcNode
}

func (n *ProcNode) String() string {
	var left, right string
	if n.Left != nil {
		left = fmt.Sprint(n.Left.Pid)
	} else {
		left = "nil"
	}
	if n.Right != nil {
		right = fmt.Sprint(n.Right.Pid)
	} else {
		right = "nil"
	}
	return fmt.Sprintf("{pid=%d, left=%s, right=%s}", n.Pid, left, right)
}

// AtomicCounter is an atomic counter that increases the
type AtomicCounter struct {
	n *uint32
}

func NewAtomicCounter(start uint32) AtomicCounter {
	return AtomicCounter{n: &start}
}

// Increase by 1.
func (p AtomicCounter) Inc() uint32 {
	return atomic.AddUint32(p.n, 1)
}

// Decrease by 1.
func (p AtomicCounter) Dec() uint32 {
	return atomic.AddUint32(p.n, ^uint32(0))
}

// Get current value.
func (p AtomicCounter) Get() uint32 {
	return atomic.LoadUint32(p.n)
}
