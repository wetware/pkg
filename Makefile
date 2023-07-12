# Set a sensible default for the $GOPATH in case it's not exported.
# If you're seeing path errors, try exporting your GOPATH.
ifeq ($(origin GOPATH), undefined)
    GOPATH := $(HOME)/Go
endif

experiment_dir := experiments/build-capnp

all: clean capnp mocks experiment


clean: clean-capnp clean-mocks experiment-clean-capnp


capnp: experiment_dir_setup experiment-capnp capnp-anchor capnp-pubsub capnp-cluster capnp-channel capnp-process capnp-registry capnp-bitswap experiment_dir_teardown
# N.B.:  compiling capnp schemas requires having capnproto.org/go/capnp/v3 installed
#        on the GOPATH.

capnp-anchor:
	@mkdir -p api/anchor
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:api/anchor --src-prefix=api api/anchor.capnp

capnp-pubsub:
	@mkdir -p api/pubsub
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:api/pubsub --src-prefix=api api/pubsub.capnp

capnp-cluster:
	@mkdir -p api/cluster
	@capnp compile -I$(experiment_dir) -ogo:api/cluster --src-prefix=api api/cluster.capnp

capnp-channel:
	@mkdir -p api/channel
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:api/channel --src-prefix=api api/channel.capnp

capnp-process:
	@mkdir -p api/process
	@capnp compile -I$(experiment_dir) -ogo:api/process --src-prefix=api api/process.capnp

capnp-registry:
	@mkdir -p api/registry
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:api/registry --src-prefix=api api/registry.capnp

capnp-bitswap:
	@mkdir -p api/bitswap
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/std -ogo:api/bitswap --src-prefix=api api/bitswap.capnp

clean-capnp: clean-capnp-anchor clean-capnp-pubsub clean-capnp-cluster clean-capnp-channel clean-capnp-process clean-capnp-registry clean-capnp-bitswap

clean-capnp-anchor:
	@rm -rf api/anchor

clean-capnp-pubsub:
	@rm -rf api/pubsub

clean-capnp-cluster:
	@rm -rf api/cluster

clean-capnp-channel:
	@rm -rf api/channel

clean-capnp-process:
	@rm -rf api/process

clean-capnp-registry:
	@rm -rf api/registry

clean-capnp-bitswap:
	@rm -rf api/bitswap

mocks: clean-mocks
# This roundabout call to 'go generate' allows us to:
#  - use modules
#  - prevent grep missing (totally fine) from causing nonzero exit
#  - mirror the pkg/ structure under internal/test/mock
	@find . -name '*.go' | xargs -I{} grep -l '//go:generate' {} | xargs -I{} -P 10 go generate {}


clean-mocks:
	@find . -name 'mock_*.go' | xargs -I{} rm {}

experiment: experiment-clean-capnp experiment-capnp

experiment_dir_setup:
	@mkdir -p $(experiment_dir)/experiments
	@cp -r $(GOPATH)/src/capnproto.org/go/capnp/v3/std/* $(experiment_dir)
	@cp experiments/api/*.capnp $(experiment_dir)/experiments

experiment_dir_teardown:
	@rm -rf $(experiment_dir)

experiment-capnp: experiment-capnp-tools experiment-capnp-http 

experiment-capnp-tools:
	@mkdir -p experiments/api/tools
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/v3/std -ogo:experiments/api/tools --src-prefix=experiments/api experiments/api/tools.capnp

experiment-capnp-http:
	@mkdir -p experiments/api/http
	@capnp compile -I$(GOPATH)/src/capnproto.org/go/capnp/v3/std -ogo:experiments/api/http --src-prefix=experiments/api experiments/api/http.capnp

experiment-clean-capnp: experiment-clean-capnp-tools experiment-clean-capnp-http

experiment-clean-capnp-tools:
	@rm -rf experiments/api/tools

experiment-clean-capnp-http:
	@rm -rf experiments/api/http
	