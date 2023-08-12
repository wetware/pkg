all: clean install

install:
	go generate ./...
	GOOS=wasip1 GOARCH=wasm gotip build -o rom/internal/main.wasm rom/internal/main.go
	go install ./cmd/...

clean:
	@rm -f $(GOPATH)/bin/ww
