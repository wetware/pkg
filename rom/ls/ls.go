package ls

import (
	"bytes"
	_ "embed"
	"fmt"

	ww "github.com/wetware/pkg"
)

//go:embed internal/main.wasm
var rom []byte

func ROM() ww.ROM {
	rom, err := ww.Read(bytes.NewReader(rom))
	if err != nil {
		panic(fmt.Errorf("read rom: %w", err))
	}

	return rom
}
