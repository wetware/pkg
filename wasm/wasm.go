package ww

/*
 * The contents of this file will be moved to the ww repository
 */

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	csp "github.com/wetware/ww/pkg/csp"
)

// Default file descriptor for Wazero pre-openned TCP connections
const (
	// file descriptor for pre-openned TCP socket
	PREOPENED_FD = 3

	// Inbox in which each element will be found by default on the inbox
	ARGS_INDEX       = 0
	CAPS_START_INDEX = 1

	// Number of default arguments
	RESERVED_ARG_N = 4
	// Argument order
	ARG_PID      = 0 // PID of the process
	ARG_MD5      = 1 // md5 sum of the process, used to self-replicate
	ARG_PROC_KEY = 2 // private key of the process
	ARG_EXEC_KEY = 3 // public key of the process
)

// Self contains the info a WASM process will need for:
type Self struct {
	Args       []string          // Receiving parameters.
	Caps       []capnp.Client    // Communication.
	Closers    io.Closer         // Cleaning up.
	Md5Sum     []byte            // Self-replicating.
	Pid        uint32            // Indetifying self.
	PrvKey     crypto.PrivateKey // Verifying self.
	ExecPubKey crypto.PublicKey  // Verifying parent executor.

	// cached signature
	signature []byte
}

func (s *Self) Close() error {
	return s.Closers.Close()
}

// Signature returns the signed pid after converting it from uint32 to []byte
func (s *Self) Signature() []byte {
	// signature was cached
	if s.signature != nil {
		return s.signature
	}

	pidAsData := make([]byte, 4)
	binary.LittleEndian.PutUint32(pidAsData, s.Pid)

	prv, _ := s.PrvKey.(*rsa.PrivateKey)
	signature, _ := prv.Sign(rand.Reader, pidAsData, crypto.SHA256)

	// cache signature
	s.signature = signature

	return signature
}

// EncryptedSignature returns s.Signature() after encrypting it with the provided public key.
func (s *Self) EncryptedSignature(pubKey crypto.PublicKey) ([]byte, error) {
	pub, _ := pubKey.(*rsa.PublicKey)
	hash := sha512.New()

	return rsa.EncryptOAEP(hash, rand.Reader, pub, s.Signature(), nil)
}

// closer contains a slice of Closers that will be closed when this type itself is closed
type closer struct {
	closers []io.Closer
}

func (c closer) Close() error {
	for _, closer := range c.closers {
		defer closer.Close()
	}
	return nil
}

// add a new closer to the list
func (c closer) add(closer io.Closer) {
	c.closers = append(c.closers, closer)
}

// return the a TCP listener from pre-opened tcp connection by using the fd
func preopenedListener(c closer) (net.Listener, error) {
	f := os.NewFile(uintptr(PREOPENED_FD), "")

	if err := syscall.SetNonblock(PREOPENED_FD, false); err != nil {
		return nil, err
	}

	c.add(f)

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	c.add(l)

	return l, err
}

// BootstrapClient bootstraps and resolves the Capnp client attached
// to the other end of the pre-openned TCP connection
func BootstrapClient(ctx context.Context) (capnp.Client, io.Closer, error) {
	closer := closer{
		closers: make([]io.Closer, 0),
	}

	l, err := preopenedListener(closer)
	if err != nil {
		return capnp.Client{}, closer, err
	}

	tcpConn, err := l.Accept()
	if err != nil {
		return capnp.Client{}, closer, err
	}

	closer.add(tcpConn)

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	closer.add(conn)

	client := conn.Bootstrap(ctx)

	err = client.Resolve(ctx)

	return client, closer, err
}

// OpenInbox may be called whenever a process starts. It loads and resolves
// any capabilities left by call that created the process. Not required if
// Init was called.
func OpenInbox(ctx context.Context) ([]capnp.Client, io.Closer, error) {
	inbox, closer, err := BootstrapClient(ctx)
	if err != nil {
		return nil, closer, err
	}

	if err := inbox.Resolve(context.Background()); err != nil {
		return nil, closer, err
	}

	clients, err := csp.Inbox(inbox).Open(context.TODO())

	return clients, closer, err
}

func Init(ctx context.Context) (Self, error) {
	clients, closers, err := OpenInbox(ctx)
	if err != nil {
		return Self{}, err
	}
	args, err := csp.Args(clients[ARGS_INDEX]).Args(ctx)
	if err != nil {
		return Self{}, err
	}
	pid64, err := strconv.ParseUint(args[ARG_PID], 10, 32)
	if err != nil {
		return Self{}, err
	}
	md5sum := args[ARG_MD5]
	prvPem := args[ARG_PROC_KEY]
	pubPem := args[ARG_EXEC_KEY]

	prvKey, err := DecodePrvPEM([]byte(prvPem))
	if err != nil {
		return Self{}, err
	}
	pubKey, err := DecodePubPEM([]byte(pubPem))
	if err != nil {
		return Self{}, err
	}

	trimmedArgs := make([]string, len(args)-RESERVED_ARG_N)
	for i := 0; i < len(trimmedArgs); i++ {
		trimmedArgs[i] = args[i+RESERVED_ARG_N]
	}

	return Self{
		Args:       trimmedArgs,
		Caps:       clients[1:],
		Closers:    closers,
		Md5Sum:     []byte(md5sum),
		Pid:        uint32(pid64),
		PrvKey:     prvKey,
		ExecPubKey: pubKey,
	}, nil
}

// errLogger panics when an error occurs
type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		panic(err)
	}
}

// Extract a private key form a PEM certificate
func DecodePrvPEM(prvPEM []byte) (crypto.PrivateKey, error) {
	block, rest := pem.Decode(prvPEM)
	if rest != nil {
		return nil, errors.New("failed to extract private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// Extract a public key from a PEM certificate
func DecodePubPEM(pubPEM []byte) (crypto.PrivateKey, error) {
	block, rest := pem.Decode(pubPEM)
	if rest != nil {
		return nil, errors.New("failed to extract private key")
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}
