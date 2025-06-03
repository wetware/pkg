package main

import (
	"context"
	"os"
	"runtime"

	core_api "github.com/wetware/pkg/api/core"
	proc_api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp/events"
)

var urls chan string

func main() {
	urls = make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	external := core_api.ProcessInit{}
	eventHandler := events.New()
	eventCap := proc_api.Events_ServerToClient(eventHandler)
	external.Events(ctx, func(pi core_api.ProcessInit_events_Params) error {
		return pi.SetHandler(eventCap)
	})

	urls <- os.Args[1]

	for {
		select {
		case <-eventHandler.OnPause():
			// Quote:
			// Gosched yields the processor, allowing other goroutines to run. It does not
			// suspend the current goroutine, so execution resumes automatically.
			runtime.Gosched()
			select {
			case <-eventHandler.OnResume():
			}
		case <-crawl(ctx, <-urls):
		}
	}
}

func crawl(ctx context.Context, url string) <-chan struct{} {
	// http get...
	// parse page...
	var results []string

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			return
		case done <- struct{}{}:
		}
		for _, r := range results {
			select {
			case <-ctx.Done():
				return
			case urls <- r:
			}
		}
	}()
	return done
}
