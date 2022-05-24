package anchor_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/pkg/cap/cluster/internal/anchor"
	"github.com/wetware/ww/pkg/internal/bounded"
)

func TestPathFromProvider(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Succeed", func(t *testing.T) {
		p := pathProvider(func() (string, error) {
			return "foo/bar", nil
		})

		path, err := anchor.PathFromProvider(p)
		require.NoError(t, err, "should succeed")
		require.NoError(t, path.Err(),
			"path should not contain error")
		require.Equal(t, "/foo/bar", path.String(),
			"should contain expected path string")
	})

	t.Run("Fail", func(t *testing.T) {
		p := pathProvider(func() (string, error) {
			return "/should/not/appear", errors.New("error")
		})

		path, err := anchor.PathFromProvider(p)
		require.Error(t, err, "should fail")
		require.Error(t, path.Err(),
			"error should be present in returned path")
		require.Zero(t, path.String(),
			"should not contain path string")
	})
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
			want: []string{"/foo"},
		},
		{
			path: anchor.NewPath("/foo/bar/baz"),
			want: []string{"/foo", "/bar", "/baz"},
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

func TestPathValidation(t *testing.T) {
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
		{name: "alphabet", input: alpha},
		{name: "number", input: number},
		{name: "symbol", input: symbol},
		{name: "all_valid", input: alpha + number + symbol},
		{name: "sep_prefixed", input: "/" + valid},
		{name: "invalid", input: "foo" + invalid + "bar", invalid: true},
		{name: "not_in_range", input: "foo™bar", invalid: true},
	} {
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
			require.NoError(t, path.Bind(test(t, tt.input)).Err(),
				"should successfully bind")
		}
	}
}

func test(t *testing.T, want string) func(string) bounded.Type[string] {
	return func(got string) bounded.Type[string] {
		require.Equal(t, "/"+trimmed(want), got,
			"should match input stripped of separators.")
		return bounded.Value(got)
	}
}

func trimmed(s string) string {
	return strings.Trim(s, "/")
}

type pathProvider func() (string, error)

func (path pathProvider) Path() (string, error) {
	return path()
}
