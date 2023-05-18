# Wetware Developer's Guide

## How to build & Run

```bash
make && go run cmd/ww/main.go
```

The `make` command uses `go:generate` to call `tingo`.  See the `//go:generate` directive at the top of `system/internal/main.go` to see which command is run.

`cmd/ww/main.go` is the entrypoint for the `ww` command-line interface.  It's not very interesting.

## Where to start Reading

@lthibault recommends starting with `ww.go` in the repository's root directory.  In particular, the `Ww.Exec` function sets up a wasm runtime, loads a blob of wasm bytecode (called a "ROM"), and executes it.  When executing, the guest and host commonicate over a Cap'n Proto `rpc.Conn`.  The host connection is set up in `Ww.Exec`.  The guest is set up in `system/internal/main.go`.
