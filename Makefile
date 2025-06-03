.PHONY: all build build-arm clean gen install

FLAGS := CGO_ENABLED=0
ARM_FLAGS := ${FLAGS} GOOS=linux GOARCH=arm GOARM=7
WASM_FLAGS := ${FLAGS} GOOS=wasip1 GOARCH=wasm

all: clean install

gen:
	go generate ./...
	env ${WASM_FLAGS} go build -o rom/internal/main.wasm rom/internal/main.go

build: gen
	env ${FLAGS} go build ./cmd/ww

build-arm:
	GOOS=linux GOARCH=arm GOARM=7
	env ${ARM_FLAGS}  go build -o ww-arm ./cmd/ww

install: gen
	env ${FLAGS} go install ./cmd/...

clean:
	@rm -f $(GOPATH)/bin/ww
	@rm -f api/*/*.go
	@rm -f test/**/*.wasm
