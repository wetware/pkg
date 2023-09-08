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
	Cmd  []string
}

func (a Args) Encode() []string {
	return append([]string{
		strconv.FormatUint(uint64(a.Pid), 10),
		strconv.FormatUint(uint64(a.Ppid), 10),
		a.Cid.Encode(multibase.MustNewEncoder(base)), // TODO validate cid
	}, a.Cmd...)
}

func (a *Args) Decode(argv []string) error {
	if len(argv) < 3 {
		return fmt.Errorf("args len %d < 3", len(argv))
	}

	u, err := strconv.ParseUint(argv[0], 10, 32)
	if err != nil {
		return err
	}
	a.Pid = uint32(u)

	u, err = strconv.ParseUint(argv[1], 10, 32)
	if err != nil {
		return err
	}
	a.Ppid = uint32(u)

	a.Cid, err = cid.Decode(argv[2])

	a.Cmd = []string{}
	if len(argv) >= 3 {
		a.Cmd = append(a.Cmd, argv[3:]...)
	}

	return err
}
