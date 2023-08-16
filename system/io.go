package system

import (
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero/api"
)

type ioStat struct {
	types.Uint32
}

func (arg ioStat) LoadObject(memory api.Memory, object []byte) ioStat {
	return ioStat{arg.Uint32.LoadObject(memory, object)}
}

func (arg ioStat) LoadValue(memory api.Memory, stack []uint64) ioStat {
	return ioStat{arg.Uint32.LoadValue(memory, stack)}
}
