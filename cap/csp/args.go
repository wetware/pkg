package csp

import (
	"fmt"
	"strconv"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
)

const base = multibase.Base58BTC

type Args struct {
	Pid  uint32
	Ppid uint32
	Cid  cid.Cid
	// extra []string // todo
}

func (a Args) Encode() []string {
	return []string{
		strconv.FormatUint(uint64(a.Pid), 10),
		strconv.FormatUint(uint64(a.Ppid), 10),
		a.Cid.Encode(multibase.MustNewEncoder(base)), // TODO validate cid
	}
}

func (a *Args) Decode(v []string) error {
	if len(v) < 3 {
		return fmt.Errorf("args len %d < 3", len(v))
	}

	u, err := strconv.ParseUint(v[0], 10, 32)
	if err != nil {
		return err
	}
	a.Pid = uint32(u)

	u, err = strconv.ParseUint(v[1], 10, 32)
	if err != nil {
		return err
	}
	a.Ppid = uint32(u)

	a.Cid, err = cid.Decode(v[2])
	return err
}
