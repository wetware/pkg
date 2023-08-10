package ww

import (
	_ "embed"
	"encoding/hex"
	"io"

	"lukechampine.com/blake3"
)

// ROM is an immutable, read-only memory segment containing WASM
// bytecode.  It is uniquely identified by its hash.
type ROM struct {
	bytecode []byte
}

func Read(r io.Reader) (rom ROM, err error) {
	rom.bytecode, err = io.ReadAll(r)
	return
}

func (rom ROM) Hash() [64]byte {
	return blake3.Sum512(rom.bytecode)
}

// String returns the BLAKE3-512 hash of the ROM, truncated to the
// first 8 bytes.  It is intended as a human-readable symbol.  Use
// the Hash() method to verify integrity.
func (rom ROM) String() string {
	hash := rom.Hash()
	return hex.Dump(hash[:8])
}
