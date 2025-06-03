package system

import (
	"os"
	"strconv"

	"github.com/ipfs/go-cid"
)

// os.Args = {pid, ppid, cid, ...}
const (
	ipid = iota
	ippid
	icid
	istart
)

// cache responses as they be called many times.
var (
	cachedPid  *uint32
	cachedPpid *uint32
	cachedCid  *cid.Cid
)

// Args passed onto the process.
func Args() []string {

	if len(os.Args) < istart {
		return []string{}
	}

	return os.Args[istart:]
}

// Pid of the process.
func Pid() uint32 {
	if cachedPid == nil {
		u, err := strconv.ParseUint(os.Args[ipid], 10, 32)
		if err != nil {
			panic(err)
		}
		pid := uint32(u)
		cachedPid = &pid
	}
	return *cachedPid
}

// Pid of the parent process.
func Ppid() uint32 {
	if cachedPpid == nil {
		u, err := strconv.ParseUint(os.Args[ippid], 10, 32)
		if err != nil {
			panic(err)
		}
		ppid := uint32(u)
		cachedPpid = &ppid
	}
	return *cachedPpid
}

// CID of the bytecode the process is running.
func Cid() cid.Cid {
	if cachedCid == nil {
		c, err := cid.Decode(os.Args[icid])
		if err != nil {
			panic(err)
		}
		cachedCid = &c
	}
	return *cachedCid
}
