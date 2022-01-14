package ww_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ww "github.com/wetware/ww/pkg"
)

func TestProto(t *testing.T) {
	t.Parallel()

	const ns = "test"
	match := ww.NewMatcher(ns)
	proto := ww.Subprotocol(ns)

	assert.True(t, match(string(proto)),
		"matcher should match subprotocol")
}
