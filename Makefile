# N.B.  make sure that your GOPATH is exported or that you've run `make GOPATH=$GOPATH`.

all: capnp

clean:
	@rm -f internal/api/*.capnp.go

capnp: clean
	@capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:internal api/api.capnp

cleanmocks:
	@find . -name 'mock_*.go' | xargs -I{} rm {}

mocks: cleanmocks
	# This roundabout call to 'go generate' allows us to:
	# 	- use modules
	# 	- prevent grep missing (totally fine) from causing nonzero exit
	#   - mirror the pkg/ structure under internal/test/mock
	@find . -name '*.go' | xargs -I{} grep -l '//go:generate' {} | xargs -I{} -P 10 go generate {}