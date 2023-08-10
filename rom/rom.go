package rom

import (
	"bytes"
	_ "embed"
	"fmt"

	ww "github.com/wetware/pkg"
)

//go:embed internal/main.wasm
var defaultROM []byte

func Default() ww.ROM {
	rom, err := ww.Read(bytes.NewReader(defaultROM))
	if err != nil {
		panic(fmt.Errorf("read rom: %w", err))
	}

	return rom
}
