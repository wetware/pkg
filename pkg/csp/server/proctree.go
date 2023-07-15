package server

import (
	"context"
	"fmt"
	"sync/atomic"

	api "github.com/wetware/ww/api/process"
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
	Map map[uint32]api.Process_Server
}

// NewProcTree is the default constuctor for ProcTree, but it may
// also be maually constructed.
func NewProcTree(ctx context.Context) ProcTree {
	return ProcTree{
		Ctx:  ctx,
		PIDC: NewAtomicCounter(INIT_PID),
		TPC:  NewAtomicCounter(1),
		Root: &ProcNode{Pid: INIT_PID},
		Map:  make(map[uint32]api.Process_Server),
	}
}

// Kill recursively kills a process and it's children
func (pt *ProcTree) Kill(pid uint32) {
	// Can't kill root process.
	if pid == pt.Root.Pid {
		return
	}

	n := pt.Pop(pid)
	p, ok := pt.Map[pid]
	if ok && p != nil {
		pt.TPC.Dec()
		stop(pt.Ctx, p)
		delete(pt.Map, pid)
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
	p, ok := pt.Map[n.Pid]
	if ok && p != nil {
		pt.TPC.Dec()
		stop(pt.Ctx, p)
		delete(pt.Map, n.Pid)
	}
	pt.kill(n.Left)
	pt.kill(n.Right)
}

// stop a process in a specific way based on its implementation type.
func stop(ctx context.Context, p api.Process_Server) {
	fmt.Printf("killing process %d\n", p.(*process).pid)
	// *process p calls this function from its Kill implementation
	// thus we must avoid infinite recursivity. The process is
	// killed with p.cancel() instead.
	if ps, ok := p.(*process); ok {
		ps.cancel()
	} else {
		// Generic implementation.
		p.Kill(ctx, api.Process_kill{})
	}
}

// Pop removes the node with PID=pid and replaces it with a sibling
// in the process tree.
func (pt ProcTree) Pop(pid uint32) *ProcNode {
	// Root proc.
	if pid == pt.Root.Pid {
		return nil
	}

	// Find the parent.
	parent := pt.FindParent(pid)

	// Orphaned node.
	if parent == nil {
		return pt.Find(pid)
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
	return find(pt.Root, pid)
}

// FindParent returns the parent of the process with PID=pid. nil if not found.
func (pt ProcTree) FindParent(pid uint32) *ProcNode {
	n, _ := findParent(pt.Root, pid)
	return n
}

// Insert creates a node with PID=pid as a child of PID=ppid.
func (pt ProcTree) Insert(pid, ppid uint32) error {
	err := insert(pt.Root, pid, ppid)
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

	// Check left node.
	if n.Left != nil && n.Left.Pid == pid {
		return n.Left
	}

	// Explore left branch.
	if n.Left != nil {
		x := find(n.Left, pid)
		if x != nil {
			return x
		}
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

	// Child is first node to the left, immediate child.
	if n.Left != nil && n.Left.Pid == pid {
		return n, false
	}

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
	for pid := range pt.Map {
		if pt.Find(pid) == nil {
			pt.Kill(pid)
		}
	}
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

// Decrease by 2.
func (p AtomicCounter) Dec() uint32 {
	return atomic.AddUint32(p.n, ^uint32(0))
}

// Get current value.
func (p AtomicCounter) Get() uint32 {
	return atomic.LoadUint32(p.n)
}
