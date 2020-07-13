package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
	service_test.go tests unexported types in the service package
*/

func TestScheduler(t *testing.T) {
	s := newScheduler(time.Second, noJitter{})

	t.Run("Initial state", func(t *testing.T) {
		assert.Equal(t, time.Second, s.d)
		assert.Equal(t, time.Second, s.remaining)
		assert.Equal(t, noJitter{}, s.j)
	})

	t.Run("Advance before deadline", func(t *testing.T) {
		assert.False(t, s.Advance(time.Millisecond*999))
		assert.Equal(t, time.Millisecond, s.remaining)
	})

	t.Run("Advance past deadline", func(t *testing.T) {
		assert.True(t, s.Advance(time.Millisecond))
		assert.Equal(t, time.Millisecond*0, s.remaining)
	})

	t.Run("Reset", func(t *testing.T) {
		s.Reset()
		assert.Equal(t, time.Second, s.d)
		assert.Equal(t, time.Second, s.remaining)
	})
}

type noJitter struct{}

func (noJitter) Jitter(d time.Duration) time.Duration {
	return d
}
