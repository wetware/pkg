package anchor_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/anchor"
)

func TestPath(t *testing.T) {
	t.Parallel()

	const (
		alpha   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		number  = "1234567890"
		symbol  = "!@#$%^&*()_+-=,./<>?[]{}|`~;:'/\""
		valid   = alpha + number + symbol
		invalid = "\\"
	)

	for _, tt := range []struct {
		name, input string
		invalid     bool
	}{
		{name: "root"},
		{name: "alphabet", input: alpha},
		{name: "number", input: number},
		{name: "symbol", input: symbol},
		{name: "all_valid", input: alpha + number + symbol},
		{name: "sep_prefixed", input: "/" + valid},
		{name: "invalid", input: "foo" + invalid + "bar", invalid: true},
		{name: "not_in_range", input: "foo™bar", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := anchor.NewPath(tt.input)
			if tt.invalid {
				require.Error(t, path.Err(),
					"should fail on invalid input")
				require.Zero(t, path.String(),
					"invalid path should produce empty string")
			} else {
				require.NoError(t, path.Err(),
					"should succeed on valid input")
				require.Equal(t, "/"+trimmed(tt.input), path.String(),
					"should match input and be separator-prefixed")
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	t.Parallel()

	path := anchor.JoinPath([]string{"foo", "bar"})
	require.NoError(t, path.Err(), "should bind path from parts")
	require.Equal(t, "/foo/bar", path.String())

	path = anchor.JoinPath([]string{"/foo/", "/bar"})
	require.NoError(t, path.Err(), "should bind path from parts")
	require.Equal(t, "/foo/bar", path.String())
}

func TestPathIteration(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		path anchor.Path
		want []string
	}{
		{}, // empty
		{
			path: anchor.NewPath("/foo"),
			want: []string{"foo"},
		},
		{
			path: anchor.NewPath("/foo/bar/baz"),
			want: []string{"foo", "bar", "baz"},
		},
	} {
		t.Run(trimmed(tt.path.String()), func(t *testing.T) {
			var got []string

			for path, s := tt.path.Next(); s != ""; path, s = path.Next() {
				got = append(got, s)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func trimmed(s string) string {
	return strings.Trim(s, "/")
}
