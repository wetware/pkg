package anchor

// type Node struct {
// 	Path   Path
// 	Client *capnp.Client
// }

// // String returns the node's name, which is the last segment in its path,
// // stripped of its leading separator.
// func (n Node) String() string {
// 	return n.Path.bind(last).String()
// }

// func (n Node) Bind(a NamedAnchorSetter) (err error) {
// 	if err = a.SetName(n.String()); err == nil {
// 		err = a.SetAnchor(n.Anchor())
// 	}

// 	return
// }

// func (n Node) Anchor() cluster.Anchor {
// 	return cluster.Anchor{Client: n.Client}.AddRef()
// }
