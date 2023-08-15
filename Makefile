all: clean deps bin

clean:
	@rm -f $(GOPATH)/bin/ww

deps:
	go generate ./...
	GOOS=wasip1 GOARCH=wasm gotip build -o rom/internal/main.wasm rom/internal/main.go

bin:
	@go install ./cmd/...

ready: kill bin
	@go install ./cmd/...
	ww start

kill:
	@pkill -9 ww || true
