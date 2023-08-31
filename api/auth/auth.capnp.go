// Code generated by capnpc-go. DO NOT EDIT.

package auth

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	fc "capnproto.org/go/capnp/v3/flowcontrol"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
	anchor "github.com/wetware/pkg/api/anchor"
	cluster "github.com/wetware/pkg/api/cluster"
	pubsub "github.com/wetware/pkg/api/pubsub"
	strconv "strconv"
)

type Session capnp.Struct

// Session_TypeID is the unique identifier for the type Session.
const Session_TypeID = 0xc7aa1c890147e28a

func NewSession(s *capnp.Segment) (Session, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 3})
	return Session(st), err
}

func NewRootSession(s *capnp.Segment) (Session, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 3})
	return Session(st), err
}

func ReadRootSession(msg *capnp.Message) (Session, error) {
	root, err := msg.Root()
	return Session(root.Struct()), err
}

func (s Session) String() string {
	str, _ := text.Marshal(0xc7aa1c890147e28a, capnp.Struct(s))
	return str
}

func (s Session) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Session) DecodeFromPtr(p capnp.Ptr) Session {
	return Session(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Session) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Session) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Session) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Session) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Session) View() cluster.View {
	p, _ := capnp.Struct(s).Ptr(0)
	return cluster.View(p.Interface().Client())
}

func (s Session) HasView() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Session) SetView(v cluster.View) error {
	if !v.IsValid() {
		return capnp.Struct(s).SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().CapTable().Add(capnp.Client(v)))
	return capnp.Struct(s).SetPtr(0, in.ToPtr())
}

func (s Session) Root() anchor.Anchor {
	p, _ := capnp.Struct(s).Ptr(1)
	return anchor.Anchor(p.Interface().Client())
}

func (s Session) HasRoot() bool {
	return capnp.Struct(s).HasPtr(1)
}

func (s Session) SetRoot(v anchor.Anchor) error {
	if !v.IsValid() {
		return capnp.Struct(s).SetPtr(1, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().CapTable().Add(capnp.Client(v)))
	return capnp.Struct(s).SetPtr(1, in.ToPtr())
}

func (s Session) Pubsub() pubsub.Router {
	p, _ := capnp.Struct(s).Ptr(2)
	return pubsub.Router(p.Interface().Client())
}

func (s Session) HasPubsub() bool {
	return capnp.Struct(s).HasPtr(2)
}

func (s Session) SetPubsub(v pubsub.Router) error {
	if !v.IsValid() {
		return capnp.Struct(s).SetPtr(2, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().CapTable().Add(capnp.Client(v)))
	return capnp.Struct(s).SetPtr(2, in.ToPtr())
}

// Session_List is a list of Session.
type Session_List = capnp.StructList[Session]

// NewSession creates a new list of Session.
func NewSession_List(s *capnp.Segment, sz int32) (Session_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 3}, sz)
	return capnp.StructList[Session](l), err
}

// Session_Future is a wrapper for a Session promised by a client call.
type Session_Future struct{ *capnp.Future }

func (f Session_Future) Struct() (Session, error) {
	p, err := f.Future.Ptr()
	return Session(p.Struct()), err
}
func (p Session_Future) View() cluster.View {
	return cluster.View(p.Future.Field(0, nil).Client())
}

func (p Session_Future) Root() anchor.Anchor {
	return anchor.Anchor(p.Future.Field(1, nil).Client())
}

func (p Session_Future) Pubsub() pubsub.Router {
	return pubsub.Router(p.Future.Field(2, nil).Client())
}

type Terminal capnp.Client

// Terminal_TypeID is the unique identifier for the type Terminal.
const Terminal_TypeID = 0xe9885abc9e5f9481

func (c Terminal) Login(ctx context.Context, params func(Terminal_login_Params) error) (Terminal_login_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xe9885abc9e5f9481,
			MethodID:      0,
			InterfaceName: "auth.capnp:Terminal",
			MethodName:    "login",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Terminal_login_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Terminal_login_Results_Future{Future: ans.Future()}, release

}

func (c Terminal) WaitStreaming() error {
	return capnp.Client(c).WaitStreaming()
}

// String returns a string that identifies this capability for debugging
// purposes.  Its format should not be depended on: in particular, it
// should not be used to compare clients.  Use IsSame to compare clients
// for equality.
func (c Terminal) String() string {
	return "Terminal(" + capnp.Client(c).String() + ")"
}

// AddRef creates a new Client that refers to the same capability as c.
// If c is nil or has resolved to null, then AddRef returns nil.
func (c Terminal) AddRef() Terminal {
	return Terminal(capnp.Client(c).AddRef())
}

// Release releases a capability reference.  If this is the last
// reference to the capability, then the underlying resources associated
// with the capability will be released.
//
// Release will panic if c has already been released, but not if c is
// nil or resolved to null.
func (c Terminal) Release() {
	capnp.Client(c).Release()
}

// Resolve blocks until the capability is fully resolved or the Context
// expires.
func (c Terminal) Resolve(ctx context.Context) error {
	return capnp.Client(c).Resolve(ctx)
}

func (c Terminal) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Client(c).EncodeAsPtr(seg)
}

func (Terminal) DecodeFromPtr(p capnp.Ptr) Terminal {
	return Terminal(capnp.Client{}.DecodeFromPtr(p))
}

// IsValid reports whether c is a valid reference to a capability.
// A reference is invalid if it is nil, has resolved to null, or has
// been released.
func (c Terminal) IsValid() bool {
	return capnp.Client(c).IsValid()
}

// IsSame reports whether c and other refer to a capability created by the
// same call to NewClient.  This can return false negatives if c or other
// are not fully resolved: use Resolve if this is an issue.  If either
// c or other are released, then IsSame panics.
func (c Terminal) IsSame(other Terminal) bool {
	return capnp.Client(c).IsSame(capnp.Client(other))
}

// Update the flowcontrol.FlowLimiter used to manage flow control for
// this client. This affects all future calls, but not calls already
// waiting to send. Passing nil sets the value to flowcontrol.NopLimiter,
// which is also the default.
func (c Terminal) SetFlowLimiter(lim fc.FlowLimiter) {
	capnp.Client(c).SetFlowLimiter(lim)
}

// Get the current flowcontrol.FlowLimiter used to manage flow control
// for this client.
func (c Terminal) GetFlowLimiter() fc.FlowLimiter {
	return capnp.Client(c).GetFlowLimiter()
}

// A Terminal_Server is a Terminal with a local implementation.
type Terminal_Server interface {
	Login(context.Context, Terminal_login) error
}

// Terminal_NewServer creates a new Server from an implementation of Terminal_Server.
func Terminal_NewServer(s Terminal_Server) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Terminal_Methods(nil, s), s, c)
}

// Terminal_ServerToClient creates a new Client from an implementation of Terminal_Server.
// The caller is responsible for calling Release on the returned Client.
func Terminal_ServerToClient(s Terminal_Server) Terminal {
	return Terminal(capnp.NewClient(Terminal_NewServer(s)))
}

// Terminal_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Terminal_Methods(methods []server.Method, s Terminal_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xe9885abc9e5f9481,
			MethodID:      0,
			InterfaceName: "auth.capnp:Terminal",
			MethodName:    "login",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Login(ctx, Terminal_login{call})
		},
	})

	return methods
}

// Terminal_login holds the state for a server call to Terminal.login.
// See server.Call for documentation.
type Terminal_login struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Terminal_login) Args() Terminal_login_Params {
	return Terminal_login_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Terminal_login) AllocResults() (Terminal_login_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Terminal_login_Results(r), err
}

// Terminal_List is a list of Terminal.
type Terminal_List = capnp.CapList[Terminal]

// NewTerminal creates a new list of Terminal.
func NewTerminal_List(s *capnp.Segment, sz int32) (Terminal_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Terminal](l), err
}

type Terminal_Status capnp.Struct
type Terminal_Status_Which uint16

const (
	Terminal_Status_Which_success Terminal_Status_Which = 0
	Terminal_Status_Which_failure Terminal_Status_Which = 1
)

func (w Terminal_Status_Which) String() string {
	const s = "successfailure"
	switch w {
	case Terminal_Status_Which_success:
		return s[0:7]
	case Terminal_Status_Which_failure:
		return s[7:14]

	}
	return "Terminal_Status_Which(" + strconv.FormatUint(uint64(w), 10) + ")"
}

// Terminal_Status_TypeID is the unique identifier for the type Terminal_Status.
const Terminal_Status_TypeID = 0x8932d3dc82306df4

func NewTerminal_Status(s *capnp.Segment) (Terminal_Status, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Terminal_Status(st), err
}

func NewRootTerminal_Status(s *capnp.Segment) (Terminal_Status, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1})
	return Terminal_Status(st), err
}

func ReadRootTerminal_Status(msg *capnp.Message) (Terminal_Status, error) {
	root, err := msg.Root()
	return Terminal_Status(root.Struct()), err
}

func (s Terminal_Status) String() string {
	str, _ := text.Marshal(0x8932d3dc82306df4, capnp.Struct(s))
	return str
}

func (s Terminal_Status) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Terminal_Status) DecodeFromPtr(p capnp.Ptr) Terminal_Status {
	return Terminal_Status(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Terminal_Status) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}

func (s Terminal_Status) Which() Terminal_Status_Which {
	return Terminal_Status_Which(capnp.Struct(s).Uint16(0))
}
func (s Terminal_Status) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Terminal_Status) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Terminal_Status) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Terminal_Status) Success() (Session, error) {
	if capnp.Struct(s).Uint16(0) != 0 {
		panic("Which() != success")
	}
	p, err := capnp.Struct(s).Ptr(0)
	return Session(p.Struct()), err
}

func (s Terminal_Status) HasSuccess() bool {
	if capnp.Struct(s).Uint16(0) != 0 {
		return false
	}
	return capnp.Struct(s).HasPtr(0)
}

func (s Terminal_Status) SetSuccess(v Session) error {
	capnp.Struct(s).SetUint16(0, 0)
	return capnp.Struct(s).SetPtr(0, capnp.Struct(v).ToPtr())
}

// NewSuccess sets the success field to a newly
// allocated Session struct, preferring placement in s's segment.
func (s Terminal_Status) NewSuccess() (Session, error) {
	capnp.Struct(s).SetUint16(0, 0)
	ss, err := NewSession(capnp.Struct(s).Segment())
	if err != nil {
		return Session{}, err
	}
	err = capnp.Struct(s).SetPtr(0, capnp.Struct(ss).ToPtr())
	return ss, err
}

func (s Terminal_Status) Failure() (string, error) {
	if capnp.Struct(s).Uint16(0) != 1 {
		panic("Which() != failure")
	}
	p, err := capnp.Struct(s).Ptr(0)
	return p.Text(), err
}

func (s Terminal_Status) HasFailure() bool {
	if capnp.Struct(s).Uint16(0) != 1 {
		return false
	}
	return capnp.Struct(s).HasPtr(0)
}

func (s Terminal_Status) FailureBytes() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return p.TextBytes(), err
}

func (s Terminal_Status) SetFailure(v string) error {
	capnp.Struct(s).SetUint16(0, 1)
	return capnp.Struct(s).SetText(0, v)
}

// Terminal_Status_List is a list of Terminal_Status.
type Terminal_Status_List = capnp.StructList[Terminal_Status]

// NewTerminal_Status creates a new list of Terminal_Status.
func NewTerminal_Status_List(s *capnp.Segment, sz int32) (Terminal_Status_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 1}, sz)
	return capnp.StructList[Terminal_Status](l), err
}

// Terminal_Status_Future is a wrapper for a Terminal_Status promised by a client call.
type Terminal_Status_Future struct{ *capnp.Future }

func (f Terminal_Status_Future) Struct() (Terminal_Status, error) {
	p, err := f.Future.Ptr()
	return Terminal_Status(p.Struct()), err
}
func (p Terminal_Status_Future) Success() Session_Future {
	return Session_Future{Future: p.Future.Field(0, nil)}
}

type Terminal_login_Params capnp.Struct

// Terminal_login_Params_TypeID is the unique identifier for the type Terminal_login_Params.
const Terminal_login_Params_TypeID = 0xf431cebfcf594719

func NewTerminal_login_Params(s *capnp.Segment) (Terminal_login_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Terminal_login_Params(st), err
}

func NewRootTerminal_login_Params(s *capnp.Segment) (Terminal_login_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Terminal_login_Params(st), err
}

func ReadRootTerminal_login_Params(msg *capnp.Message) (Terminal_login_Params, error) {
	root, err := msg.Root()
	return Terminal_login_Params(root.Struct()), err
}

func (s Terminal_login_Params) String() string {
	str, _ := text.Marshal(0xf431cebfcf594719, capnp.Struct(s))
	return str
}

func (s Terminal_login_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Terminal_login_Params) DecodeFromPtr(p capnp.Ptr) Terminal_login_Params {
	return Terminal_login_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Terminal_login_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Terminal_login_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Terminal_login_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Terminal_login_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Terminal_login_Params) Account() Signer {
	p, _ := capnp.Struct(s).Ptr(0)
	return Signer(p.Interface().Client())
}

func (s Terminal_login_Params) HasAccount() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Terminal_login_Params) SetAccount(v Signer) error {
	if !v.IsValid() {
		return capnp.Struct(s).SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().CapTable().Add(capnp.Client(v)))
	return capnp.Struct(s).SetPtr(0, in.ToPtr())
}

// Terminal_login_Params_List is a list of Terminal_login_Params.
type Terminal_login_Params_List = capnp.StructList[Terminal_login_Params]

// NewTerminal_login_Params creates a new list of Terminal_login_Params.
func NewTerminal_login_Params_List(s *capnp.Segment, sz int32) (Terminal_login_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Terminal_login_Params](l), err
}

// Terminal_login_Params_Future is a wrapper for a Terminal_login_Params promised by a client call.
type Terminal_login_Params_Future struct{ *capnp.Future }

func (f Terminal_login_Params_Future) Struct() (Terminal_login_Params, error) {
	p, err := f.Future.Ptr()
	return Terminal_login_Params(p.Struct()), err
}
func (p Terminal_login_Params_Future) Account() Signer {
	return Signer(p.Future.Field(0, nil).Client())
}

type Terminal_login_Results capnp.Struct

// Terminal_login_Results_TypeID is the unique identifier for the type Terminal_login_Results.
const Terminal_login_Results_TypeID = 0xfa21596ea7e84ffe

func NewTerminal_login_Results(s *capnp.Segment) (Terminal_login_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Terminal_login_Results(st), err
}

func NewRootTerminal_login_Results(s *capnp.Segment) (Terminal_login_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Terminal_login_Results(st), err
}

func ReadRootTerminal_login_Results(msg *capnp.Message) (Terminal_login_Results, error) {
	root, err := msg.Root()
	return Terminal_login_Results(root.Struct()), err
}

func (s Terminal_login_Results) String() string {
	str, _ := text.Marshal(0xfa21596ea7e84ffe, capnp.Struct(s))
	return str
}

func (s Terminal_login_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Terminal_login_Results) DecodeFromPtr(p capnp.Ptr) Terminal_login_Results {
	return Terminal_login_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Terminal_login_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Terminal_login_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Terminal_login_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Terminal_login_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Terminal_login_Results) Status() (Terminal_Status, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return Terminal_Status(p.Struct()), err
}

func (s Terminal_login_Results) HasStatus() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Terminal_login_Results) SetStatus(v Terminal_Status) error {
	return capnp.Struct(s).SetPtr(0, capnp.Struct(v).ToPtr())
}

// NewStatus sets the status field to a newly
// allocated Terminal_Status struct, preferring placement in s's segment.
func (s Terminal_login_Results) NewStatus() (Terminal_Status, error) {
	ss, err := NewTerminal_Status(capnp.Struct(s).Segment())
	if err != nil {
		return Terminal_Status{}, err
	}
	err = capnp.Struct(s).SetPtr(0, capnp.Struct(ss).ToPtr())
	return ss, err
}

// Terminal_login_Results_List is a list of Terminal_login_Results.
type Terminal_login_Results_List = capnp.StructList[Terminal_login_Results]

// NewTerminal_login_Results creates a new list of Terminal_login_Results.
func NewTerminal_login_Results_List(s *capnp.Segment, sz int32) (Terminal_login_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Terminal_login_Results](l), err
}

// Terminal_login_Results_Future is a wrapper for a Terminal_login_Results promised by a client call.
type Terminal_login_Results_Future struct{ *capnp.Future }

func (f Terminal_login_Results_Future) Struct() (Terminal_login_Results, error) {
	p, err := f.Future.Ptr()
	return Terminal_login_Results(p.Struct()), err
}
func (p Terminal_login_Results_Future) Status() Terminal_Status_Future {
	return Terminal_Status_Future{Future: p.Future.Field(0, nil)}
}

type Signer capnp.Client

// Signer_TypeID is the unique identifier for the type Signer.
const Signer_TypeID = 0xf6d7ee5d0ce04043

func (c Signer) Sign(ctx context.Context, params func(Signer_sign_Params) error) (Signer_sign_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xf6d7ee5d0ce04043,
			MethodID:      0,
			InterfaceName: "auth.capnp:Signer",
			MethodName:    "sign",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Signer_sign_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Signer_sign_Results_Future{Future: ans.Future()}, release

}

func (c Signer) WaitStreaming() error {
	return capnp.Client(c).WaitStreaming()
}

// String returns a string that identifies this capability for debugging
// purposes.  Its format should not be depended on: in particular, it
// should not be used to compare clients.  Use IsSame to compare clients
// for equality.
func (c Signer) String() string {
	return "Signer(" + capnp.Client(c).String() + ")"
}

// AddRef creates a new Client that refers to the same capability as c.
// If c is nil or has resolved to null, then AddRef returns nil.
func (c Signer) AddRef() Signer {
	return Signer(capnp.Client(c).AddRef())
}

// Release releases a capability reference.  If this is the last
// reference to the capability, then the underlying resources associated
// with the capability will be released.
//
// Release will panic if c has already been released, but not if c is
// nil or resolved to null.
func (c Signer) Release() {
	capnp.Client(c).Release()
}

// Resolve blocks until the capability is fully resolved or the Context
// expires.
func (c Signer) Resolve(ctx context.Context) error {
	return capnp.Client(c).Resolve(ctx)
}

func (c Signer) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Client(c).EncodeAsPtr(seg)
}

func (Signer) DecodeFromPtr(p capnp.Ptr) Signer {
	return Signer(capnp.Client{}.DecodeFromPtr(p))
}

// IsValid reports whether c is a valid reference to a capability.
// A reference is invalid if it is nil, has resolved to null, or has
// been released.
func (c Signer) IsValid() bool {
	return capnp.Client(c).IsValid()
}

// IsSame reports whether c and other refer to a capability created by the
// same call to NewClient.  This can return false negatives if c or other
// are not fully resolved: use Resolve if this is an issue.  If either
// c or other are released, then IsSame panics.
func (c Signer) IsSame(other Signer) bool {
	return capnp.Client(c).IsSame(capnp.Client(other))
}

// Update the flowcontrol.FlowLimiter used to manage flow control for
// this client. This affects all future calls, but not calls already
// waiting to send. Passing nil sets the value to flowcontrol.NopLimiter,
// which is also the default.
func (c Signer) SetFlowLimiter(lim fc.FlowLimiter) {
	capnp.Client(c).SetFlowLimiter(lim)
}

// Get the current flowcontrol.FlowLimiter used to manage flow control
// for this client.
func (c Signer) GetFlowLimiter() fc.FlowLimiter {
	return capnp.Client(c).GetFlowLimiter()
}

// A Signer_Server is a Signer with a local implementation.
type Signer_Server interface {
	Sign(context.Context, Signer_sign) error
}

// Signer_NewServer creates a new Server from an implementation of Signer_Server.
func Signer_NewServer(s Signer_Server) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Signer_Methods(nil, s), s, c)
}

// Signer_ServerToClient creates a new Client from an implementation of Signer_Server.
// The caller is responsible for calling Release on the returned Client.
func Signer_ServerToClient(s Signer_Server) Signer {
	return Signer(capnp.NewClient(Signer_NewServer(s)))
}

// Signer_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Signer_Methods(methods []server.Method, s Signer_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xf6d7ee5d0ce04043,
			MethodID:      0,
			InterfaceName: "auth.capnp:Signer",
			MethodName:    "sign",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Sign(ctx, Signer_sign{call})
		},
	})

	return methods
}

// Signer_sign holds the state for a server call to Signer.sign.
// See server.Call for documentation.
type Signer_sign struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Signer_sign) Args() Signer_sign_Params {
	return Signer_sign_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Signer_sign) AllocResults() (Signer_sign_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Signer_sign_Results(r), err
}

// Signer_List is a list of Signer.
type Signer_List = capnp.CapList[Signer]

// NewSigner creates a new list of Signer.
func NewSigner_List(s *capnp.Segment, sz int32) (Signer_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Signer](l), err
}

type Signer_sign_Params capnp.Struct

// Signer_sign_Params_TypeID is the unique identifier for the type Signer_sign_Params.
const Signer_sign_Params_TypeID = 0xd185e18419bf9fff

func NewSigner_sign_Params(s *capnp.Segment) (Signer_sign_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Signer_sign_Params(st), err
}

func NewRootSigner_sign_Params(s *capnp.Segment) (Signer_sign_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Signer_sign_Params(st), err
}

func ReadRootSigner_sign_Params(msg *capnp.Message) (Signer_sign_Params, error) {
	root, err := msg.Root()
	return Signer_sign_Params(root.Struct()), err
}

func (s Signer_sign_Params) String() string {
	str, _ := text.Marshal(0xd185e18419bf9fff, capnp.Struct(s))
	return str
}

func (s Signer_sign_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Signer_sign_Params) DecodeFromPtr(p capnp.Ptr) Signer_sign_Params {
	return Signer_sign_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Signer_sign_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Signer_sign_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Signer_sign_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Signer_sign_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Signer_sign_Params) Challenge() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return []byte(p.Data()), err
}

func (s Signer_sign_Params) HasChallenge() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Signer_sign_Params) SetChallenge(v []byte) error {
	return capnp.Struct(s).SetData(0, v)
}

// Signer_sign_Params_List is a list of Signer_sign_Params.
type Signer_sign_Params_List = capnp.StructList[Signer_sign_Params]

// NewSigner_sign_Params creates a new list of Signer_sign_Params.
func NewSigner_sign_Params_List(s *capnp.Segment, sz int32) (Signer_sign_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Signer_sign_Params](l), err
}

// Signer_sign_Params_Future is a wrapper for a Signer_sign_Params promised by a client call.
type Signer_sign_Params_Future struct{ *capnp.Future }

func (f Signer_sign_Params_Future) Struct() (Signer_sign_Params, error) {
	p, err := f.Future.Ptr()
	return Signer_sign_Params(p.Struct()), err
}

type Signer_sign_Results capnp.Struct

// Signer_sign_Results_TypeID is the unique identifier for the type Signer_sign_Results.
const Signer_sign_Results_TypeID = 0x99a87339e9b14f0b

func NewSigner_sign_Results(s *capnp.Segment) (Signer_sign_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Signer_sign_Results(st), err
}

func NewRootSigner_sign_Results(s *capnp.Segment) (Signer_sign_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Signer_sign_Results(st), err
}

func ReadRootSigner_sign_Results(msg *capnp.Message) (Signer_sign_Results, error) {
	root, err := msg.Root()
	return Signer_sign_Results(root.Struct()), err
}

func (s Signer_sign_Results) String() string {
	str, _ := text.Marshal(0x99a87339e9b14f0b, capnp.Struct(s))
	return str
}

func (s Signer_sign_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Signer_sign_Results) DecodeFromPtr(p capnp.Ptr) Signer_sign_Results {
	return Signer_sign_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Signer_sign_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Signer_sign_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Signer_sign_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Signer_sign_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Signer_sign_Results) Signed() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return []byte(p.Data()), err
}

func (s Signer_sign_Results) HasSigned() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Signer_sign_Results) SetSigned(v []byte) error {
	return capnp.Struct(s).SetData(0, v)
}

// Signer_sign_Results_List is a list of Signer_sign_Results.
type Signer_sign_Results_List = capnp.StructList[Signer_sign_Results]

// NewSigner_sign_Results creates a new list of Signer_sign_Results.
func NewSigner_sign_Results_List(s *capnp.Segment, sz int32) (Signer_sign_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Signer_sign_Results](l), err
}

// Signer_sign_Results_Future is a wrapper for a Signer_sign_Results promised by a client call.
type Signer_sign_Results_Future struct{ *capnp.Future }

func (f Signer_sign_Results_Future) Struct() (Signer_sign_Results, error) {
	p, err := f.Future.Ptr()
	return Signer_sign_Results(p.Struct()), err
}

const schema_81484d9336a7c5d3 = "x\xda\x84\x93Oh\x13i\x18\xc6\xdf\xe7\x9bd\xd3\x9d" +
	"Ih\xbfN\x96\xdd\xb2\xbb\x94]r\xd8-l\xe9\x9f" +
	"\xfd\xc3\xe6\x92\xd0\xdd\x92.li\xbe\xb6\x97VD\xa6" +
	"qL\x07\xd2I\xcddZ<\x95\x8a\x82\xedA\xf0\xdf" +
	"\xa1\x05\x15\x05\xa9z\x11\x11oJO\xe2IEz\x10" +
	"ED\xa5\x88\x05\x11\x91\xe2\xa1H?\xf9\xa6M\x93\xaa" +
	"\xc5\xeb\xfb\xce\xfc\xde\xe7y\xdf\xe7k\xdb\xa5\xa5C\xed" +
	"\xb1\xbe81\xf1*\xfc\x95\\\x1dk;\xf8x\xa9c" +
	"\x86D\x1c\x90\xd3'\xf7\x9c\xb91|d\x85\xba\x11a" +
	"DfL\x7fd6\xe9\x11\"\xf3\x1b}\x92 \x8d\xbe" +
	"\xab+\x7f{\x17\xe7\x887\x82(\x0c\xd5\xd9\xaf\xaf\x11" +
	"L_O\x11\xe4\xec\xf3\x0cf~\xb8|\x9b\xb8\x01\xb9" +
	"tk\xe1\xcf\x13\xbd=\xd3\x14\xd6\xd4w\xa7\xf4y\xf3" +
	"t\xc0\x9a\xd3\xaf\x10\xa4<\xbb\xd8t\xe8\xd9\xe1\xfb\xb5" +
	"\xacv\xe3-\xc1\xfc\xc3P\xac--\xdc\xd0\xaa,\x82" +
	"9d\x9c7-\xe3[\"\xd312\xe61#B$" +
	"\x9b2C\xf7\x16\xef\xb6\xaf\x12\x8fW`\x9d\x07\x0c\x1d" +
	"\x04s:\xa0\xfd\x93~\x1a\xdd\xfd\xfa\xc1\xbbOh\xe7" +
	"\x8c\xe3\xe6%\xc50/\x18\x19\xf3N@[\xef{\xb9" +
	"\xe0\x0e\xfd\xb4VK\xbbn4*\xdaM#E\x92$" +
	"uH\xcb/\x8f\xb6\xe6\xacq\xe6\x8e'\x07\xed\xd2\x98" +
	"\xe3Z\x85\xd6\x81\xb2U\xd6|/\x0b\x88:-\x14\x95" +
	"2\x04\"\xfek\x17\x91Hh\x10m\x0c1\xac\xcb8" +
	"T\xf57U\xfdE\x83\xf8\x9da\xca\xf3s9\xdb\xf3" +
	"\xd0P\xdd!\x01\x0d\x84\xa9}\x96S\xf0K6\xa2\xc4" +
	"\x10%l\x9b;\xe0\xe4]\xbb\xd4\xea9y7\xd1\x9f" +
	"\xb2=\xbfP\xf6DH\x0b\x11\x05scI\"Q\xa7" +
	"A\xc4\x19R\xea#{/b\xc4\x10\xab\xc1@al" +
	"\xcfs\xb4\xa2\xabdG\xb7\xfe\xeen!\x12i\x0d\xe2" +
	"\x7f\x06\x0el\x88\xfeO\x15\xff\xd5 \xb2\x0c\x9c\xb18" +
	"\x18\x11\xefUsz4\x88A\x86\xfa\x09\xc7\x9e\x04\x97" +
	"\xf3\x89\xf7\xc3\x9do~\x9cU68\xa1\xbeT,\x96" +
	"\xc1\xe5\xcf\x0f\x8f~\xbd\xfcW\xe3\xf2f95\xee\x8f" +
	"x\xfe\x08\xb8L\x16'\xbe\x7fq-\xfbd\xb3\xb1\xa3" +
	"\xcdl\xb3U\xb2\xc6\xb6\xb9\xec'\x12Q\x0d\xe2;\x06" +
	"\x99\x1b\xb5\x0a\x05\xdb\xcd\x13\xec\xcfZ\x0d.\x15q\xad" +
	"\x82\x08\x01\xd5\xf4s$S\xeav\xbe\x02\x87k\x02\x85" +
	"J\x168\xef \xc6\xc3\x91\xe6B1\xef\xb8idQ" +
	"\x05k\xb5\x11\x08\xfa\x89l\xa0\x92\xa8VgW\xf5\x1a" +
	"SV.W\xf4]\xb5\x91\xadd~d\x1c\x15\xe3(" +
	"\xa9\xbbl\xc8\xaa<\x1aT^\"\xe7-\x81\xacz\xb5" +
	"\x9c/\xaa\xea\x0f\"\x82\x9d3\x12\xac\x00\x0d\xd5\xc5l" +
	"\x84\xf0C\x00\x00\x00\xff\xffoT%\xcb"

func RegisterSchema(reg *schemas.Registry) {
	reg.Register(&schemas.Schema{
		String: schema_81484d9336a7c5d3,
		Nodes: []uint64{
			0x8932d3dc82306df4,
			0x99a87339e9b14f0b,
			0xc7aa1c890147e28a,
			0xd185e18419bf9fff,
			0xe9885abc9e5f9481,
			0xf431cebfcf594719,
			0xf6d7ee5d0ce04043,
			0xfa21596ea7e84ffe,
		},
		Compressed: true,
	})
}
