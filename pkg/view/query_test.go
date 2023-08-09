package view_test

import (
	"testing"

	"capnproto.org/go/capnp/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/wetware/ww/api/cluster"
	"github.com/wetware/ww/pkg/view"
)

func TestSelector(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name     string
		selector view.Selector
		which    api.View_Selector_Which
	}{
		{
			name:     "Match",
			selector: view.Match(hostIndex("foo")),
			which:    api.View_Selector_Which_match,
		},
		{
			name:     "From",
			selector: view.From(hostIndex("foo")),
			which:    api.View_Selector_Which_from,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := selector()
			err := tt.selector(s)
			require.NoError(t, err, "should succeed")
			assert.Equal(t, tt.which, s.Which(), "should be %s", tt.which)
		})
	}
}

func selector() api.View_Selector {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	s, _ := api.NewRootView_Selector(seg)
	return s
}

type hostIndex string

func (hostIndex) String() string                { return "host" }
func (hostIndex) Prefix() bool                  { return false }
func (ix hostIndex) HostBytes() ([]byte, error) { return []byte(ix), nil }
