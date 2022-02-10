package serviceutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	logutil "github.com/wetware/ww/internal/util/log"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
)

func NewEventHook(c *cli.Context) suture.EventHook {
	return func(e suture.Event) {
		switch ev := e.(type) {
		case suture.EventBackoff:
			logutil.New(c).
				WithFields(ev.Map()).
				Debugf("%s suspended", ev.SupervisorName)

		case suture.EventResume:
			logutil.New(c).
				WithField("parent", ev.SupervisorName).
				Infof("%s resumed", ev.SupervisorName)

		case suture.EventServiceTerminate:
			logutil.New(c).
				With(Exception{
					Value:        ev.Err,
					Parent:       ev.SupervisorName,
					Restart:      ev.Restarting,
					Backpressure: ev.CurrentFailures / ev.FailureThreshold,
					Service:      ev.ServiceName,
				}).Warn("caught exception")

			bucket := fmt.Sprintf("%s.%s.", ev.SupervisorName, ev.ServiceName)
			statsdutil.Must(c).Increment(bucket + "restarts")
			if ev.Err != nil {
				statsdutil.Must(c).Increment(bucket + "errors")
			}

		case suture.EventServicePanic:
			logutil.New(c).
				With(Exception{
					Value:        fmt.Errorf(ev.PanicMsg),
					Parent:       ev.SupervisorName,
					Restart:      ev.Restarting,
					Backpressure: ev.CurrentFailures / ev.FailureThreshold,
					Service:      ev.ServiceName,
				}).Warn("unhandled exception")

			bucket := fmt.Sprintf("%s.%s.", ev.SupervisorName, ev.ServiceName)
			statsdutil.Must(c).Increment(bucket + "restarts")
			statsdutil.Must(c).Increment(bucket + "panics")

			// Print to stdout to avoid interferring with log
			// collection daemons.
			fmt.Fprintf(os.Stdout, "%s\n%s\n",
				ev.PanicMsg,
				ev.Stacktrace)

		case suture.EventStopTimeout:
			logutil.New(c).
				WithField("parent", ev.SupervisorName).
				WithField("service", ev.ServiceName).
				Fatal("failed to stop in a timely manner")
		}
	}
}

// Exception is thrown asynchronously from services.
type Exception struct {
	Value        interface{} `json:"value" cbor:"value"`
	Parent       string      `json:"parent" cbor:"parent"`
	Restart      bool        `json:"restart" cbor:"restart"`
	Backpressure float64     `json:"backpressure" cbor:"backpressure"`
	Service      string      `json:"service" cbor:"service"`
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
