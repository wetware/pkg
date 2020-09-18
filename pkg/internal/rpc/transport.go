package rpc

import (
	"bytes"
	"context"
	"io"
	"time"

	capnp "zombiezen.com/go/capnproto2"
	"zombiezen.com/go/capnproto2/rpc"
	rpccapnp "zombiezen.com/go/capnproto2/std/capnp/rpc"
)

// CodecFactory specifies the capnp Encoder/Decoder pair to be used
type CodecFactory interface {
	NewEncoder(io.Writer) *capnp.Encoder
	NewDecoder(io.Reader) *capnp.Decoder
}

// BasicCodec (de)codes unpacked capnp messages
type BasicCodec struct{}

// NewEncoder .
func (BasicCodec) NewEncoder(w io.Writer) *capnp.Encoder { return capnp.NewEncoder(w) }

// NewDecoder .
func (BasicCodec) NewDecoder(r io.Reader) *capnp.Decoder { return capnp.NewDecoder(r) }

// PackedCodec (de)codes packed capnp messages
type PackedCodec struct{}

// NewEncoder .
func (PackedCodec) NewEncoder(w io.Writer) *capnp.Encoder { return capnp.NewPackedEncoder(w) }

// NewDecoder .
func (PackedCodec) NewDecoder(r io.Reader) *capnp.Decoder { return capnp.NewPackedDecoder(r) }

type streamTransport struct {
	rwc      io.ReadWriteCloser
	deadline writeDeadlineSetter

	enc  *capnp.Encoder
	dec  *capnp.Decoder
	wbuf bytes.Buffer
}

// StreamTransport creates a transport that sends and receives messages
// by serializing and deserializing unpacked Cap'n Proto messages.
// Closing the transport will close the underlying ReadWriteCloser.
func StreamTransport(f CodecFactory, rwc io.ReadWriteCloser) rpc.Transport {
	d, _ := rwc.(writeDeadlineSetter)
	s := &streamTransport{
		rwc:      rwc,
		deadline: d,
		dec:      f.NewDecoder(rwc),
	}
	s.wbuf.Grow(4096)
	s.enc = f.NewEncoder(&s.wbuf)
	return s
}

func (s *streamTransport) SendMessage(ctx context.Context, msg rpccapnp.Message) error {
	s.wbuf.Reset()
	if err := s.enc.Encode(msg.Segment().Message()); err != nil {
		return err
	}
	if s.deadline != nil {
		// TODO(light): log errors
		if d, ok := ctx.Deadline(); ok {
			s.deadline.SetWriteDeadline(d)
		} else {
			s.deadline.SetWriteDeadline(time.Time{})
		}
	}
	_, err := s.rwc.Write(s.wbuf.Bytes())
	return err
}

func (s *streamTransport) RecvMessage(ctx context.Context) (rpccapnp.Message, error) {
	var (
		msg *capnp.Message
		err error
	)
	read := make(chan struct{})
	go func() {
		msg, err = s.dec.Decode()
		close(read)
	}()
	select {
	case <-read:
	case <-ctx.Done():
		return rpccapnp.Message{}, ctx.Err()
	}
	if err != nil {
		return rpccapnp.Message{}, err
	}
	return rpccapnp.ReadRootMessage(msg)
}

func (s *streamTransport) Close() error {
	return s.rwc.Close()
}

type writeDeadlineSetter interface {
	SetWriteDeadline(t time.Time) error
}
