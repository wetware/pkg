FROM golang:1.21-alpine
RUN apk add --no-cache git make capnproto-dev libc6-compat gcompat htop
RUN go install github.com/golang/mock/mockgen@v1.6.0
RUN go install capnproto.org/go/capnp/v3/capnpc-go@latest
RUN mkdir -p /go/src/capnproto.org/go/capnp && \
    git clone https://github.com/capnproto/go-capnp /go/src/capnproto.org/go/capnp