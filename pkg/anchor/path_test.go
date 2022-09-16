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

func TestPathFromParts(t *testing.T) {
	t.Parallel()

	path := anchor.PathFromParts([]string{"foo", "bar"})
	assert.NoError(t, path.Err(), "should bind path from parts")

	failed := anchor.PathFromParts([]string{"foo", "/fail"})
	assert.Error(t, failed.Err(), "should not bind invalid path segment")
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

func TestIsRoot(t *testing.T) {
	t.Parallel()

	assert.True(t, anchor.NewPath("").IsRoot(),
		"should be root")
	assert.True(t, anchor.Path{}.IsRoot(),
		"zero-value path should be root")
	assert.False(t, anchor.NewPath("/foo/bar").IsRoot(),
		"should not be root")
	assert.False(t, anchor.NewPath("µ").IsRoot(),
		"invalid path should not be root")
}

func TestIsZero(t *testing.T) {
	t.Parallel()

	assert.False(t, anchor.NewPath("").IsZero(),
		`root path should not be zero-value`)
	assert.True(t, anchor.Path{}.IsZero(),
		"anchor.Path{} should be zero-value")
	assert.False(t, anchor.NewPath("/foo").IsZero(),
		"non-root path should not be zero-value")
	assert.False(t, anchor.NewPath("µ").IsZero(),
		"invalid path should not be zero-value")
}

func TestWithChild(t *testing.T) {
	t.Parallel()

	parent := anchor.NewPath("/foo")

	child := parent.WithChild("baz")
	assert.NoError(t, child.Err(), "should bind child")

	fail := parent.WithChild("/baz")
	assert.Error(t, fail.Err(), "should not bind invalid child")
}

func TestIsSubpath(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name            string
		parent, subpath anchor.Path
		expect          bool
	}{
		{
			name:   "root",
			expect: false,
		},
		{
			name:    "child",
			parent:  anchor.NewPath("/foo"),
			subpath: anchor.NewPath("/foo/bar"),
			expect:  true,
		},
		{
			name:    "nonchild",
			parent:  anchor.NewPath("/foo"),
			subpath: anchor.NewPath("/foobar"),
			expect:  false,
		},
		{
			name:    "nonchild",
			parent:  anchor.NewPath("/foo"),
			subpath: anchor.NewPath("/foobar/baz"),
			expect:  false,
		},
		{
			name:    "childOfRoot",
			parent:  anchor.NewPath(""),
			subpath: anchor.NewPath("/foo"),
			expect:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ok := tt.parent.IsSubpath(tt.subpath)
			if tt.expect {
				assert.True(t, ok, "%s should be a supbath of %s",
					tt.subpath,
					tt.parent)
			} else {
				assert.False(t, ok, "%s should NOT be a subpath of %s",
					tt.subpath,
					tt.parent)
			}
		})
	}
}

func TestIsChild(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name          string
		parent, child anchor.Path
		expect        bool
	}{
		{
			name:   "root",
			expect: false,
		},
		{
			name:   "child",
			parent: anchor.NewPath("/foo"),
			child:  anchor.NewPath("/foo/bar"),
			expect: true,
		},
		{
			name:   "deep",
			parent: anchor.NewPath("/foo/bar/baz"),
			child:  anchor.NewPath("/foo/bar/baz/qux"),
			expect: true,
		},
		{
			name:   "samePrefix",
			parent: anchor.NewPath("/foo"),
			child:  anchor.NewPath("/foobar"),
			expect: false,
		},
		{
			name:   "nonchild",
			parent: anchor.NewPath("/foo"),
			child:  anchor.NewPath("/foobar/baz"),
			expect: false,
		},
		{
			name:   "child",
			parent: anchor.NewPath("/foo"),
			child:  anchor.NewPath("/foo/bar/baz"),
			expect: false,
		},
		{
			name:   "childOfRoot",
			parent: anchor.NewPath(""),
			child:  anchor.NewPath("/foo"),
			expect: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ok := tt.parent.IsChild(tt.child)
			if tt.expect {
				assert.True(t, ok, "%s should be a subpath of %s",
					tt.child,
					tt.parent)
			} else {
				assert.False(t, ok, "%s should NOT be a subpath of %s",
					tt.child,
					tt.parent)
			}
		})
	}
}

func trimmed(s string) string {
	return strings.Trim(s, "/")
}
