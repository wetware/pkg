package stm

import "errors"

type Node struct {
	Name     string
	Children []*Node
}

// // NewRootPath allocates a new root path.  Users should generally
// // prefer Path(), which takes a root parameter and reduces allocs.
// func NewRootPath(path []string) (root *Node) {
// 	if len(path) > 0 {
// 		root = new(Node)
// 		root.Name, path = path[0], path[1:]

// 		for _, name := range path {
// 			root.Children = append(root.Children, &Node{Name: name})
// 		}
// 	}

// 	return
// }

type PathIterator interface {
	Len() int
	At(int) (string, error)
}

type StringSliceIterator []string

func (ss StringSliceIterator) Len() int { return len(ss) }

func (ss StringSliceIterator) At(i int) (string, error) {
	if i < 0 || i >= len(ss) {
		return "", errors.New("out of bounds")
	}

	return ss[i], nil
}

type NodeIterator struct {
	Current *Node
	Path    PathIterator
	index   int
}

/*

	TODO:  YOU ARE HERE:  create 'Walk' using 'Visitor' => should create missing children

*/

// func Walk(iter NodeIterator) (u *Node, err error) {
// 	visit := Visitor(func(n *Node) (stop bool) {
// 		if n == nil {
// 			// ...
// 		}

// 	})

// 	err = visit(iter)
// 	return
// }

func Visitor(visit func(*Node) (stop bool)) func(NodeIterator) error {
	return func(iter NodeIterator) (err error) {
		var stop bool

		for !stop && err == nil && iter.Current != nil {
			stop = visit(iter.Current)
			iter, err = Next(iter)
		}

		return
	}
}

func Next(iter NodeIterator) (NodeIterator, error) {
	if exhausted(iter) {
		return NodeIterator{}, nil
	}

	name, err := iter.Path.At(iter.index)
	if err != nil {
		return NodeIterator{}, err
	}

	return NodeIterator{
		Current: Child(iter.Current, name),
		Path:    iter.Path,
		index:   iter.index + 1,
	}, nil
}

func Child(parent *Node, name string) (child *Node) {
	if child = Find(parent, name); child == nil {
		child = &Node{Name: name}
	}

	return
}

func Find(n *Node, name string) (u *Node) {
	for _, u = range children(n) {
		if n.Name == name {
			break
		}
	}

	return
}

func children(n *Node) []*Node {
	if n == nil {
		return nil
	}

	return n.Children
}

func exhausted(iter NodeIterator) bool {
	return iter.Current == nil || iter.Path.Len() >= iter.index
}
