package logutil

import (
	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
)

func NewEventHook(log log.Logger) suture.EventHook {
	return func(e suture.Event) {
		switch e.Type() {
		case suture.EventTypeServicePanic:
			log.WithFields(e.Map()).Error("service panicked")

		case suture.EventTypeBackoff:
			log.WithFields(e.Map()).Debug("entered backoff state")

		case suture.EventTypeResume:
			log.WithFields(e.Map()).Debug("resumed")

		case suture.EventTypeServiceTerminate:
			log.WithFields(e.Map()).Warn("service terminated")
		}
	}
}
