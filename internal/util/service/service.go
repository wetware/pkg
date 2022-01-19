package serviceutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
)

func New(log log.Logger, name string) *suture.Supervisor {
	return suture.New(name, suture.Spec{
		EventHook: NewEventHook(log, name),
	})
}

func NewEventHook(log log.Logger, name string) suture.EventHook {
	return func(e suture.Event) {
		switch ev := e.(type) {
		case suture.EventBackoff:
			log.WithFields(ev.Map()).Debugf("%s suspended", ev.SupervisorName)

		case suture.EventResume:
			log.
				WithField("parent", ev.SupervisorName).
				Infof("%s resumed", ev.SupervisorName)

		case suture.EventServiceTerminate:
			log.With(Exception{
				Value:        ev.Err,
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
			}).
				Warnf("caught exception in %s", ev.ServiceName)

		case suture.EventServicePanic:
			log.With(Exception{
				Value:        name,
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
			}).
				Warnf("unhandled exception in %s", ev.ServiceName)

			fmt.Fprintf(os.Stdout, "%s\n%s\n",
				ev.PanicMsg,
				ev.Stacktrace)

		case suture.EventStopTimeout:
			log.
				WithField("parent", ev.SupervisorName).
				Fatalf("%s failed to stop in a timely manner", ev.ServiceName)
		}
	}
}

// Exception is thrown asynchronously from services.
type Exception struct {
	Value        interface{} `json:"value" cbor:"value"`
	Parent       string      `json:"parent" cbor:"parent"`
	Restart      bool        `json:"restart" cbor:"restart"`
	Backpressure float64     `json:"backpressure" cbor:"backpressure"`
}

func (e Exception) GoString() string {
	return fmt.Sprintf(strings.TrimSpace(`
Exception{
	Value:       "%#v",
	Parent:      "%s",
	Restart:      %t,
	Backpressure: %.2f,
}`),
		e.Value,
		strconv.Quote(e.Parent),
		e.Restart,
		e.Backpressure)
}

func (e Exception) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"value":        e.Value,
		"parent":       e.Parent,
		"restart":      e.Restart,
		"backpressure": e.Backpressure,
	}
}
