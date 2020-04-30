# N.B.  make sure that your GOPATH is exported or that you've run `make GOPATH=$GOPATH`.

all: capnp

clean:
	@rm -f internal/api/*.capnp.go

capnp: clean
	@capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:internal api/heartbeat.capnp
	@capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:internal api/anchor.capnp

