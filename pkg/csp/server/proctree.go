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

func findParent(n *ProcNode, id uint32) (*ProcNode, bool) {
	// child is first node to the left, immediate child
	if n.Left != nil && n.Left.Id == id {
		return n, false
	}

	if n.Left != nil {
		x, childInRight := findParent(n.Left, id)
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
	if n.Right != nil && n.Right.Id == id {
		return nil, true
	}

	// explore immediate sibling
	if n.Right != nil {
		return findParent(n.Right, id)
	}
	return nil, false
}

type ProcNode struct {
	Id    uint32
	Left  *ProcNode
	Right *ProcNode
}
