package proto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetware/pkg/util/proto"
)

func TestProto(t *testing.T) {
	t.Parallel()

	const ns = "test"
	matcher := proto.NewMatcher(ns)
	hit := proto.Root(ns)
	miss := proto.Root("miss")

	assert.True(t, matcher.Match(hit),
		"should match %s", hit)
	assert.False(t, matcher.Match(miss),
		"should not match %s", miss)

}
