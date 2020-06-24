// Package routingutil provides public utility functions for interacting with pkg/internal/routing
package routingutil

import (
	"github.com/lthibault/wetware/pkg/internal/routing"
)

// Heartbeat is a message that announces a host's liveliness in a cluster.
type Heartbeat = routing.Heartbeat

// MarshalHeartbeat serializes a heartbeat message.
func MarshalHeartbeat(h Heartbeat) ([]byte, error) {
	return routing.MarshalHeartbeat(h)
}

// UnmarshalHeartbeat reads a heartbeat message from bytes.
func UnmarshalHeartbeat(b []byte) (Heartbeat, error) {
	return routing.UnmarshalHeartbeat(b)
}
