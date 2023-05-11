package system

import (
	_ "embed"
	"encoding/hex"

	"lukechampine.com/blake3"
)

//go:embed internal/main.wasm
var defaultROM []byte

func DefaultROM() ROM {
	copied := make(ROM, len(defaultROM))
	copy(copied, defaultROM) // defensive copy
	return copied
}

type ROM []byte

func (bc ROM) Hash() [64]byte {
	return blake3.Sum512(bc)
}

// String returns the BLAKE3-512 hash of the ROM, truncated to the
// first 8 bytes.  It is intended as a human-readable symbol.
func (bc ROM) String() string {
	hash := bc.Hash()
	return hex.Dump(hash[:8])
}
