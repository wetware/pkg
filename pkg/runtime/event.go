package runtime

import (
	"fmt"
)

type (
	// EvtServiceStateChanged fires when a service changes its runtime state
	EvtServiceStateChanged struct {
		loggable
		State ServiceState
	}
)

// ServiceState .
type ServiceState uint8

const (
	// ServiceStateStarting indicates that a service has initiated its startup sequence.
	ServiceStateStarting = iota
	// ServiceStateStopping indicates that a service has initiated its shutdown sequence.
	ServiceStateStopping
)

func (s ServiceState) String() string {
	switch s {
	case ServiceStateStarting:
		return "service starting"
	case ServiceStateStopping:
		return "service stopping"
	default:
		return fmt.Sprintf("<invalid :: %d>", s)
	}
}

type loggable interface {
	Loggable() map[string]interface{}
}
