package socket

import (
	"errors"
	"net"

	"github.com/lthibault/log"
)

var (
	// ErrIgnore causes a message to be dropped silently.
	// It is typically used when filtering out messgaes that
	// originate from the local host.
	ErrIgnore = errors.New("ignore")

	// ErrClosed is returned when performing operations against
	// a closed socket.
	ErrClosed = errors.New("closed")
)

// ProtocolError signals a non-fatal error caused either either
// by a malformed *record.Envelope, or by a *Record containing
// unexpected values.
//
// The default error callback will log a protocol error at the
// DEBUG level, using 'Message' as the logging message and the
// 'Meta' field as a set of structured logging fields.  If the
// 'Cause' field is non-nil, it will be added to 'Meta' before
// writing the log message.
//
// User-supplied error handlers SHOULD test for ProtocolError
// via type-assertion and treat any instances as a non-fatal
// error.
type ProtocolError struct {
	Message string
	Cause   error
	Meta    log.F
}

func (pe ProtocolError) Loggable() map[string]interface{} {
	if pe.Cause != nil {
		pe.Meta["error"] = pe.Cause
	}

	return pe.Meta
}

func (pe ProtocolError) Error() string {
	return pe.Cause.Error()
}

func (pe ProtocolError) Is(err error) bool {
	return errors.Is(err, pe.Cause)
}

func (pe ProtocolError) Unwrap() error {
	return pe.Cause
}

// ValidationError signals that a packet contains the expected
// data, but that its authenticity and provenance could not be
// proven.
//
// The default error callback will log validation errors at the
// DEBUG level.  In high-security environments, it MAY be
// adviseable to log such events at the WARN level, and to take
// further action.
type ValidationError struct {
	Cause error
	From  net.Addr
}

func (ve ValidationError) Loggable() map[string]interface{} {
	return log.F{
		"error": ve.Cause,
		"from":  ve.From,
	}
}

func (ve ValidationError) Error() string {
	return ve.Cause.Error()
}

func (ve ValidationError) Is(err error) bool {
	return errors.Is(err, ve.Cause)
}

func (ve ValidationError) Unwrap() error {
	return ve.Cause
}
