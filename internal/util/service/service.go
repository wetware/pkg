package serviceutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
)

func New(c *cli.Context, log log.Logger) *suture.Supervisor {
	return suture.New(c.App.Name, suture.Spec{
		EventHook: NewEventHook(log, c.App),
	})
}

func NewEventHook(logger log.Logger, app *cli.App) suture.EventHook {
	return func(e suture.Event) {
		switch ev := e.(type) {
		case suture.EventBackoff:
			logger.WithFields(ev.Map()).Debugf("%s suspended", ev.SupervisorName)

		case suture.EventResume:
			logger.
				WithField("parent", ev.SupervisorName).
				Infof("%s resumed", ev.SupervisorName)

		case suture.EventServiceTerminate:
			logger.With(Exception{
				Value:        ev.Err,
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
			}).
				Warnf("encountered exception in %s", ev.ServiceName)

		case suture.EventServicePanic:
			logger.With(Exception{
				Value:        app.Metadata,
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
			}).
				Warnf("unhandled exception in %s", ev.ServiceName)

			fmt.Fprintf(app.Writer, "%s\n%s\n",
				ev.PanicMsg,
				ev.Stacktrace)

		case suture.EventStopTimeout:
			logger.
				WithField("parent", ev.SupervisorName).
				Fatal("%w encountered a fatal error during restart", ev.ServiceName)
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
