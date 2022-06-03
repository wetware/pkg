package io_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/wetware/ww/pkg/ocap/io"
)

func BenchmarkReader(b *testing.B) {
	for _, size := range [...]int{
		1 << 10,
		2 << 10,
		4 << 10,
		8 << 10,
		16 << 10,
	} {
		b.Run(fmt.Sprintf("%dkB", size/1024), func(b *testing.B) {
			benchRead(b, size)
		})
	}
}

func BenchmarkWriter(b *testing.B) {
	for _, size := range [...]int{
		1 << 10,
		2 << 10,
		4 << 10,
		8 << 10,
		16 << 10,
	} {
		b.Run(fmt.Sprintf("%dkB", size/1024), func(b *testing.B) {
			benchWrite(b, size)
		})
	}
}

func benchRead(b *testing.B, payloadSize int) {
	var (
		payload = make(staticReader, payloadSize)
		r       = io.NewReader(payload, nil)
		waiters = make([]<-chan struct{}, b.N)
	)

	rand.Read(payload)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		f, release := r.Read(context.TODO(), payloadSize)
		defer release()

		waiters[i] = f.Done()
	}

	// wait for all tasks to complete
	for _, wait := range waiters {
		<-wait
	}

	// stop the timer before the deferred calls are made
	b.StopTimer()
}

func benchWrite(b *testing.B, payloadSize int) {
	var (
		w       = io.NewWriter(ioutil.Discard, nil)
		waiters = make([]<-chan struct{}, b.N)
		payload = make([]byte, payloadSize)
	)

	rand.Read(payload)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		f, release := w.Write(context.TODO(), payload)
		defer release()

		waiters[i] = f.Done()
	}

	// wait for all tasks to complete
	for _, wait := range waiters {
		<-wait
	}

	// stop the timer before the deferred calls are made
	b.StopTimer()
}

type staticReader []byte

func (r staticReader) Read(b []byte) (int, error) {
	return copy(b, r), nil
}
