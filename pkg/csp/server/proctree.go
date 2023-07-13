package server

type ProcTree struct {
	IDC  ProcConter
	PC   ProcConter
	Root *ProcNode
}

// func (pt ProcTree) Pop(id uint32) *Node {

// }

func (pt ProcTree) FindParent(id uint32) *ProcNode {
	n, _ := findParent(pt.Root, id)
	return n
}

func findParent(n *ProcNode, pid uint32) (*ProcNode, bool) {
	// child is first node to the left, immediate child
	if n.Left != nil && n.Left.Pid == pid {
		return n, false
	}

	if n.Left != nil {
		x, childInRight := findParent(n.Left, pid)
		// node was a children or grandchildren
		if x != nil {
			return x, false
		} else {
			// node was a sibling of right
			if childInRight {
				return n, false
			}
		}
	}

	// immeadiate sibling had the id
	if n.Right != nil && n.Right.Pid == pid {
		return nil, true
	}

	// explore immediate sibling
	if n.Right != nil {
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
