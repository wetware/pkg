// Code generated by capnpc-go. DO NOT EDIT.

package cluster

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
	anchor "github.com/wetware/ww/internal/api/anchor"
	channel "github.com/wetware/ww/internal/api/channel"
)

type View struct{ Client capnp.Client }

// View_TypeID is the unique identifier for the type View.
const View_TypeID = 0x8a1df0335afc249a

func (c View) Iter(ctx context.Context, params func(View_iter_Params) error) (View_iter_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0x8a1df0335afc249a,
			MethodID:      0,
			InterfaceName: "cluster.capnp:View",
			MethodName:    "iter",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(View_iter_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return View_iter_Results_Future{Future: ans.Future()}, release
}
func (c View) Lookup(ctx context.Context, params func(View_lookup_Params) error) (View_lookup_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0x8a1df0335afc249a,
			MethodID:      1,
			InterfaceName: "cluster.capnp:View",
			MethodName:    "lookup",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(View_lookup_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return View_lookup_Results_Future{Future: ans.Future()}, release
}

func (c View) AddRef() View {
	return View{
		Client: c.Client.AddRef(),
	}
}

func (c View) Release() {
	c.Client.Release()
}

// A View_Server is a View with a local implementation.
type View_Server interface {
	Iter(context.Context, View_iter) error

	Lookup(context.Context, View_lookup) error
}

// View_NewServer creates a new Server from an implementation of View_Server.
func View_NewServer(s View_Server, policy *server.Policy) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(View_Methods(nil, s), s, c, policy)
}

// View_ServerToClient creates a new Client from an implementation of View_Server.
// The caller is responsible for calling Release on the returned Client.
func View_ServerToClient(s View_Server, policy *server.Policy) View {
	return View{Client: capnp.NewClient(View_NewServer(s, policy))}
}

// View_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func View_Methods(methods []server.Method, s View_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 2)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0x8a1df0335afc249a,
			MethodID:      0,
			InterfaceName: "cluster.capnp:View",
			MethodName:    "iter",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Iter(ctx, View_iter{call})
		},
	})

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0x8a1df0335afc249a,
			MethodID:      1,
			InterfaceName: "cluster.capnp:View",
			MethodName:    "lookup",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Lookup(ctx, View_lookup{call})
		},
	})

	return methods
}

// View_iter holds the state for a server call to View.iter.
// See server.Call for documentation.
type View_iter struct {
	*server.Call
}

// Args returns the call's arguments.
func (c View_iter) Args() View_iter_Params {
	return View_iter_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c View_iter) AllocResults() (View_iter_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return View_iter_Results{Struct: r}, err
}

// View_lookup holds the state for a server call to View.lookup.
// See server.Call for documentation.
type View_lookup struct {
	*server.Call
}

// Args returns the call's arguments.
func (c View_lookup) Args() View_lookup_Params {
	return View_lookup_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c View_lookup) AllocResults() (View_lookup_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return View_lookup_Results{Struct: r}, err
}

// View_List is a list of View.
type View_List = capnp.CapList[View]

// NewView creates a new list of View.
func NewView_List(s *capnp.Segment, sz int32) (View_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[View](l), err
}

type View_Record struct{ capnp.Struct }

// View_Record_TypeID is the unique identifier for the type View_Record.
const View_Record_TypeID = 0xcdcf42beb2537d20

func NewView_Record(s *capnp.Segment) (View_Record, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1})
	return View_Record{st}, err
}

func NewRootView_Record(s *capnp.Segment) (View_Record, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1})
	return View_Record{st}, err
}

func ReadRootView_Record(msg *capnp.Message) (View_Record, error) {
	root, err := msg.Root()
	return View_Record{root.Struct()}, err
}

func (s View_Record) String() string {
	str, _ := text.Marshal(0xcdcf42beb2537d20, s.Struct)
	return str
}

func (s View_Record) Peer() (string, error) {
	p, err := s.Struct.Ptr(0)
	return p.Text(), err
}

func (s View_Record) HasPeer() bool {
	return s.Struct.HasPtr(0)
}

func (s View_Record) PeerBytes() ([]byte, error) {
	p, err := s.Struct.Ptr(0)
	return p.TextBytes(), err
}

func (s View_Record) SetPeer(v string) error {
	return s.Struct.SetText(0, v)
}

func (s View_Record) Ttl() int64 {
	return int64(s.Struct.Uint64(0))
}

func (s View_Record) SetTtl(v int64) {
	s.Struct.SetUint64(0, uint64(v))
}

func (s View_Record) Seq() uint64 {
	return s.Struct.Uint64(8)
}

func (s View_Record) SetSeq(v uint64) {
	s.Struct.SetUint64(8, v)
}

// View_Record_List is a list of View_Record.
type View_Record_List = capnp.StructList[View_Record]

// NewView_Record creates a new list of View_Record.
func NewView_Record_List(s *capnp.Segment, sz int32) (View_Record_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1}, sz)
	return capnp.StructList[View_Record]{List: l}, err
}

// View_Record_Future is a wrapper for a View_Record promised by a client call.
type View_Record_Future struct{ *capnp.Future }

func (p View_Record_Future) Struct() (View_Record, error) {
	s, err := p.Future.Struct()
	return View_Record{s}, err
}

type View_iter_Params struct{ capnp.Struct }

// View_iter_Params_TypeID is the unique identifier for the type View_iter_Params.
const View_iter_Params_TypeID = 0xd929e054f82b286c

func NewView_iter_Params(s *capnp.Segment) (View_iter_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return View_iter_Params{st}, err
}

func NewRootView_iter_Params(s *capnp.Segment) (View_iter_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return View_iter_Params{st}, err
}

func ReadRootView_iter_Params(msg *capnp.Message) (View_iter_Params, error) {
	root, err := msg.Root()
	return View_iter_Params{root.Struct()}, err
}

func (s View_iter_Params) String() string {
	str, _ := text.Marshal(0xd929e054f82b286c, s.Struct)
	return str
}

func (s View_iter_Params) Handler() channel.Sender {
	p, _ := s.Struct.Ptr(0)
	return channel.Sender{Client: p.Interface().Client()}
}

func (s View_iter_Params) HasHandler() bool {
	return s.Struct.HasPtr(0)
}

func (s View_iter_Params) SetHandler(v channel.Sender) error {
	if !v.Client.IsValid() {
		return s.Struct.SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().AddCap(v.Client))
	return s.Struct.SetPtr(0, in.ToPtr())
}

// View_iter_Params_List is a list of View_iter_Params.
type View_iter_Params_List = capnp.StructList[View_iter_Params]

// NewView_iter_Params creates a new list of View_iter_Params.
func NewView_iter_Params_List(s *capnp.Segment, sz int32) (View_iter_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[View_iter_Params]{List: l}, err
}

// View_iter_Params_Future is a wrapper for a View_iter_Params promised by a client call.
type View_iter_Params_Future struct{ *capnp.Future }

func (p View_iter_Params_Future) Struct() (View_iter_Params, error) {
	s, err := p.Future.Struct()
	return View_iter_Params{s}, err
}

func (p View_iter_Params_Future) Handler() channel.Sender {
	return channel.Sender{Client: p.Future.Field(0, nil).Client()}
}

type View_iter_Results struct{ capnp.Struct }

// View_iter_Results_TypeID is the unique identifier for the type View_iter_Results.
const View_iter_Results_TypeID = 0xe6df611247a8fc13

func NewView_iter_Results(s *capnp.Segment) (View_iter_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return View_iter_Results{st}, err
}

func NewRootView_iter_Results(s *capnp.Segment) (View_iter_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return View_iter_Results{st}, err
}

func ReadRootView_iter_Results(msg *capnp.Message) (View_iter_Results, error) {
	root, err := msg.Root()
	return View_iter_Results{root.Struct()}, err
}

func (s View_iter_Results) String() string {
	str, _ := text.Marshal(0xe6df611247a8fc13, s.Struct)
	return str
}

// View_iter_Results_List is a list of View_iter_Results.
type View_iter_Results_List = capnp.StructList[View_iter_Results]

// NewView_iter_Results creates a new list of View_iter_Results.
func NewView_iter_Results_List(s *capnp.Segment, sz int32) (View_iter_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[View_iter_Results]{List: l}, err
}

// View_iter_Results_Future is a wrapper for a View_iter_Results promised by a client call.
type View_iter_Results_Future struct{ *capnp.Future }

func (p View_iter_Results_Future) Struct() (View_iter_Results, error) {
	s, err := p.Future.Struct()
	return View_iter_Results{s}, err
}

type View_lookup_Params struct{ capnp.Struct }

// View_lookup_Params_TypeID is the unique identifier for the type View_lookup_Params.
const View_lookup_Params_TypeID = 0xf495a555c9344000

func NewView_lookup_Params(s *capnp.Segment) (View_lookup_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return View_lookup_Params{st}, err
}

func NewRootView_lookup_Params(s *capnp.Segment) (View_lookup_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return View_lookup_Params{st}, err
}

func ReadRootView_lookup_Params(msg *capnp.Message) (View_lookup_Params, error) {
	root, err := msg.Root()
	return View_lookup_Params{root.Struct()}, err
}

func (s View_lookup_Params) String() string {
	str, _ := text.Marshal(0xf495a555c9344000, s.Struct)
	return str
}

func (s View_lookup_Params) PeerID() (string, error) {
	p, err := s.Struct.Ptr(0)
	return p.Text(), err
}

func (s View_lookup_Params) HasPeerID() bool {
	return s.Struct.HasPtr(0)
}

func (s View_lookup_Params) PeerIDBytes() ([]byte, error) {
	p, err := s.Struct.Ptr(0)
	return p.TextBytes(), err
}

func (s View_lookup_Params) SetPeerID(v string) error {
	return s.Struct.SetText(0, v)
}

// View_lookup_Params_List is a list of View_lookup_Params.
type View_lookup_Params_List = capnp.StructList[View_lookup_Params]

// NewView_lookup_Params creates a new list of View_lookup_Params.
func NewView_lookup_Params_List(s *capnp.Segment, sz int32) (View_lookup_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[View_lookup_Params]{List: l}, err
}

// View_lookup_Params_Future is a wrapper for a View_lookup_Params promised by a client call.
type View_lookup_Params_Future struct{ *capnp.Future }

func (p View_lookup_Params_Future) Struct() (View_lookup_Params, error) {
	s, err := p.Future.Struct()
	return View_lookup_Params{s}, err
}

type View_lookup_Results struct{ capnp.Struct }

// View_lookup_Results_TypeID is the unique identifier for the type View_lookup_Results.
const View_lookup_Results_TypeID = 0xe54acc44b61fd7ef

func NewView_lookup_Results(s *capnp.Segment) (View_lookup_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return View_lookup_Results{st}, err
}

func NewRootView_lookup_Results(s *capnp.Segment) (View_lookup_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return View_lookup_Results{st}, err
}

func ReadRootView_lookup_Results(msg *capnp.Message) (View_lookup_Results, error) {
	root, err := msg.Root()
	return View_lookup_Results{root.Struct()}, err
}

func (s View_lookup_Results) String() string {
	str, _ := text.Marshal(0xe54acc44b61fd7ef, s.Struct)
	return str
}

func (s View_lookup_Results) Record() (View_Record, error) {
	p, err := s.Struct.Ptr(0)
	return View_Record{Struct: p.Struct()}, err
}

func (s View_lookup_Results) HasRecord() bool {
	return s.Struct.HasPtr(0)
}

func (s View_lookup_Results) SetRecord(v View_Record) error {
	return s.Struct.SetPtr(0, v.Struct.ToPtr())
}

// NewRecord sets the record field to a newly
// allocated View_Record struct, preferring placement in s's segment.
func (s View_lookup_Results) NewRecord() (View_Record, error) {
	ss, err := NewView_Record(s.Struct.Segment())
	if err != nil {
		return View_Record{}, err
	}
	err = s.Struct.SetPtr(0, ss.Struct.ToPtr())
	return ss, err
}

func (s View_lookup_Results) Ok() bool {
	return s.Struct.Bit(0)
}

func (s View_lookup_Results) SetOk(v bool) {
	s.Struct.SetBit(0, v)
}

// View_lookup_Results_List is a list of View_lookup_Results.
type View_lookup_Results_List = capnp.StructList[View_lookup_Results]

// NewView_lookup_Results creates a new list of View_lookup_Results.
func NewView_lookup_Results_List(s *capnp.Segment, sz int32) (View_lookup_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return capnp.StructList[View_lookup_Results]{List: l}, err
}

// View_lookup_Results_Future is a wrapper for a View_lookup_Results promised by a client call.
type View_lookup_Results_Future struct{ *capnp.Future }

func (p View_lookup_Results_Future) Struct() (View_lookup_Results, error) {
	s, err := p.Future.Struct()
	return View_lookup_Results{s}, err
}

func (p View_lookup_Results_Future) Record() View_Record_Future {
	return View_Record_Future{Future: p.Future.Field(0, nil)}
}

type Host struct{ Client capnp.Client }

// Host_TypeID is the unique identifier for the type Host.
const Host_TypeID = 0x957cbefc645fd307

func (c Host) View(ctx context.Context, params func(Host_view_Params) error) (Host_view_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0x957cbefc645fd307,
			MethodID:      0,
			InterfaceName: "cluster.capnp:Host",
			MethodName:    "view",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 0}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Host_view_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return Host_view_Results_Future{Future: ans.Future()}, release
}
func (c Host) Ls(ctx context.Context, params func(anchor.Anchor_ls_Params) error) (anchor.Anchor_ls_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xe41237e4098ed922,
			MethodID:      0,
			InterfaceName: "anchor.capnp:Anchor",
			MethodName:    "ls",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 0}
		s.PlaceArgs = func(s capnp.Struct) error { return params(anchor.Anchor_ls_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return anchor.Anchor_ls_Results_Future{Future: ans.Future()}, release
}
func (c Host) Walk(ctx context.Context, params func(anchor.Anchor_walk_Params) error) (anchor.Anchor_walk_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xe41237e4098ed922,
			MethodID:      1,
			InterfaceName: "anchor.capnp:Anchor",
			MethodName:    "walk",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(anchor.Anchor_walk_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return anchor.Anchor_walk_Results_Future{Future: ans.Future()}, release
}

func (c Host) AddRef() Host {
	return Host{
		Client: c.Client.AddRef(),
	}
}

func (c Host) Release() {
	c.Client.Release()
}

// A Host_Server is a Host with a local implementation.
type Host_Server interface {
	View(context.Context, Host_view) error

	Ls(context.Context, anchor.Anchor_ls) error

	Walk(context.Context, anchor.Anchor_walk) error
}

// Host_NewServer creates a new Server from an implementation of Host_Server.
func Host_NewServer(s Host_Server, policy *server.Policy) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Host_Methods(nil, s), s, c, policy)
}

// Host_ServerToClient creates a new Client from an implementation of Host_Server.
// The caller is responsible for calling Release on the returned Client.
func Host_ServerToClient(s Host_Server, policy *server.Policy) Host {
	return Host{Client: capnp.NewClient(Host_NewServer(s, policy))}
}

// Host_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Host_Methods(methods []server.Method, s Host_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 3)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0x957cbefc645fd307,
			MethodID:      0,
			InterfaceName: "cluster.capnp:Host",
			MethodName:    "view",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.View(ctx, Host_view{call})
		},
	})

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xe41237e4098ed922,
			MethodID:      0,
			InterfaceName: "anchor.capnp:Anchor",
			MethodName:    "ls",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Ls(ctx, anchor.Anchor_ls{call})
		},
	})

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xe41237e4098ed922,
			MethodID:      1,
			InterfaceName: "anchor.capnp:Anchor",
			MethodName:    "walk",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Walk(ctx, anchor.Anchor_walk{call})
		},
	})

	return methods
}

// Host_view holds the state for a server call to Host.view.
// See server.Call for documentation.
type Host_view struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Host_view) Args() Host_view_Params {
	return Host_view_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c Host_view) AllocResults() (Host_view_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Host_view_Results{Struct: r}, err
}

// Host_List is a list of Host.
type Host_List = capnp.CapList[Host]

// NewHost creates a new list of Host.
func NewHost_List(s *capnp.Segment, sz int32) (Host_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Host](l), err
}

type Host_view_Params struct{ capnp.Struct }

// Host_view_Params_TypeID is the unique identifier for the type Host_view_Params.
const Host_view_Params_TypeID = 0xa404c24b5375b9e4

func NewHost_view_Params(s *capnp.Segment) (Host_view_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Host_view_Params{st}, err
}

func NewRootHost_view_Params(s *capnp.Segment) (Host_view_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Host_view_Params{st}, err
}

func ReadRootHost_view_Params(msg *capnp.Message) (Host_view_Params, error) {
	root, err := msg.Root()
	return Host_view_Params{root.Struct()}, err
}

func (s Host_view_Params) String() string {
	str, _ := text.Marshal(0xa404c24b5375b9e4, s.Struct)
	return str
}

// Host_view_Params_List is a list of Host_view_Params.
type Host_view_Params_List = capnp.StructList[Host_view_Params]

// NewHost_view_Params creates a new list of Host_view_Params.
func NewHost_view_Params_List(s *capnp.Segment, sz int32) (Host_view_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[Host_view_Params]{List: l}, err
}

// Host_view_Params_Future is a wrapper for a Host_view_Params promised by a client call.
type Host_view_Params_Future struct{ *capnp.Future }

func (p Host_view_Params_Future) Struct() (Host_view_Params, error) {
	s, err := p.Future.Struct()
	return Host_view_Params{s}, err
}

type Host_view_Results struct{ capnp.Struct }

// Host_view_Results_TypeID is the unique identifier for the type Host_view_Results.
const Host_view_Results_TypeID = 0x8f58928e854cd4f5

func NewHost_view_Results(s *capnp.Segment) (Host_view_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Host_view_Results{st}, err
}

func NewRootHost_view_Results(s *capnp.Segment) (Host_view_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Host_view_Results{st}, err
}

func ReadRootHost_view_Results(msg *capnp.Message) (Host_view_Results, error) {
	root, err := msg.Root()
	return Host_view_Results{root.Struct()}, err
}

func (s Host_view_Results) String() string {
	str, _ := text.Marshal(0x8f58928e854cd4f5, s.Struct)
	return str
}

func (s Host_view_Results) View() View {
	p, _ := s.Struct.Ptr(0)
	return View{Client: p.Interface().Client()}
}

func (s Host_view_Results) HasView() bool {
	return s.Struct.HasPtr(0)
}

func (s Host_view_Results) SetView(v View) error {
	if !v.Client.IsValid() {
		return s.Struct.SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().AddCap(v.Client))
	return s.Struct.SetPtr(0, in.ToPtr())
}

// Host_view_Results_List is a list of Host_view_Results.
type Host_view_Results_List = capnp.StructList[Host_view_Results]

// NewHost_view_Results creates a new list of Host_view_Results.
func NewHost_view_Results_List(s *capnp.Segment, sz int32) (Host_view_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Host_view_Results]{List: l}, err
}

// Host_view_Results_Future is a wrapper for a Host_view_Results promised by a client call.
type Host_view_Results_Future struct{ *capnp.Future }

func (p Host_view_Results_Future) Struct() (Host_view_Results, error) {
	s, err := p.Future.Struct()
	return Host_view_Results{s}, err
}

func (p Host_view_Results_Future) View() View {
	return View{Client: p.Future.Field(0, nil).Client()}
}

const schema_fcf6ac08e448a6ac = "x\xdad\x92OhSY\x18\xc5\xbfs\xdf\xbdy\xc9" +
	"\xd0L{{3Sf`\x98\x99\xd2B\xdbaJK" +
	"\x0b3d\x93P*M\xb5bn[\x8b\xb8\x91\xd0<" +
	"0461/1\x1b\xc5\x95\x08\x82\x15\x84.,t" +
	"\xa7\xe2\xa2 \xb8\x10D\xecB\x10\x11\x14\xc1?HW" +
	"\xba\x88\"B\xd5\x85\x8a-\xd1'/\xf1\xe5\xc5t\xfb" +
	"\xb8\xef\x9c\xdf9\xdf\x19\xdaF\x9c\x0f\x87\xf7\x85\x88\xe9" +
	"\xac\x088+=\xd5\x83#\xef\xff8C\xf2g\xc3Y" +
	"\xbb\x9c\xa8\x04\xd7>U\x89\xa0\xce\xf2\x15\xb5\xcc\xbb\x88" +
	"\xd4*?\xad6\xb9I\xe4||2uj\xe9\xfc\x81" +
	"s$\x15\x88\x04L\xa2\x91\xa7\x9c\x81\xa06x\x8c\xe0" +
	"\x98\x8f\x0f\xa5\xab\xeb\xc7\x97w\xa8m\xf1\x15\x05a\x12" +
	"\xa9/|B\xf5\x8a.\"\xa7r\xa34\xb3\xe76\xbf" +
	"XWs\x0d\xd4/b\x9b\xb8\xf3\xd7\x89\x99k\xebc" +
	"\x0f\x1f\x90V`>a\xcdOm\xf1GJ\xd4\x94 " +
	"\xae\x12\x9cl\xdf?\x9fg_\xf4o41\xa9\x0bb" +
	"\x9b\xa0V\x85\x8b\xf4\xee\xd9\x9f\xd7\xc7\xef\xef~\xe9\x8a" +
	"5\xa0o\x89N\x17\xfa\x8e(\x13\x1cU\xbd2\xd1\x99" +
	"z\xfe\xca\xe7\x18\xf9;\xc0@\xfck|\xf4\xde\xfeK" +
	"\xcb\x1f\x9a\xd2\x8a\xc0O\xee\x8f\xe1@\x8c\xee:\xf3\xd9" +
	"\x92]\xb4\x0a\x83\x98O\xe5\x17\xf3\xd1\xb9\x8ca\x955" +
	"\x07\xfc\x00\x12\xd1\xd8\xb45\x9f+\xa4u\xd0\x10D\x0d" +
	"Zx\xa6rx\x80\x98\xec5\x01\xcf\xcd'\x96\xbfE" +
	"\x89\xc9\xb0\xd9\x9e)Z\x858b\xd9\\n\xa1\x94\x8f" +
	"#\x094\xbc\x8d\xbaw\"g\x17\x07\x8fe\xacr\xcf" +
	"\xb4e\x97\xb2E\x9b478\x11\x07\x91\x0c\x0f\x10\xe9" +
	"\xa0\x01\x1dahw\x1fA\xfa\xad\x12 \x09\xadY\x12" +
	"9\xc3.&\x01\xcdk\xd8\xde\xa9\xe0-@J\x17[" +
	"\x985\xb98j\xa9\xbb7\x96B\x95\xff:+D\xd4" +
	"\x90c\xadx\xb1d\xaa\x90:b\xb7>\x98\xcbX\xe5" +
	"\xc1ZSH\xbb\xb6m\x0d\xf8].|\xdc\x80\x9eb" +
	"\x00\"\xee\x0d\xe5d7\x91\x1e7\xa0\x93\x0c\x92!\x02" +
	"F$\xf7\xba\x1f\x13\x06\xf4,C{\xde\xb2\x0ah#" +
	"\x866\x82Y,f!\x88A\x10L\xdb:\x8a\x101" +
	"\x84\x9a273\xb8U{\x90\xcd\x15\x8e\xf9\x15\x9e<" +
	"\x9cZLg\xad\x02\xa4\xf3\xf6\xd77\xffG6o\xbe" +
	"nm\xd1hR\xac_\xed\xfb]`\xeb`C\xb4?" +
	"J\xa4{\x0c\xe8!?\xda\xbf\xbf\x13\xe9>\x03z\x94" +
	"!V\xa8M\x07\x1d\xfe\xa0\x08\xe8 \x18\xb9\x05\x80\x18" +
	"\xd0di\xb6\x86\xf0\x86\xe0=\xd8\x89TOI?L" +
	"%\xea\xe7\x8c\xb9%N\x8e{5~\x0b\x00\x00\xff\xff" +
	"\xban&A"

func init() {
	schemas.Register(schema_fcf6ac08e448a6ac,
		0x8a1df0335afc249a,
		0x8f58928e854cd4f5,
		0x957cbefc645fd307,
		0xa404c24b5375b9e4,
		0xcdcf42beb2537d20,
		0xd929e054f82b286c,
		0xe54acc44b61fd7ef,
		0xe6df611247a8fc13,
		0xf495a555c9344000)
}
