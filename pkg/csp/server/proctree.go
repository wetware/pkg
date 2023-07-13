package server

type ProcTree struct {
	IDC  ProcConter
	PC   ProcConter
	Root *ProcNode
}

// func (pt ProcTree) Pop(id uint32) *Node {

// FindParent returns the parent of the process with PID=pid. nil if not found.
func (pt ProcTree) FindParent(pid uint32) *ProcNode {
	n, _ := findParent(pt.Root, pid)
	return n
}

func (pt ProcTree) FindParent(id uint32) *ProcNode {
	n, _ := findParent(pt.Root, id)
	return n
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
	return nil, false
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
