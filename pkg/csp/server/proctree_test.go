package server_test

import (
	"math"
	"testing"

	csp "github.com/wetware/ww/pkg/csp/server"
)

func testProcTree() csp.ProcTree {
	/*
	        0
	        |
	        1
	      /   \
	     2     10
	    / \   /
	   3   6 11
	    \   \
	     4   7
	    /   / \
	   5   8   9
	*/
	root := &csp.ProcNode{
		Pid: 0,
		Left: &csp.ProcNode{
			Pid: 1,
			Left: &csp.ProcNode{
				Pid: 2,
				Left: &csp.ProcNode{
					Pid: 3,
					Right: &csp.ProcNode{
						Pid: 4,
						Left: &csp.ProcNode{
							Pid: 5,
						},
					},
				},
				Right: &csp.ProcNode{
					Pid: 6,
					Right: &csp.ProcNode{
						Pid: 7,
						Left: &csp.ProcNode{
							Pid: 8,
						},
						Right: &csp.ProcNode{
							Pid: 9,
						},
					},
				},
			},
			Right: &csp.ProcNode{
				Pid: 10,
				Left: &csp.ProcNode{
					Pid: 11,
				},
			},
		},
	}
	return csp.ProcTree{
		IDC:  csp.AtomicCounter{},
		PC:   csp.AtomicCounter{},
		Root: root,
	}
}

func TestProcTree_Find(t *testing.T) {
	pt := testProcTree()
	for i := uint32(0); i <= 11; i++ {
		n := pt.Find(i)
		if n == nil {
			t.Fatalf("failed to find node %d", i)
		}
		if n.Pid != i {
			t.Fatalf("found node %d instead of %d", n.Pid, i)
		}
	}
}

func TestProcTree_FindParent(t *testing.T) {
	// child, parent
	matches := [6][2]uint32{
		{8, 7},
		{2, 1},
		{9, 1},
		{11, 10},
		{3, 2},
		{4, 2},
		{5, 4},
	}
	pt := testProcTree()
	for _, match := range matches {
		c := match[0]
		p := pt.FindParent(c)
		if p == nil {
			t.Fatalf("nil parent for %d", c)
		}
		e := match[1]
		if p.Pid != e {
			t.Fatalf("found parent %d for %d but expected %d", p.Pid, c, e)
		}
	}
	c := uint32(math.MaxUint32)
	p := pt.FindParent(c)
	if p != nil {
		t.Fatalf("found parent %d for %d but expected no parent", p.Pid, c)
	}
}

func TestProcTree_Insert(t *testing.T) {
	// child, parent, branchof, 0=left 1=right
	matches := [4][4]uint32{
		{12, 5, 5, 0},
		{13, 12, 12, 0},
		{13, 1, 9, 1},
		{14, 7, 8, 1},
	}
	pt := testProcTree()
	for _, match := range matches {
		pid, ppid, expectedId, side := match[0], match[1], match[2], match[3]
		pt.Insert(pid, ppid)
		n := pt.Find(expectedId)
		if side == 0 {
			if n.Left == nil || n.Left.Pid != pid {
				t.Fatalf("failet to insert %d at %d (branch %s)", pid, ppid, n)
			}
		} else {
			if n.Right == nil || n.Right.Pid != pid {
				t.Fatalf("failet to insert %d at %d (branch %s)", pid, ppid, n)
			}
		}
	}

}

func TestProcTree_Pop(t *testing.T) {
	pt := testProcTree()
	parent := pt.FindParent(6)
	sibling := pt.Find(2)
	child := pt.Find(6)
	popped := pt.Pop(6)
	if popped.Pid != child.Pid {
		t.Fatalf("popped item with PID %d instead of %d", popped.Pid, child.Pid)
	}
	if sibling.Right.Pid != 7 {
		t.Fatalf("new right branch of %d should be 7, not %d", parent.Pid, sibling.Right.Pid)
	}
	// this test makes me dizzy
	parent = sibling.Right
	child = parent.Left
	if child.Pid != 8 {
		t.Fatalf("expected pid 8 got %d", child.Pid)
	}
	popped = pt.Pop(child.Pid)
	if popped.Pid != child.Pid {
		t.Fatalf("popped item with PID %d instead of %d", popped.Pid, child.Pid)
	}
	if parent.Left != nil {
		t.Fatalf("left branch of %d should be nil, not %d", sibling.Pid, sibling.Left.Pid)
	}
}
