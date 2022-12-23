package serviceutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lthibault/log"

	"github.com/thejerf/suture/v4"
)

type Metrics interface {
	Incr(string)
	Duration(string, time.Duration)
}

func New(log log.Logger, m Metrics) suture.EventHook {
	metric := &metrics{Metrics: m}

	return func(e suture.Event) {
		e.Type()
		switch ev := e.(type) {
		case suture.EventBackoff:
			log.WithFields(ev.Map()).
				Debugf("%s suspended", ev.SupervisorName)
			metric.OnBackoff(ev)

		case suture.EventResume:
			log.WithField("parent", ev.SupervisorName).
				Infof("%s resumed", ev.SupervisorName)
			metric.OnResume(ev)

		case suture.EventServiceTerminate:
			log.With(Exception{
				Value:        ev.Err,
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
				Service:      ev.ServiceName,
			}).Warn("caught exception")
			metric.OnTerm(ev)

		case suture.EventServicePanic:
			log.With(Exception{
				Value:        fmt.Errorf(ev.PanicMsg),
				Parent:       ev.SupervisorName,
				Restart:      ev.Restarting,
				Backpressure: ev.CurrentFailures / ev.FailureThreshold,
				Service:      ev.ServiceName,
			}).Warn("unhandled exception")
			metric.OnPanic(ev)

			// Print to stdout to avoid interfering with log
			// collection daemons.
			fmt.Fprintf(os.Stdout, "%s\n%s\n",
				ev.PanicMsg,
				ev.Stacktrace)

		case suture.EventStopTimeout:
			log.WithField("parent", ev.SupervisorName).
				WithField("service", ev.ServiceName).
				Fatal("failed to stop in a timely manner")
		}
	}
}

type metrics struct {
	Metrics
	t0 time.Time
}

func (m *metrics) OnBackoff(suture.EventBackoff) {
	m.t0 = time.Now()
}

func (m *metrics) OnResume(event suture.EventResume) {
	bucket := fmt.Sprintf("%s.", event.SupervisorName)
	m.Metrics.Duration(bucket, time.Since(m.t0))
}

func (m metrics) OnError(bucket string, err error) {
	if err != nil {
		m.Incr(bucket + "errors")
	}
}

func (m metrics) OnTerm(event suture.EventServiceTerminate) {
	bucket := fmt.Sprintf("%s.%s.", event.SupervisorName, event.ServiceName)
	m.Incr(bucket + "restarts")

	if err, ok := event.Err.(error); ok {
		m.OnError(bucket, err)
	}
}

func (m metrics) OnPanic(event suture.EventServicePanic) {
	bucket := fmt.Sprintf("%s.%s.", event.SupervisorName, event.ServiceName)
	m.Incr(bucket + "restarts")
	m.Incr(bucket + "panics")
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
