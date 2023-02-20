package ww_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ww "github.com/wetware/ww/pkg"
)

func TestProto(t *testing.T) {
	t.Parallel()

	const ns = "test"
	matcher := ww.NewMatcher(ns)
	proto := ww.Subprotocol(ns)
	t.Log(proto)

	assert.True(t, matcher.Match(proto),
		"matcher should match subprotocol")
}
