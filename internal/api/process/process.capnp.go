// Code generated by capnpc-go. DO NOT EDIT.

package process

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	fc "capnproto.org/go/capnp/v3/flowcontrol"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
)

type Executor capnp.Client

// Executor_TypeID is the unique identifier for the type Executor.
const Executor_TypeID = 0xaf2e5ebaa58175d2

func (c Executor) Exec(ctx context.Context, params func(Executor_exec_Params) error) (Executor_exec_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xaf2e5ebaa58175d2,
			MethodID:      0,
			InterfaceName: "process.capnp:Executor",
			MethodName:    "exec",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 1}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Executor_exec_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Executor_exec_Results_Future{Future: ans.Future()}, release

}

func (c Executor) WaitStreaming() error {
	return capnp.Client(c).WaitStreaming()
}

// String returns a string that identifies this capability for debugging
// purposes.  Its format should not be depended on: in particular, it
// should not be used to compare clients.  Use IsSame to compare clients
// for equality.
func (c Executor) String() string {
	return "Executor(" + capnp.Client(c).String() + ")"
}

// AddRef creates a new Client that refers to the same capability as c.
// If c is nil or has resolved to null, then AddRef returns nil.
func (c Executor) AddRef() Executor {
	return Executor(capnp.Client(c).AddRef())
}

// Release releases a capability reference.  If this is the last
// reference to the capability, then the underlying resources associated
// with the capability will be released.
//
// Release will panic if c has already been released, but not if c is
// nil or resolved to null.
func (c Executor) Release() {
	capnp.Client(c).Release()
}

// Resolve blocks until the capability is fully resolved or the Context
// expires.
func (c Executor) Resolve(ctx context.Context) error {
	return capnp.Client(c).Resolve(ctx)
}

func (c Executor) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Client(c).EncodeAsPtr(seg)
}

func (Executor) DecodeFromPtr(p capnp.Ptr) Executor {
	return Executor(capnp.Client{}.DecodeFromPtr(p))
}

// IsValid reports whether c is a valid reference to a capability.
// A reference is invalid if it is nil, has resolved to null, or has
// been released.
func (c Executor) IsValid() bool {
	return capnp.Client(c).IsValid()
}

// IsSame reports whether c and other refer to a capability created by the
// same call to NewClient.  This can return false negatives if c or other
// are not fully resolved: use Resolve if this is an issue.  If either
// c or other are released, then IsSame panics.
func (c Executor) IsSame(other Executor) bool {
	return capnp.Client(c).IsSame(capnp.Client(other))
}

// Update the flowcontrol.FlowLimiter used to manage flow control for
// this client. This affects all future calls, but not calls already
// waiting to send. Passing nil sets the value to flowcontrol.NopLimiter,
// which is also the default.
func (c Executor) SetFlowLimiter(lim fc.FlowLimiter) {
	capnp.Client(c).SetFlowLimiter(lim)
}

// Get the current flowcontrol.FlowLimiter used to manage flow control
// for this client.
func (c Executor) GetFlowLimiter() fc.FlowLimiter {
	return capnp.Client(c).GetFlowLimiter()
}

// A Executor_Server is a Executor with a local implementation.
type Executor_Server interface {
	Exec(context.Context, Executor_exec) error
}

// Executor_NewServer creates a new Server from an implementation of Executor_Server.
func Executor_NewServer(s Executor_Server) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Executor_Methods(nil, s), s, c)
}

// Executor_ServerToClient creates a new Client from an implementation of Executor_Server.
// The caller is responsible for calling Release on the returned Client.
func Executor_ServerToClient(s Executor_Server) Executor {
	return Executor(capnp.NewClient(Executor_NewServer(s)))
}

// Executor_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Executor_Methods(methods []server.Method, s Executor_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xaf2e5ebaa58175d2,
			MethodID:      0,
			InterfaceName: "process.capnp:Executor",
			MethodName:    "exec",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Exec(ctx, Executor_exec{call})
		},
	})

	return methods
}

// Executor_exec holds the state for a server call to Executor.exec.
// See server.Call for documentation.
type Executor_exec struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Executor_exec) Args() Executor_exec_Params {
	return Executor_exec_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Executor_exec) AllocResults() (Executor_exec_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Executor_exec_Results(r), err
}

// Executor_List is a list of Executor.
type Executor_List = capnp.CapList[Executor]

// NewExecutor creates a new list of Executor.
func NewExecutor_List(s *capnp.Segment, sz int32) (Executor_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Executor](l), err
}

type Executor_exec_Params capnp.Struct

// Executor_exec_Params_TypeID is the unique identifier for the type Executor_exec_Params.
const Executor_exec_Params_TypeID = 0xf20b3dea95929312

func NewExecutor_exec_Params(s *capnp.Segment) (Executor_exec_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Executor_exec_Params(st), err
}

func NewRootExecutor_exec_Params(s *capnp.Segment) (Executor_exec_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Executor_exec_Params(st), err
}

func ReadRootExecutor_exec_Params(msg *capnp.Message) (Executor_exec_Params, error) {
	root, err := msg.Root()
	return Executor_exec_Params(root.Struct()), err
}

func (s Executor_exec_Params) String() string {
	str, _ := text.Marshal(0xf20b3dea95929312, capnp.Struct(s))
	return str
}

func (s Executor_exec_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Executor_exec_Params) DecodeFromPtr(p capnp.Ptr) Executor_exec_Params {
	return Executor_exec_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Executor_exec_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Executor_exec_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Executor_exec_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Executor_exec_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Executor_exec_Params) Bytecode() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return []byte(p.Data()), err
}

func (s Executor_exec_Params) HasBytecode() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Executor_exec_Params) SetBytecode(v []byte) error {
	return capnp.Struct(s).SetData(0, v)
}

// Executor_exec_Params_List is a list of Executor_exec_Params.
type Executor_exec_Params_List = capnp.StructList[Executor_exec_Params]

// NewExecutor_exec_Params creates a new list of Executor_exec_Params.
func NewExecutor_exec_Params_List(s *capnp.Segment, sz int32) (Executor_exec_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Executor_exec_Params](l), err
}

// Executor_exec_Params_Future is a wrapper for a Executor_exec_Params promised by a client call.
type Executor_exec_Params_Future struct{ *capnp.Future }

func (f Executor_exec_Params_Future) Struct() (Executor_exec_Params, error) {
	p, err := f.Future.Ptr()
	return Executor_exec_Params(p.Struct()), err
}

type Executor_exec_Results capnp.Struct

// Executor_exec_Results_TypeID is the unique identifier for the type Executor_exec_Results.
const Executor_exec_Results_TypeID = 0xbb4f16b0a7d2d09b

func NewExecutor_exec_Results(s *capnp.Segment) (Executor_exec_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Executor_exec_Results(st), err
}

func NewRootExecutor_exec_Results(s *capnp.Segment) (Executor_exec_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1})
	return Executor_exec_Results(st), err
}

func ReadRootExecutor_exec_Results(msg *capnp.Message) (Executor_exec_Results, error) {
	root, err := msg.Root()
	return Executor_exec_Results(root.Struct()), err
}

func (s Executor_exec_Results) String() string {
	str, _ := text.Marshal(0xbb4f16b0a7d2d09b, capnp.Struct(s))
	return str
}

func (s Executor_exec_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Executor_exec_Results) DecodeFromPtr(p capnp.Ptr) Executor_exec_Results {
	return Executor_exec_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Executor_exec_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Executor_exec_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Executor_exec_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Executor_exec_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Executor_exec_Results) Process() Process {
	p, _ := capnp.Struct(s).Ptr(0)
	return Process(p.Interface().Client())
}

func (s Executor_exec_Results) HasProcess() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Executor_exec_Results) SetProcess(v Process) error {
	if !v.IsValid() {
		return capnp.Struct(s).SetPtr(0, capnp.Ptr{})
	}
	seg := s.Segment()
	in := capnp.NewInterface(seg, seg.Message().CapTable().Add(capnp.Client(v)))
	return capnp.Struct(s).SetPtr(0, in.ToPtr())
}

// Executor_exec_Results_List is a list of Executor_exec_Results.
type Executor_exec_Results_List = capnp.StructList[Executor_exec_Results]

// NewExecutor_exec_Results creates a new list of Executor_exec_Results.
func NewExecutor_exec_Results_List(s *capnp.Segment, sz int32) (Executor_exec_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 1}, sz)
	return capnp.StructList[Executor_exec_Results](l), err
}

// Executor_exec_Results_Future is a wrapper for a Executor_exec_Results promised by a client call.
type Executor_exec_Results_Future struct{ *capnp.Future }

func (f Executor_exec_Results_Future) Struct() (Executor_exec_Results, error) {
	p, err := f.Future.Ptr()
	return Executor_exec_Results(p.Struct()), err
}
func (p Executor_exec_Results_Future) Process() Process {
	return Process(p.Future.Field(0, nil).Client())
}

type Process capnp.Client

// Process_TypeID is the unique identifier for the type Process.
const Process_TypeID = 0xda23f0d3a8250633

func (c Process) Wait(ctx context.Context, params func(Process_wait_Params) error) (Process_wait_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xda23f0d3a8250633,
			MethodID:      0,
			InterfaceName: "process.capnp:Process",
			MethodName:    "wait",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 0}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Process_wait_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Process_wait_Results_Future{Future: ans.Future()}, release

}

func (c Process) Kill(ctx context.Context, params func(Process_kill_Params) error) (Process_kill_Results_Future, capnp.ReleaseFunc) {

	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xda23f0d3a8250633,
			MethodID:      1,
			InterfaceName: "process.capnp:Process",
			MethodName:    "kill",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 0, PointerCount: 0}
		s.PlaceArgs = func(s capnp.Struct) error { return params(Process_kill_Params(s)) }
	}

	ans, release := capnp.Client(c).SendCall(ctx, s)
	return Process_kill_Results_Future{Future: ans.Future()}, release

}

func (c Process) WaitStreaming() error {
	return capnp.Client(c).WaitStreaming()
}

// String returns a string that identifies this capability for debugging
// purposes.  Its format should not be depended on: in particular, it
// should not be used to compare clients.  Use IsSame to compare clients
// for equality.
func (c Process) String() string {
	return "Process(" + capnp.Client(c).String() + ")"
}

// AddRef creates a new Client that refers to the same capability as c.
// If c is nil or has resolved to null, then AddRef returns nil.
func (c Process) AddRef() Process {
	return Process(capnp.Client(c).AddRef())
}

// Release releases a capability reference.  If this is the last
// reference to the capability, then the underlying resources associated
// with the capability will be released.
//
// Release will panic if c has already been released, but not if c is
// nil or resolved to null.
func (c Process) Release() {
	capnp.Client(c).Release()
}

// Resolve blocks until the capability is fully resolved or the Context
// expires.
func (c Process) Resolve(ctx context.Context) error {
	return capnp.Client(c).Resolve(ctx)
}

func (c Process) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Client(c).EncodeAsPtr(seg)
}

func (Process) DecodeFromPtr(p capnp.Ptr) Process {
	return Process(capnp.Client{}.DecodeFromPtr(p))
}

// IsValid reports whether c is a valid reference to a capability.
// A reference is invalid if it is nil, has resolved to null, or has
// been released.
func (c Process) IsValid() bool {
	return capnp.Client(c).IsValid()
}

// IsSame reports whether c and other refer to a capability created by the
// same call to NewClient.  This can return false negatives if c or other
// are not fully resolved: use Resolve if this is an issue.  If either
// c or other are released, then IsSame panics.
func (c Process) IsSame(other Process) bool {
	return capnp.Client(c).IsSame(capnp.Client(other))
}

// Update the flowcontrol.FlowLimiter used to manage flow control for
// this client. This affects all future calls, but not calls already
// waiting to send. Passing nil sets the value to flowcontrol.NopLimiter,
// which is also the default.
func (c Process) SetFlowLimiter(lim fc.FlowLimiter) {
	capnp.Client(c).SetFlowLimiter(lim)
}

// Get the current flowcontrol.FlowLimiter used to manage flow control
// for this client.
func (c Process) GetFlowLimiter() fc.FlowLimiter {
	return capnp.Client(c).GetFlowLimiter()
}

// A Process_Server is a Process with a local implementation.
type Process_Server interface {
	Wait(context.Context, Process_wait) error

	Kill(context.Context, Process_kill) error
}

// Process_NewServer creates a new Server from an implementation of Process_Server.
func Process_NewServer(s Process_Server) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(Process_Methods(nil, s), s, c)
}

// Process_ServerToClient creates a new Client from an implementation of Process_Server.
// The caller is responsible for calling Release on the returned Client.
func Process_ServerToClient(s Process_Server) Process {
	return Process(capnp.NewClient(Process_NewServer(s)))
}

// Process_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func Process_Methods(methods []server.Method, s Process_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 2)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xda23f0d3a8250633,
			MethodID:      0,
			InterfaceName: "process.capnp:Process",
			MethodName:    "wait",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Wait(ctx, Process_wait{call})
		},
	})

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xda23f0d3a8250633,
			MethodID:      1,
			InterfaceName: "process.capnp:Process",
			MethodName:    "kill",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.Kill(ctx, Process_kill{call})
		},
	})

	return methods
}

// Process_wait holds the state for a server call to Process.wait.
// See server.Call for documentation.
type Process_wait struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Process_wait) Args() Process_wait_Params {
	return Process_wait_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Process_wait) AllocResults() (Process_wait_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return Process_wait_Results(r), err
}

// Process_kill holds the state for a server call to Process.kill.
// See server.Call for documentation.
type Process_kill struct {
	*server.Call
}

// Args returns the call's arguments.
func (c Process_kill) Args() Process_kill_Params {
	return Process_kill_Params(c.Call.Args())
}

// AllocResults allocates the results struct.
func (c Process_kill) AllocResults() (Process_kill_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_kill_Results(r), err
}

// Process_List is a list of Process.
type Process_List = capnp.CapList[Process]

// NewProcess creates a new list of Process.
func NewProcess_List(s *capnp.Segment, sz int32) (Process_List, error) {
	l, err := capnp.NewPointerList(s, sz)
	return capnp.CapList[Process](l), err
}

type Process_wait_Params capnp.Struct

// Process_wait_Params_TypeID is the unique identifier for the type Process_wait_Params.
const Process_wait_Params_TypeID = 0xf9694ae208dbb3e3

func NewProcess_wait_Params(s *capnp.Segment) (Process_wait_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_wait_Params(st), err
}

func NewRootProcess_wait_Params(s *capnp.Segment) (Process_wait_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_wait_Params(st), err
}

func ReadRootProcess_wait_Params(msg *capnp.Message) (Process_wait_Params, error) {
	root, err := msg.Root()
	return Process_wait_Params(root.Struct()), err
}

func (s Process_wait_Params) String() string {
	str, _ := text.Marshal(0xf9694ae208dbb3e3, capnp.Struct(s))
	return str
}

func (s Process_wait_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Process_wait_Params) DecodeFromPtr(p capnp.Ptr) Process_wait_Params {
	return Process_wait_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Process_wait_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Process_wait_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Process_wait_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Process_wait_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}

// Process_wait_Params_List is a list of Process_wait_Params.
type Process_wait_Params_List = capnp.StructList[Process_wait_Params]

// NewProcess_wait_Params creates a new list of Process_wait_Params.
func NewProcess_wait_Params_List(s *capnp.Segment, sz int32) (Process_wait_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[Process_wait_Params](l), err
}

// Process_wait_Params_Future is a wrapper for a Process_wait_Params promised by a client call.
type Process_wait_Params_Future struct{ *capnp.Future }

func (f Process_wait_Params_Future) Struct() (Process_wait_Params, error) {
	p, err := f.Future.Ptr()
	return Process_wait_Params(p.Struct()), err
}

type Process_wait_Results capnp.Struct

// Process_wait_Results_TypeID is the unique identifier for the type Process_wait_Results.
const Process_wait_Results_TypeID = 0xd72ab4a0243047ac

func NewProcess_wait_Results(s *capnp.Segment) (Process_wait_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return Process_wait_Results(st), err
}

func NewRootProcess_wait_Results(s *capnp.Segment) (Process_wait_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return Process_wait_Results(st), err
}

func ReadRootProcess_wait_Results(msg *capnp.Message) (Process_wait_Results, error) {
	root, err := msg.Root()
	return Process_wait_Results(root.Struct()), err
}

func (s Process_wait_Results) String() string {
	str, _ := text.Marshal(0xd72ab4a0243047ac, capnp.Struct(s))
	return str
}

func (s Process_wait_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Process_wait_Results) DecodeFromPtr(p capnp.Ptr) Process_wait_Results {
	return Process_wait_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Process_wait_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Process_wait_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Process_wait_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Process_wait_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Process_wait_Results) ExitCode() uint32 {
	return capnp.Struct(s).Uint32(0)
}

func (s Process_wait_Results) SetExitCode(v uint32) {
	capnp.Struct(s).SetUint32(0, v)
}

// Process_wait_Results_List is a list of Process_wait_Results.
type Process_wait_Results_List = capnp.StructList[Process_wait_Results]

// NewProcess_wait_Results creates a new list of Process_wait_Results.
func NewProcess_wait_Results_List(s *capnp.Segment, sz int32) (Process_wait_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return capnp.StructList[Process_wait_Results](l), err
}

// Process_wait_Results_Future is a wrapper for a Process_wait_Results promised by a client call.
type Process_wait_Results_Future struct{ *capnp.Future }

func (f Process_wait_Results_Future) Struct() (Process_wait_Results, error) {
	p, err := f.Future.Ptr()
	return Process_wait_Results(p.Struct()), err
}

type Process_kill_Params capnp.Struct

// Process_kill_Params_TypeID is the unique identifier for the type Process_kill_Params.
const Process_kill_Params_TypeID = 0xeea7ae19b02f5d47

func NewProcess_kill_Params(s *capnp.Segment) (Process_kill_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_kill_Params(st), err
}

func NewRootProcess_kill_Params(s *capnp.Segment) (Process_kill_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_kill_Params(st), err
}

func ReadRootProcess_kill_Params(msg *capnp.Message) (Process_kill_Params, error) {
	root, err := msg.Root()
	return Process_kill_Params(root.Struct()), err
}

func (s Process_kill_Params) String() string {
	str, _ := text.Marshal(0xeea7ae19b02f5d47, capnp.Struct(s))
	return str
}

func (s Process_kill_Params) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Process_kill_Params) DecodeFromPtr(p capnp.Ptr) Process_kill_Params {
	return Process_kill_Params(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Process_kill_Params) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Process_kill_Params) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Process_kill_Params) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Process_kill_Params) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}

// Process_kill_Params_List is a list of Process_kill_Params.
type Process_kill_Params_List = capnp.StructList[Process_kill_Params]

// NewProcess_kill_Params creates a new list of Process_kill_Params.
func NewProcess_kill_Params_List(s *capnp.Segment, sz int32) (Process_kill_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[Process_kill_Params](l), err
}

// Process_kill_Params_Future is a wrapper for a Process_kill_Params promised by a client call.
type Process_kill_Params_Future struct{ *capnp.Future }

func (f Process_kill_Params_Future) Struct() (Process_kill_Params, error) {
	p, err := f.Future.Ptr()
	return Process_kill_Params(p.Struct()), err
}

type Process_kill_Results capnp.Struct

// Process_kill_Results_TypeID is the unique identifier for the type Process_kill_Results.
const Process_kill_Results_TypeID = 0xc53168b273d497ee

func NewProcess_kill_Results(s *capnp.Segment) (Process_kill_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_kill_Results(st), err
}

func NewRootProcess_kill_Results(s *capnp.Segment) (Process_kill_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0})
	return Process_kill_Results(st), err
}

func ReadRootProcess_kill_Results(msg *capnp.Message) (Process_kill_Results, error) {
	root, err := msg.Root()
	return Process_kill_Results(root.Struct()), err
}

func (s Process_kill_Results) String() string {
	str, _ := text.Marshal(0xc53168b273d497ee, capnp.Struct(s))
	return str
}

func (s Process_kill_Results) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Process_kill_Results) DecodeFromPtr(p capnp.Ptr) Process_kill_Results {
	return Process_kill_Results(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Process_kill_Results) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}
func (s Process_kill_Results) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Process_kill_Results) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Process_kill_Results) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}

// Process_kill_Results_List is a list of Process_kill_Results.
type Process_kill_Results_List = capnp.StructList[Process_kill_Results]

// NewProcess_kill_Results creates a new list of Process_kill_Results.
func NewProcess_kill_Results_List(s *capnp.Segment, sz int32) (Process_kill_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 0, PointerCount: 0}, sz)
	return capnp.StructList[Process_kill_Results](l), err
}

// Process_kill_Results_Future is a wrapper for a Process_kill_Results promised by a client call.
type Process_kill_Results_Future struct{ *capnp.Future }

func (f Process_kill_Results_Future) Struct() (Process_kill_Results, error) {
	p, err := f.Future.Ptr()
	return Process_kill_Results(p.Struct()), err
}

const schema_9a51e53177277763 = "x\xda|\x921h\x13Q\x18\xc7\xff\xffw/\xbd\x03" +
	"=\xc3\xebi\xb4\x82\x08\xdaP\xe8\x10\x13\x8a\x8b \x0d" +
	"\x8a\x04\x0bb\x9e\xbbB<\x0f\x0c&&\xe4.$N" +
	"\xe2 \xee\x8a\x88\x8a\xe0\"u\x10-\xd5\xa9\xce\xeeR" +
	"\x05A7\xc5\xc5\xa1\x88[\x079y\xd7\\<\xa9\xe9" +
	"v\x8f\xef\xfb\xfe\xff\xff\xf7\xfb\xae,Y\x95\x15wN" +
	"B\xe8rn*^\xef\xdfz\xb6v\xa9\xf4\x0aj\x8f" +
	"\x15\xfb\x83\xb9A\xe5\xbb~\x04\xd0\xdb\xe4\x9aGa\x03" +
	"\xdeo\xd6\xbc\xa2\xf9\x8a\x1f\xbf__^)\x9c\x7f\x0b" +
	"\xb5\x9f@\x8e6\xb0\xe0\x8a#\x04\xbd}b\x11\x8c7" +
	"\x1e|\x0cW\xafV\xdeA\x15\x08HS?.\x0e\x12" +
	"2~Q+\xcf>}3\xff\x09\xba\xc0\xb4t\xc8\x94" +
	"\xe8\x15\x93\xd1\x85\xa9\xe2\xf3\x0f?\x8f~\xde\x16\xe4\xac" +
	"X\xf5t\x12\xe4\x9c\xb8\xe3=I\x82\xd4.\x1e[\x99" +
	"y\xb9\xbc\x91\xf1\xb9-\xa6\x8d\xcf\xf4\xbd\xbb\xf7\x7f\x9c" +
	"\xdc\xf5+\x1b\xb1\xbd\xe5\xd3O|\xbe\xbd\xfe\xe2|]" +
	"jnfF\x1f\x9a\xd1r\xdc\xedu\xfc \x0cK\xf4" +
	"\x1b\xdd\xeb\xdd\x13g\x86\x8b\x81\xdf\x8f:\xbd:\xa9\xa5" +
	"\x95\x03\xc6\xe2LA(5\x0f\xa1rv>\x18\x06~" +
	"\x95ur\xacb\xa5*[\"%\xd31{!\x08\xfb" +
	"v+\x0a\xb5\xb4$ \x09(\xf7\x14\xa0\x1d\x8bz\xaf" +
	"\xe0\xcd\xd10\xd5_\x1c \x15\xb6\xc9\xd6G\xcfk\xcd" +
	"V+QmYQ8\xa9i\xd0hF\xe3\xa6\xac\xf5" +
	"\x12\xa0w[\xd4\x07\x04\xe3`\xd8\x8cNw\xae\x04\x00" +
	"\xe8@\xd0\xc9\x982\xd5;\x9c\xbc\x0d\x10'\x01\x92\xc2" +
	"dz^U1@\x8a69\xbe\x11\xd3\x9fB\xcd\x98" +
	"\x9ak\xe7M\x9e*\xf3&\xfb\x7f\x99\xfd\xb3\\\xbd\xd1" +
	"k\xb4\x19\xee\xcc\xd54Y\xed\x89\xbb]\xbe\x11\x05\xfe" +
	"h7\x17\x82\xeed\xa0\x09\xab\x91\xe7\x9f\x00\x00\x00\xff" +
	"\xff8\xaa\xe7q"

func RegisterSchema(reg *schemas.Registry) {
	reg.Register(&schemas.Schema{
		String: schema_9a51e53177277763,
		Nodes: []uint64{
			0xaf2e5ebaa58175d2,
			0xbb4f16b0a7d2d09b,
			0xc53168b273d497ee,
			0xd72ab4a0243047ac,
			0xda23f0d3a8250633,
			0xeea7ae19b02f5d47,
			0xf20b3dea95929312,
			0xf9694ae208dbb3e3,
		},
		Compressed: true,
	})
}
