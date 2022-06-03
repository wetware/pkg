package io

import (
	"context"
	"errors"
	"io"
	"net"
	"os"

	api "github.com/wetware/ww/internal/api/io"
)

type ErrorCode = api.Error_Code

type Error struct {
	Cause error
}

func (ioerr Error) Is(err error) bool {
	if err == nil && ioerr.Cause == nil {
		return true
	}

	for _, target := range []error{
		io.EOF,
		io.ErrUnexpectedEOF,
		net.ErrClosed,
		context.Canceled,
		context.DeadlineExceeded,
		io.ErrNoProgress,
		io.ErrShortWrite,
		io.ErrShortBuffer,
		os.ErrDeadlineExceeded,
	} {
		if errors.Is(err, target) {
			return true
		}
	}

	// matches a known error code other than 'nil'?
	if (Error{Cause: err}).Code() != 0 {
		return true
	}

	// last-ditch attempt; string-matching
	return ioerr.Cause.Error() == err.Error()
}

func (ioerr Error) Error() string {
	return ioerr.Cause.Error()
}

func (ioerr Error) Code() ErrorCode {
	if ioerr.Cause == nil {
		return 0
	}

	switch err := ioerr.Cause; err {
	case io.ErrShortWrite:
		return api.Error_Code_shortWrite

	case io.ErrShortBuffer:
		return api.Error_Code_shortBuf

	case io.EOF:
		return api.Error_Code_eof

	case io.ErrUnexpectedEOF:
		return api.Error_Code_unexpectedEOF

	case io.ErrNoProgress:
		return api.Error_Code_noProgress

	case net.ErrClosed:
		return api.Error_Code_closed

	case context.Canceled:
		return api.Error_Code_canceled

	case context.DeadlineExceeded:
		return api.Error_Code_deadlineExceeded

	case os.ErrDeadlineExceeded:
		return api.Error_Code_deadlineExceeded

	default:
		// Try string matching
		return api.Error_CodeFromString(err.Error())
	}
}

type ioError api.Error

func (ioerr ioError) Set(err error) error {
	e := api.Error(ioerr)
	e.SetCode(Error{Cause: err}.Code())

	if e.Code() == api.Error_Code_nil {
		err = e.SetMessage(Error{Cause: err}.Error())
	}

	return err
}

func (ioerr ioError) Err() error {
	if err := ioerr.parseErr(); err != nil {
		return Error{Cause: err}
	}

	return nil
}

func (ioerr ioError) parseErr() error {
	var e = api.Error(ioerr)

	// Recognized error code?
	if err := errorFromCode(e.Code()); err != nil {
		return err
	}

	// No error?
	if !e.HasMessage() {
		return nil
	}

	// Unspecified error; Get the error message.
	msg, err := api.Error(ioerr).Message()
	if err != nil {
		return err // is a capnp Exception type
	}

	// Last ditch attempt to parse this into something
	// we recognize...
	code := api.Error_CodeFromString(msg)
	if err = errorFromCode(code); err != nil {
		return err
	}

	return errors.New(msg)
}

// Returns nil if code == 0, or is unrecognized.
func errorFromCode(code api.Error_Code) error {
	switch code {
	case api.Error_Code_nil:
		break

	case api.Error_Code_shortWrite:
		return io.ErrShortWrite

	case api.Error_Code_shortBuf:
		return io.ErrShortBuffer

	case api.Error_Code_eof:
		return io.EOF

	case api.Error_Code_unexpectedEOF:
		return io.ErrUnexpectedEOF

	case api.Error_Code_noProgress:
		return io.ErrNoProgress

	case api.Error_Code_closed:
		return net.ErrClosed

	case api.Error_Code_canceled:
		return context.Canceled

	case api.Error_Code_deadlineExceeded:
		return context.DeadlineExceeded
	}

	return nil
}
