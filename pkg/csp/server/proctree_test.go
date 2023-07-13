package server_test

import (
	"math"
	"testing"

	csp "github.com/wetware/ww/pkg/csp/server"
)

func testProcTree() csp.ProcTree {
	/*
		          0
		        /    \
				1     7
			   / \    /
		      2   3  8
			   \   \
				9   4
			   /   / \
			  10  5   6
	*/
	root := &csp.ProcNode{
		Pid: 0,
		Left: &csp.ProcNode{
			Pid: 1,
			Left: &csp.ProcNode{
				Pid: 2,
				Right: &csp.ProcNode{
					Pid: 9,
					Left: &csp.ProcNode{
						Pid: 10,
					},
				},
			},
			Right: &csp.ProcNode{
				Pid: 3,
				Right: &csp.ProcNode{
					Pid: 4,
					Left: &csp.ProcNode{
						Pid: 5,
					},
					Right: &csp.ProcNode{
						Pid: 6,
					},
				},
			},
		},
		Right: &csp.ProcNode{
			Pid: 7,
			Left: &csp.ProcNode{
				Pid: 8,
			},
		},
	}
	return csp.ProcTree{
		Root: root,
	}
}

func TestProcTree_Find(t *testing.T) {
	pt := testProcTree()
	for i := uint32(0); i <= 10; i++ {
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
	var matches = [6][2]uint32{
		{5, 4},
		{6, 0},
		{8, 7},
		{2, 1},
		{9, 1},
		{10, 9},
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
