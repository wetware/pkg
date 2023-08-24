package system

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero/api"
)

type segment struct {
	offset, length uint32
}

func (seg segment) String() string {
	return fmt.Sprintf("segment[offset=%d len=%d]",
		seg.offset,
		seg.length)
}

func (seg segment) LoadFrom(mem api.Memory) (types.Bytes, bool) {
	return mem.Read(seg.offset, seg.length)
}

func (seg segment) Format(w io.Writer) {
	fmt.Fprint(w, seg.String())
}

func (seg segment) FormatObject(w io.Writer, memory api.Memory, object []byte) {
	seg.LoadObject(memory, object).Format(w)
}

func (seg segment) FormatValue(w io.Writer, memory api.Memory, stack []uint64) {
	seg.LoadValue(memory, stack).Format(w)
}

func (seg segment) LoadObject(memory api.Memory, object []byte) segment {
	return segment{
		offset: binary.LittleEndian.Uint32(object[:4]),
		length: binary.LittleEndian.Uint32(object[4:]),
	}
}

func (seg segment) LoadValue(memory api.Memory, stack []uint64) segment {
	return segment{
		offset: api.DecodeU32(stack[0]),
		length: api.DecodeU32(stack[1]),
	}
}

func (seg segment) StoreObject(memory api.Memory, object []byte) {
	binary.LittleEndian.PutUint32(object[:4], seg.offset)
	binary.LittleEndian.PutUint32(object[4:], seg.length)
}

func (seg segment) ObjectSize() int {
	return 8
}

func (seg segment) ValueTypes() []api.ValueType {
	return []api.ValueType{
		api.ValueTypeI32,
		api.ValueTypeI32}
}
