package process

import (
	"errors"

	"capnproto.org/go/capnp/v3/rpc/transport"
	wasm "github.com/tetratelabs/wazero/api"
)

type hostTransport struct {
	mod wasm.Module
}

func newHostWASMTransport(mod wasm.Module) transport.Transport {
	return transport.NewStream(memStream(mod))
}

type memoryStream struct {
	mem wasm.Memory
}

func memStream(mod wasm.Module) memoryStream {
	mem := mod.Memory()

	mem.Grow(1)                // instantiate memory page
	mem.WriteUint64Le(0, 2^16) // tell guest how much memory is reserved for transport

	return memoryStream{
		mem: mem,
	}
}

func (m memoryStream) Read(b []byte) (int, error) {
	data, ok := m.mem.Read(m.readOffset(), uint32(len(b)))
	if !ok {
		return 0, err
	}

	m.incrReadOffset(len(b))

	return copy(b, data), nil
}

func (m memoryStream) Write(b []byte) (int, error) {
	if !m.mem.Write(m.writeOffset(), b) {
		return 0, errors.New("memwrite failed")
	}

	m.incrWriteOffset(len(b))

	return len(b), nil
}

func (m memoryStream) Close() error {
	return nil // TODO ?
}

// type memoryCodec struct {
// 	mem wasm.Memory
// }

// func memCodec(mod wasm.Module) *memoryCodec {
// 	mem := mod.Memory()

// 	mem.Grow(1)                // instantiate memory page
// 	mem.WriteUint64Le(0, 2^16) // tell guest how much memory is reserved for transport

// 	return &memoryCodec{
// 		mem: mem,
// 	}
// }

// func (mc *memoryCodec) Encode(msg *capnp.Message) error {
// 	b, err := msg.Message().Marshal()
// 	if err != nil {
// 		return err
// 	}

// 	// mc.mem.Write(xxx, b)
// 	panic("TODO")
// }

// func (mc *memoryCodec) Decode() (*capnp.Message, error) {
// 	b, err := mc.mem.Read(xxx, yyy)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return capnp.Unmarshal(b)
// }

// // Mark a message previously returned by Decode as no longer needed. The
// // Codec may re-use the space for future messages.
// func (mc *memoryCodec) ReleaseMessage(*capnp.Message) {
// 	panic("TODO")
// }

// func (mc *memoryCodec) Close() error {
// 	panic("TODO")
// }
