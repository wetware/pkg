# Set a sensible default for the $GOPATH in case it's not exported.
# If you're seeing path errors, try exporting your GOPATH.
ifeq ($(origin GOPATH), undefined)
    GOPATH := $(HOME)/Go
endif

all: capnp


clean: clean-capnp clean-mocks


capnp: capnp-pubsub
# N.B.:  compiling capnp schemas requires having capnproto.org/go/capnp/v3 installed
#        on the GOPATH.


capnp-pubsub:
	@mkdir -p internal/api/pubsub
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/pubsub --src-prefix=api api/pubsub.capnp


clean-capnp: clean-capnp-pubsub


clean-capnp-pubsub:
	@rm -rf internal/api/pubsub


mocks: clean-mocks
# This roundabout call to 'go generate' allows us to:
#  - use modules
#  - prevent grep missing (totally fine) from causing nonzero exit
#  - mirror the pkg/ structure under internal/test/mock
	@find . -name '*.go' | xargs -I{} grep -l '//go:generate' {} | xargs -I{} -P 10 go generate {}


clean-mocks:
	@find . -name 'mock_*.go' | xargs -I{} rm {}