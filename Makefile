all: defaultROM
	@go generate ./...

defaultROM:
	@GOOS=wasip1 GOARCH=wasm gotip build -o rom/internal/main.wasm rom/internal/main.go
