all: clean install

FLAGS := CGO_ENABLED=0

gen:
	go generate ./...
	GOOS=wasip1 GOARCH=wasm go build -o rom/internal/main.wasm rom/internal/main.go
	GOOS=wasip1 GOARCH=wasm go build -o system/testdata/main.wasm system/testdata/main.go

build: gen
	env ${FLAGS} go build ./cmd/ww

install: gen
	env ${FLAGS} go install ./cmd/...

clean:
	@rm -f $(GOPATH)/bin/ww
	@rm -f api/*/*.go
	@rm -f test/**/*.wasm
