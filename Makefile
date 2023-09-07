all: clean install

gen:
	go generate ./...
	GOOS=wasip1 GOARCH=wasm go build -o rom/internal/main.wasm rom/internal/main.go

build: gen
	go build ./cmd/ww

install: gen
	go install ./cmd/...

clean:
	@rm -f $(GOPATH)/bin/ww
	@rm -f api/*/*.go
	@rm -f test/**/*.wasm
