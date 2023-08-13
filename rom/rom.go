package rom

import (
	"bytes"
	_ "embed"
	"fmt"

	_ "embed"
	"io"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"lukechampine.com/blake3"
)

const (
	Codec = 2020
)

//go:embed internal/main.wasm
var defaultROM []byte

func Default() ROM {
	rom, err := Read(bytes.NewReader(defaultROM))
	if err != nil {
		panic(fmt.Errorf("read rom: %w", err))
	}

	return rom
}

// ROM is an immutable, read-only memory segment containing WASM
// bytecode.  It is uniquely identified by its hash.
type ROM struct {
	Bytecode []byte
}

func Read(r io.Reader) (rom ROM, err error) {
	rom.Bytecode, err = io.ReadAll(r)
	return
}

func (rom ROM) Hash() []byte {
	// TODO:  compute hash only once, using sync.Once
	hash := blake3.Sum512(rom.Bytecode)
	encoded, _ := multihash.Encode(hash[:], multihash.BLAKE3) // err always nil
	return encoded
}

func (rom ROM) CID() cid.Cid {
	return cid.NewCidV1(Codec, rom.Hash())
}

// String returns the rom as a string-formatted CID.
func (rom ROM) String() string {
	return rom.CID().String()
}
