# Set a sensible default for the $GOPATH in case it's not exported.
# If you're seeing path errors, try exporting your GOPATH.
ifeq ($(origin GOPATH), undefined)
    GOPATH := $(HOME)/Go
endif

all: capnp


clean: clean-capnp clean-mocks


capnp: capnp-anchor capnp-pubsub capnp-cluster capnp-channel capnp-proc
# N.B.:  compiling capnp schemas requires having capnproto.org/go/capnp/v3 installed
#        on the GOPATH.

capnp-anchor:
	@mkdir -p internal/api/anchor
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/anchor --src-prefix=api api/anchor.capnp

capnp-pubsub:
	@mkdir -p internal/api/pubsub
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/pubsub --src-prefix=api api/pubsub.capnp

capnp-cluster:
	@mkdir -p internal/api/cluster
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/cluster --src-prefix=api api/cluster.capnp

capnp-proc:
	@mkdir -p internal/api/proc
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/proc --src-prefix=api api/proc.capnp

capnp-channel:
	@mkdir -p internal/api/channel
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:internal/api/channel --src-prefix=api api/channel.capnp

clean-capnp: clean-capnp-anchor clean-capnp-pubsub clean-capnp-cluster clean-capnp-channel clean-capnp-proc

clean-capnp-anchor:
	@rm -rf internal/api/anchor

clean-capnp-pubsub:
	@rm -rf internal/api/pubsub

clean-capnp-cluster:
	@rm -rf internal/api/cluster

clean-capnp-proc:
	@rm -rf internal/api/proc

clean-capnp-channel:
	@rm -rf internal/api/channel


mocks: clean-mocks
# This roundabout call to 'go generate' allows us to:
#  - use modules
#  - prevent grep missing (totally fine) from causing nonzero exit
#  - mirror the pkg/ structure under internal/test/mock
	@find . -name '*.go' | xargs -I{} grep -l '//go:generate' {} | xargs -I{} -P 10 go generate {}


clean-mocks:
	@find . -name 'mock_*.go' | xargs -I{} rm {}