// Code generated by capnpc-go. DO NOT EDIT.

package routing

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
)

type Record struct{ capnp.Struct }

// Record_TypeID is the unique identifier for the type Record.
const Record_TypeID = 0x82a35d1a82458a4a

func NewRecord(s *capnp.Segment) (Record, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1})
	return Record{st}, err
}

func NewRootRecord(s *capnp.Segment) (Record, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1})
	return Record{st}, err
}

func ReadRootRecord(msg *capnp.Message) (Record, error) {
	root, err := msg.Root()
	return Record{root.Struct()}, err
}

func (s Record) String() string {
	str, _ := text.Marshal(0x82a35d1a82458a4a, s.Struct)
	return str
}

func (s Record) Peer() (string, error) {
	p, err := s.Struct.Ptr(0)
	return p.Text(), err
}

func (s Record) HasPeer() bool {
	return s.Struct.HasPtr(0)
}

func (s Record) PeerBytes() ([]byte, error) {
	p, err := s.Struct.Ptr(0)
	return p.TextBytes(), err
}

func (s Record) SetPeer(v string) error {
	return s.Struct.SetText(0, v)
}

func (s Record) Ttl() int64 {
	return int64(s.Struct.Uint64(0))
}

func (s Record) SetTtl(v int64) {
	s.Struct.SetUint64(0, uint64(v))
}

func (s Record) Seq() uint64 {
	return s.Struct.Uint64(8)
}

func (s Record) SetSeq(v uint64) {
	s.Struct.SetUint64(8, v)
}

// Record_List is a list of Record.
type Record_List struct{ capnp.List }

// NewRecord creates a new list of Record.
func NewRecord_List(s *capnp.Segment, sz int32) (Record_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 16, PointerCount: 1}, sz)
	return Record_List{l}, err
}

func (s Record_List) At(i int) Record { return Record{s.List.Struct(i)} }

func (s Record_List) Set(i int, v Record) error { return s.List.SetStruct(i, v.Struct) }

func (s Record_List) String() string {
	str, _ := text.MarshalList(0x82a35d1a82458a4a, s.List)
	return str
}

// Record_Future is a wrapper for a Record promised by a client call.
type Record_Future struct{ *capnp.Future }

func (p Record_Future) Struct() (Record, error) {
	s, err := p.Future.Struct()
	return Record{s}, err
}

type Iteration struct{ capnp.Struct }

// Iteration_TypeID is the unique identifier for the type Iteration.
const Iteration_TypeID = 0xdc52a9a7339d80cd

func NewIteration(s *capnp.Segment) (Iteration, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Iteration{st}, err
}

func NewRootIteration(s *capnp.Segment) (Iteration, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Iteration{st}, err
}

func ReadRootIteration(msg *capnp.Message) (Iteration, error) {
	root, err := msg.Root()
	return Iteration{root.Struct()}, err
}

func (s Iteration) String() string {
	str, _ := text.Marshal(0xdc52a9a7339d80cd, s.Struct)
	return str
}

func (s Iteration) Record() (Record, error) {
	p, err := s.Struct.Ptr(0)
	return Record{Struct: p.Struct()}, err
}

func (s Iteration) HasRecord() bool {
	return s.Struct.HasPtr(0)
}

func (s Iteration) SetRecord(v Record) error {
	return s.Struct.SetPtr(0, v.Struct.ToPtr())
}

// NewRecord sets the record field to a newly
// allocated Record struct, preferring placement in s's segment.
func (s Iteration) NewRecord() (Record, error) {
	ss, err := NewRecord(s.Struct.Segment())
	if err != nil {
		return Record{}, err
	}
	err = s.Struct.SetPtr(0, ss.Struct.ToPtr())
	return ss, err
}

func (s Iteration) Dedadline() int64 {
	return int64(s.Struct.Uint64(0))
}

func (s Iteration) SetDedadline(v int64) {
	s.Struct.SetUint64(0, uint64(v))
}

// Iteration_List is a list of Iteration.
type Iteration_List struct{ capnp.List }

// NewIteration creates a new list of Iteration.
func NewIteration_List(s *capnp.Segment, sz int32) (Iteration_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return Iteration_List{l}, err
}

func (s Iteration_List) At(i int) Iteration { return Iteration{s.List.Struct(i)} }

func (s Iteration_List) Set(i int, v Iteration) error { return s.List.SetStruct(i, v.Struct) }

func (s Iteration_List) String() string {
	str, _ := text.MarshalList(0xdc52a9a7339d80cd, s.List)
	return str
}

// Iteration_Future is a wrapper for a Iteration promised by a client call.
type Iteration_Future struct{ *capnp.Future }

func (p Iteration_Future) Struct() (Iteration, error) {
	s, err := p.Future.Struct()
	return Iteration{s}, err
}

func (p Iteration_Future) Record() Record_Future {
	return Record_Future{Future: p.Future.Field(0, nil)}
}

type Routing struct{ Client *capnp.Client }

// Routing_TypeID is the unique identifier for the type Routing.
const Routing_TypeID = 0x9e7310daa8a8af1e

func (c Routing) Iter(ctx context.Context, params func(Routing_iter_Params) error) (Routing_iter_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0x9e7310daa8a8af1e,
			MethodID:      0,
			InterfaceName: "api/routing.capnp:Routing",
			MethodName:    "iter",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 8, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Routing_iter_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return Routing_iter_Results_Future{Future: ans.Future()}, release
}
func (c Routing) Lookup(ctx context.Context, params func(Routing_lookup_Params) error) (Routing_lookup_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0x9e7310daa8a8af1e,
			MethodID:      1,
			InterfaceName: "api/routing.capnp:Routing",
			MethodName:    "lookup",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Routing_lookup_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return Routing_lookup_Results_Future{Future: ans.Future()}, release
}

func (c Routing) AddRef() Routing {
	return Routing{
		Client: c.Client.AddRef(),
	}
}

func (c Routing) Release() {
	c.Client.Release()
}

// A Routing_Server is a Routing with a local implementation.
type Routing_Server interface {
	Iter(context.Context, Routing_iter) error

	Lookup(context.Context, Routing_lookup) error
}

// Routing_NewServer creates a new Server from an implementation of Routing_Server.
func Routing_NewServer(s Routing_Server, policy *server.Policy) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Routing_Methods(nil, s), s, c, policy)
}

// Routing_ServerToClient creates a new Client from an implementation of Routing_Server.
// The caller is responsible for calling Release on the returned Client.
func Routing_ServerToClient(s Routing_Server, policy *server.Policy) Routing {
	return Routing{Client: capnp.NewClient(Routing_NewServer(s, policy))}
}

// Routing_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Routing_Methods(methods []server.Method, s Routing_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 2)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0x9e7310daa8a8af1e,
			MethodID:      0,
			InterfaceName: "api/routing.capnp:Routing",
			MethodName:    "iter",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Iter(ctx, Routing_iter{call})
		},
	})

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0x9e7310daa8a8af1e,
			MethodID:      1,
			InterfaceName: "api/routing.capnp:Routing",
			MethodName:    "lookup",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Lookup(ctx, Routing_lookup{call})
		},
	})

	return methods
}

// Routing_iter holds the state for a server call to Routing.iter.
// See server.Call for documentation.
type Routing_iter struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Routing_iter) Args() Routing_iter_Params {
	return Routing_iter_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c Routing_iter) AllocResults() (Routing_iter_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_iter_Results{Struct: r}, err
}

// Routing_lookup holds the state for a server call to Routing.lookup.
// See server.Call for documentation.
type Routing_lookup struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Routing_lookup) Args() Routing_lookup_Params {
	return Routing_lookup_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c Routing_lookup) AllocResults() (Routing_lookup_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Routing_lookup_Results{Struct: r}, err
}

type Routing_Handler struct{ Client *capnp.Client }

// Routing_Handler_TypeID is the unique identifier for the type Routing_Handler.
const Routing_Handler_TypeID = 0xd221b2737a89d81e

func (c Routing_Handler) Handle(ctx context.Context, params func(Routing_Handler_handle_Params) error) (Routing_Handler_handle_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xd221b2737a89d81e,
			MethodID:      0,
			InterfaceName: "api/routing.capnp:Routing.Handler",
			MethodName:    "handle",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Routing_Handler_handle_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return Routing_Handler_handle_Results_Future{Future: ans.Future()}, release
}

func (c Routing_Handler) AddRef() Routing_Handler {
	return Routing_Handler{
		Client: c.Client.AddRef(),
	}
}

func (c Routing_Handler) Release() {
	c.Client.Release()
}

// A Routing_Handler_Server is a Routing_Handler with a local implementation.
type Routing_Handler_Server interface {
	Handle(context.Context, Routing_Handler_handle) error
}

// Routing_Handler_NewServer creates a new Server from an implementation of Routing_Handler_Server.
func Routing_Handler_NewServer(s Routing_Handler_Server, policy *server.Policy) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Routing_Handler_Methods(nil, s), s, c, policy)
}

// Routing_Handler_ServerToClient creates a new Client from an implementation of Routing_Handler_Server.
// The caller is responsible for calling Release on the returned Client.
func Routing_Handler_ServerToClient(s Routing_Handler_Server, policy *server.Policy) Routing_Handler {
	return Routing_Handler{Client: capnp.NewClient(Routing_Handler_NewServer(s, policy))}
}

// Routing_Handler_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Routing_Handler_Methods(methods []server.Method, s Routing_Handler_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xd221b2737a89d81e,
			MethodID:      0,
			InterfaceName: "api/routing.capnp:Routing.Handler",
			MethodName:    "handle",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Handle(ctx, Routing_Handler_handle{call})
		},
	})

	return methods
}

// Routing_Handler_handle holds the state for a server call to Routing_Handler.handle.
// See server.Call for documentation.
type Routing_Handler_handle struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Routing_Handler_handle) Args() Routing_Handler_handle_Params {
	return Routing_Handler_handle_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c Routing_Handler_handle) AllocResults() (Routing_Handler_handle_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_Handler_handle_Results{Struct: r}, err
}

type Routing_Handler_handle_Params struct{ capnp.Struct }

// Routing_Handler_handle_Params_TypeID is the unique identifier for the type Routing_Handler_handle_Params.
const Routing_Handler_handle_Params_TypeID = 0x80bfaaba61a06964

func NewRouting_Handler_handle_Params(s *capnp.Segment) (Routing_Handler_handle_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Routing_Handler_handle_Params{st}, err
}

func NewRootRouting_Handler_handle_Params(s *capnp.Segment) (Routing_Handler_handle_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Routing_Handler_handle_Params{st}, err
}

func ReadRootRouting_Handler_handle_Params(msg *capnp.Message) (Routing_Handler_handle_Params, error) {
	root, err := msg.Root()
	return Routing_Handler_handle_Params{root.Struct()}, err
}

func (s Routing_Handler_handle_Params) String() string {
	str, _ := text.Marshal(0x80bfaaba61a06964, s.Struct)
	return str
}

func (s Routing_Handler_handle_Params) Iterations() (Iteration_List, error) {
	p, err := s.Struct.Ptr(0)
	return Iteration_List{List: p.List()}, err
}

func (s Routing_Handler_handle_Params) HasIterations() bool {
	return s.Struct.HasPtr(0)
}

func (s Routing_Handler_handle_Params) SetIterations(v Iteration_List) error {
	return s.Struct.SetPtr(0, v.List.ToPtr())
}

// NewIterations sets the iterations field to a newly
// allocated Iteration_List, preferring placement in s's segment.
func (s Routing_Handler_handle_Params) NewIterations(n int32) (Iteration_List, error) {
	l, err := NewIteration_List(s.Struct.Segment(), n)
	if err != nil {
		return Iteration_List{}, err
	}
	err = s.Struct.SetPtr(0, l.List.ToPtr())
	return l, err
}

// Routing_Handler_handle_Params_List is a list of Routing_Handler_handle_Params.
type Routing_Handler_handle_Params_List struct{ capnp.List }

// NewRouting_Handler_handle_Params creates a new list of Routing_Handler_handle_Params.
func NewRouting_Handler_handle_Params_List(s *capnp.Segment, sz int32) (Routing_Handler_handle_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return Routing_Handler_handle_Params_List{l}, err
}

func (s Routing_Handler_handle_Params_List) At(i int) Routing_Handler_handle_Params {
	return Routing_Handler_handle_Params{s.List.Struct(i)}
}

func (s Routing_Handler_handle_Params_List) Set(i int, v Routing_Handler_handle_Params) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_Handler_handle_Params_List) String() string {
	str, _ := text.MarshalList(0x80bfaaba61a06964, s.List)
	return str
}

// Routing_Handler_handle_Params_Future is a wrapper for a Routing_Handler_handle_Params promised by a client call.
type Routing_Handler_handle_Params_Future struct{ *capnp.Future }

func (p Routing_Handler_handle_Params_Future) Struct() (Routing_Handler_handle_Params, error) {
	s, err := p.Future.Struct()
	return Routing_Handler_handle_Params{s}, err
}

type Routing_Handler_handle_Results struct{ capnp.Struct }

// Routing_Handler_handle_Results_TypeID is the unique identifier for the type Routing_Handler_handle_Results.
const Routing_Handler_handle_Results_TypeID = 0xf20e4198c79fd18d

func NewRouting_Handler_handle_Results(s *capnp.Segment) (Routing_Handler_handle_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_Handler_handle_Results{st}, err
}

func NewRootRouting_Handler_handle_Results(s *capnp.Segment) (Routing_Handler_handle_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_Handler_handle_Results{st}, err
}

func ReadRootRouting_Handler_handle_Results(msg *capnp.Message) (Routing_Handler_handle_Results, error) {
	root, err := msg.Root()
	return Routing_Handler_handle_Results{root.Struct()}, err
}

func (s Routing_Handler_handle_Results) String() string {
	str, _ := text.Marshal(0xf20e4198c79fd18d, s.Struct)
	return str
}

// Routing_Handler_handle_Results_List is a list of Routing_Handler_handle_Results.
type Routing_Handler_handle_Results_List struct{ capnp.List }

// NewRouting_Handler_handle_Results creates a new list of Routing_Handler_handle_Results.
func NewRouting_Handler_handle_Results_List(s *capnp.Segment, sz int32) (Routing_Handler_handle_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return Routing_Handler_handle_Results_List{l}, err
}

func (s Routing_Handler_handle_Results_List) At(i int) Routing_Handler_handle_Results {
	return Routing_Handler_handle_Results{s.List.Struct(i)}
}

func (s Routing_Handler_handle_Results_List) Set(i int, v Routing_Handler_handle_Results) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_Handler_handle_Results_List) String() string {
	str, _ := text.MarshalList(0xf20e4198c79fd18d, s.List)
	return str
}

// Routing_Handler_handle_Results_Future is a wrapper for a Routing_Handler_handle_Results promised by a client call.
type Routing_Handler_handle_Results_Future struct{ *capnp.Future }

func (p Routing_Handler_handle_Results_Future) Struct() (Routing_Handler_handle_Results, error) {
	s, err := p.Future.Struct()
	return Routing_Handler_handle_Results{s}, err
}

type Routing_iter_Params struct{ capnp.Struct }

// Routing_iter_Params_TypeID is the unique identifier for the type Routing_iter_Params.
const Routing_iter_Params_TypeID = 0xcaef48df88fdb195

func NewRouting_iter_Params(s *capnp.Segment) (Routing_iter_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Routing_iter_Params{st}, err
}

func NewRootRouting_iter_Params(s *capnp.Segment) (Routing_iter_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Routing_iter_Params{st}, err
}

func ReadRootRouting_iter_Params(msg *capnp.Message) (Routing_iter_Params, error) {
	root, err := msg.Root()
	return Routing_iter_Params{root.Struct()}, err
}

func (s Routing_iter_Params) String() string {
	str, _ := text.Marshal(0xcaef48df88fdb195, s.Struct)
	return str
}

func (s Routing_iter_Params) Handler() Routing_Handler {
	p, _ := s.Struct.Ptr(0)
	return Routing_Handler{Client: p.Interface().Client()}
}

func (s Routing_iter_Params) HasHandler() bool {
	return s.Struct.HasPtr(0)
}

func (s Routing_iter_Params) SetHandler(v Routing_Handler) error {
	if !v.Client.IsValid() {
		return s.Struct.SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().AddCap(v.Client))
	return s.Struct.SetPtr(0, in.ToPtr())
}

func (s Routing_iter_Params) BufSize() int32 {
	return int32(s.Struct.Uint32(0))
}

func (s Routing_iter_Params) SetBufSize(v int32) {
	s.Struct.SetUint32(0, uint32(v))
}

// Routing_iter_Params_List is a list of Routing_iter_Params.
type Routing_iter_Params_List struct{ capnp.List }

// NewRouting_iter_Params creates a new list of Routing_iter_Params.
func NewRouting_iter_Params_List(s *capnp.Segment, sz int32) (Routing_iter_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return Routing_iter_Params_List{l}, err
}

func (s Routing_iter_Params_List) At(i int) Routing_iter_Params {
	return Routing_iter_Params{s.List.Struct(i)}
}

func (s Routing_iter_Params_List) Set(i int, v Routing_iter_Params) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_iter_Params_List) String() string {
	str, _ := text.MarshalList(0xcaef48df88fdb195, s.List)
	return str
}

// Routing_iter_Params_Future is a wrapper for a Routing_iter_Params promised by a client call.
type Routing_iter_Params_Future struct{ *capnp.Future }

func (p Routing_iter_Params_Future) Struct() (Routing_iter_Params, error) {
	s, err := p.Future.Struct()
	return Routing_iter_Params{s}, err
}

func (p Routing_iter_Params_Future) Handler() Routing_Handler {
	return Routing_Handler{Client: p.Future.Field(0, nil).Client()}
}

type Routing_iter_Results struct{ capnp.Struct }

// Routing_iter_Results_TypeID is the unique identifier for the type Routing_iter_Results.
const Routing_iter_Results_TypeID = 0xd309de1ddf5872c8

func NewRouting_iter_Results(s *capnp.Segment) (Routing_iter_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_iter_Results{st}, err
}

func NewRootRouting_iter_Results(s *capnp.Segment) (Routing_iter_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Routing_iter_Results{st}, err
}

func ReadRootRouting_iter_Results(msg *capnp.Message) (Routing_iter_Results, error) {
	root, err := msg.Root()
	return Routing_iter_Results{root.Struct()}, err
}

func (s Routing_iter_Results) String() string {
	str, _ := text.Marshal(0xd309de1ddf5872c8, s.Struct)
	return str
}

// Routing_iter_Results_List is a list of Routing_iter_Results.
type Routing_iter_Results_List struct{ capnp.List }

// NewRouting_iter_Results creates a new list of Routing_iter_Results.
func NewRouting_iter_Results_List(s *capnp.Segment, sz int32) (Routing_iter_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return Routing_iter_Results_List{l}, err
}

func (s Routing_iter_Results_List) At(i int) Routing_iter_Results {
	return Routing_iter_Results{s.List.Struct(i)}
}

func (s Routing_iter_Results_List) Set(i int, v Routing_iter_Results) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_iter_Results_List) String() string {
	str, _ := text.MarshalList(0xd309de1ddf5872c8, s.List)
	return str
}

// Routing_iter_Results_Future is a wrapper for a Routing_iter_Results promised by a client call.
type Routing_iter_Results_Future struct{ *capnp.Future }

func (p Routing_iter_Results_Future) Struct() (Routing_iter_Results, error) {
	s, err := p.Future.Struct()
	return Routing_iter_Results{s}, err
}

type Routing_lookup_Params struct{ capnp.Struct }

// Routing_lookup_Params_TypeID is the unique identifier for the type Routing_lookup_Params.
const Routing_lookup_Params_TypeID = 0xd89c0a4fde72adb9

func NewRouting_lookup_Params(s *capnp.Segment) (Routing_lookup_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Routing_lookup_Params{st}, err
}

func NewRootRouting_lookup_Params(s *capnp.Segment) (Routing_lookup_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Routing_lookup_Params{st}, err
}

func ReadRootRouting_lookup_Params(msg *capnp.Message) (Routing_lookup_Params, error) {
	root, err := msg.Root()
	return Routing_lookup_Params{root.Struct()}, err
}

func (s Routing_lookup_Params) String() string {
	str, _ := text.Marshal(0xd89c0a4fde72adb9, s.Struct)
	return str
}

func (s Routing_lookup_Params) PeerID() (string, error) {
	p, err := s.Struct.Ptr(0)
	return p.Text(), err
}

func (s Routing_lookup_Params) HasPeerID() bool {
	return s.Struct.HasPtr(0)
}

func (s Routing_lookup_Params) PeerIDBytes() ([]byte, error) {
	p, err := s.Struct.Ptr(0)
	return p.TextBytes(), err
}

func (s Routing_lookup_Params) SetPeerID(v string) error {
	return s.Struct.SetText(0, v)
}

// Routing_lookup_Params_List is a list of Routing_lookup_Params.
type Routing_lookup_Params_List struct{ capnp.List }

// NewRouting_lookup_Params creates a new list of Routing_lookup_Params.
func NewRouting_lookup_Params_List(s *capnp.Segment, sz int32) (Routing_lookup_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return Routing_lookup_Params_List{l}, err
}

func (s Routing_lookup_Params_List) At(i int) Routing_lookup_Params {
	return Routing_lookup_Params{s.List.Struct(i)}
}

func (s Routing_lookup_Params_List) Set(i int, v Routing_lookup_Params) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_lookup_Params_List) String() string {
	str, _ := text.MarshalList(0xd89c0a4fde72adb9, s.List)
	return str
}

// Routing_lookup_Params_Future is a wrapper for a Routing_lookup_Params promised by a client call.
type Routing_lookup_Params_Future struct{ *capnp.Future }

func (p Routing_lookup_Params_Future) Struct() (Routing_lookup_Params, error) {
	s, err := p.Future.Struct()
	return Routing_lookup_Params{s}, err
}

type Routing_lookup_Results struct{ capnp.Struct }

// Routing_lookup_Results_TypeID is the unique identifier for the type Routing_lookup_Results.
const Routing_lookup_Results_TypeID = 0xa3ee2c22bdfd3b85

func NewRouting_lookup_Results(s *capnp.Segment) (Routing_lookup_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Routing_lookup_Results{st}, err
}

func NewRootRouting_lookup_Results(s *capnp.Segment) (Routing_lookup_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Routing_lookup_Results{st}, err
}

func ReadRootRouting_lookup_Results(msg *capnp.Message) (Routing_lookup_Results, error) {
	root, err := msg.Root()
	return Routing_lookup_Results{root.Struct()}, err
}

func (s Routing_lookup_Results) String() string {
	str, _ := text.Marshal(0xa3ee2c22bdfd3b85, s.Struct)
	return str
}

func (s Routing_lookup_Results) Record() (Record, error) {
	p, err := s.Struct.Ptr(0)
	return Record{Struct: p.Struct()}, err
}

func (s Routing_lookup_Results) HasRecord() bool {
	return s.Struct.HasPtr(0)
}

func (s Routing_lookup_Results) SetRecord(v Record) error {
	return s.Struct.SetPtr(0, v.Struct.ToPtr())
}

// NewRecord sets the record field to a newly
// allocated Record struct, preferring placement in s's segment.
func (s Routing_lookup_Results) NewRecord() (Record, error) {
	ss, err := NewRecord(s.Struct.Segment())
	if err != nil {
		return Record{}, err
	}
	err = s.Struct.SetPtr(0, ss.Struct.ToPtr())
	return ss, err
}

func (s Routing_lookup_Results) Ok() bool {
	return s.Struct.Bit(0)
}

func (s Routing_lookup_Results) SetOk(v bool) {
	s.Struct.SetBit(0, v)
}

// Routing_lookup_Results_List is a list of Routing_lookup_Results.
type Routing_lookup_Results_List struct{ capnp.List }

// NewRouting_lookup_Results creates a new list of Routing_lookup_Results.
func NewRouting_lookup_Results_List(s *capnp.Segment, sz int32) (Routing_lookup_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return Routing_lookup_Results_List{l}, err
}

func (s Routing_lookup_Results_List) At(i int) Routing_lookup_Results {
	return Routing_lookup_Results{s.List.Struct(i)}
}

func (s Routing_lookup_Results_List) Set(i int, v Routing_lookup_Results) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s Routing_lookup_Results_List) String() string {
	str, _ := text.MarshalList(0xa3ee2c22bdfd3b85, s.List)
	return str
}

// Routing_lookup_Results_Future is a wrapper for a Routing_lookup_Results promised by a client call.
type Routing_lookup_Results_Future struct{ *capnp.Future }

func (p Routing_lookup_Results_Future) Struct() (Routing_lookup_Results, error) {
	s, err := p.Future.Struct()
	return Routing_lookup_Results{s}, err
}

func (p Routing_lookup_Results_Future) Record() Record_Future {
	return Record_Future{Future: p.Future.Field(0, nil)}
}

const schema_fcf6ac08e448a6ac = "x\xda\x94\x94A\x88\x1bU\x18\xc7\xff\xff\xf7\xde8\xbb" +
	"\x92\xb6y;\x0bb\xc0\x8aK\x84Zj\xdd5\x1eJ" +
	"D\x12C\x17\x93b1/ZP\xa1\x87\xb1\x19\xeb\xd0" +
	"\x98\x89\x93\x04\xa1\xb0l]\x10m\xc1\x83\x87\x82\x82U" +
	"\xa1\xa0U+Eo\x0a\xe2Q)x\xb0\x1e,\xa2-" +
	"\x88\x1eU\xf0\xe0\xc92\xf2fL&\xb5\xdb\xaa\xa7L" +
	"^>\xf2\xfb\xbe\xdf\xf7\x7f\xb3\xfc\x01\xebb\xc5\xf9V" +
	"\x01f\x8fsS\xd2\x0d\xdf\xf6?}\xff\xf3c\xd0K" +
	"\x04\x1c\xba@\xc5\x88\x0d\x82\xdeAQ\x03\x93}'V" +
	"7J\x07Oo\xc0,P$g\xdfi\xfe4w\xf6" +
	"\x8f?\xb3J\xefUq\xde;%\xec\xd3\xeb\xe2\x1c\x98" +
	"l?w\xe6\xccw\xc5\xe1\x9b\xd0\x0b2\xaf\x05\xbd\x15" +
	"y\xc1{@\xde\x02x\xab\xf2%\xef\xa4t\x81\xe4\xc5" +
	"\xfb\xaf|\xb6\xb4\xeb\xd7\xd30%N\xd9kr\x9fe" +
	"\x1f\x97\xcf\x83\xc9\xc9\x8f\xae\xbc|\xb9\xf9\xdb\xf9\xab*" +
	"~\x96\xf7\xda\x8a_\xd2\x8a\xed\x17\x8f\x1f\x1d~|\xc7" +
	"\x05\xe8\x92\xcc\xe9`\xa5\xa5\x16\xe8=\xa1lk\x07\xd4" +
	"C\xde\x9a}J\xbe\x8c\x1f\xbf|\xdb\xa5\xf9o\xa0K" +
	"\x04\xecQ%PUB%\x9f|\x18_z\xe4\xe67" +
	".f\xbfd\xa0\xfd\xaaaA\x07\x94\xd5\xf0\xd5\xb1S" +
	"\x95w\xdf\xeb|o5\xf0\x9f\x1a\xc6\xeaG\xef\x85\x94" +
	"\xb5\xa6lW\xaf|\xfd\xd6\x17\xaf=\xb8\xf5\xf7Lj" +
	"\xca\xf9A\x9d \x96\x13\x7f\x10\xde\x13G\xe3\x91\x13\xf6" +
	"\x0f\xef>\xe4\x0f\xfa\x83j'\x1a\x8f\xec\xb7\xa6\xdf\xef" +
	"\xf6\x82x\xf73\xe9g\xb9\xed\xc7\xfe\xb3C\x18%\x15" +
	"\xa0\x08\xe8-O\x02\xa6 iv\x08&\xe1(\x88\xfd" +
	"Q\x18A\xf6\x87\xdc\x0a\xb6%Y\xcc\xbb\x04\xed\xe1\x14" +
	"'fp\xc1\xa1(\xee\x02m\xd2\x14\xa6\xff\xbd\xba\x13" +
	"0uI\xf3\xb0 \xb9hm\xeb\xd6\x12`\xf6J\x9a" +
	"\xb6\xa0\x16\\\xa4\x00\xf4~{\xd8\x944\x8f\x09n\x1b" +
	"\x04A\xcc\x02\x04\x0b\xa0;\x1a\xf5\xe8@\xd0\x01\xdda" +
	"\xf0\x1c\xe7!8\x7f\x9d\x1e\xd2\x91y\xd8(\xce\xae\x90" +
	"\x8d\xf5\xbf%\x989\xe9\x00\xd3\xfds\xb28\xbd\xb2\x13" +
	"B\xdf\xe9\x92\xd3\x8dq\x12#}k\x15Boq\xb7" +
	"Y5u\xd6zQtd<\xa8\xb3\xcd\xbc\x07u\xad" +
	"\xf6\xac\xac\xdc\x09\x86\xe3\xdeh\x08\x98\xb9\xa9\x94\xbb\xaa" +
	"\x80)K\x9a\xe5\\\xca\xdd%\xc0\xec\x904\xf7\x09\xd6" +
	"\xe2\xd4%\x8b\xf9%\x01Y\x04et\x84\x84 g\xc6" +
	"\x97\xd7\xa2m\xa3\xe5\xf6\xed\xe9\xa2g\xb1\x8dM\xb0\x8d" +
	"\x1c\xbb\x9e%$\xa6\xce\xdd\x81\xd4\xe0\xfaS\xe3\xa7\x1f" +
	"\x0d\x8f\x06T\x10T7\x86g\xa6\x19\xdb\x1c\xa8\xd4\xf6" +
	"\xe4U\xc0I|\xb5\xb6F\x1d\xb7\x96\x01\xafVy\xbd" +
	"y:\xb5\xcc\xe4\x7fp\xbeI\xc4\xab\xe9\x02h\x16\x05" +
	"k6]\xad\xbd\x93|m\x96\xa3Vz\x09\xdc0\xea" +
	"\xdb)\xfeeq\x1d\xc0\xec\x924{n\xb0\xb8\xa4\x1b" +
	"t\xfdn/\xec\x83\xc1$\xcc\xff\xe3\xceNC\xf4W" +
	"\x00\x00\x00\xff\xff!\x96x\xcd"

func init() {
	schemas.Register(schema_fcf6ac08e448a6ac,
		0x80bfaaba61a06964,
		0x82a35d1a82458a4a,
		0x9e7310daa8a8af1e,
		0xa3ee2c22bdfd3b85,
		0xcaef48df88fdb195,
		0xd221b2737a89d81e,
		0xd309de1ddf5872c8,
		0xd89c0a4fde72adb9,
		0xdc52a9a7339d80cd,
		0xf20e4198c79fd18d)
}
