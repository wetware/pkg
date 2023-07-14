package server

import "sync/atomic"

// ProcTree represents the process tree of an executor.
// It is represented a binary tree, in which the left branch of a node
// represents a child process, while the right branch represents a
// sibling process (shares the same parent).
type ProcTree struct {
	// IDC is a couter that increases to assign new PIDs.
	IDC AtomicCounter
	// PC keeps track of the number of processes in the tree.
	PC AtomicCounter
	// Root of the process tree.
	Root *ProcNode
}

// Pop removes the node with PID=pid and replaces it with a sibling
// in the process tree.
func (pt ProcTree) Pop(pid uint32) *ProcNode {
	// Find the parent.
	parent := pt.FindParent(pid)
	if parent == nil {
		return nil
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

// ProcNode represents a process in the process tree.
type ProcNode struct {
	// Pid contais the Process ID.
	Pid uint32
	// Left contains a child process.
	Left *ProcNode
	// Right contains a sibling process.
	Right *ProcNode
}

// AtomicCounter is an atomic counter that increases the
type AtomicCounter struct {
	n uint32
}

// Increase by 1.
func (p AtomicCounter) Inc() uint32 {
	return atomic.AddUint32(&p.n, 1)
}

// Decrease by 2.
func (p AtomicCounter) Dec() uint32 {
	return atomic.AddUint32(&p.n, ^uint32(0))
}
