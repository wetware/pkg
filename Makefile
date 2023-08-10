all: commands
	@go generate ./...

commands: defaultROM ls

defaultROM:
	@GOOS=wasip1 GOARCH=wasm gotip build -o rom/internal/main.wasm rom/internal/main.go

ls:
	@GOOS=wasip1 GOARCH=wasm gotip build -o rom/ls/internal/main.wasm rom/ls/internal/main.go
