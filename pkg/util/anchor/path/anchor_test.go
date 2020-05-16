package anchorpath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

func TestParts(t *testing.T) {
	testCases := []struct {
		desc, path string
		expected   []string
	}{{
		desc:     "empty",
		path:     "",
		expected: []string{},
	}, {
		desc:     "root",
		path:     "/",
		expected: []string{},
	}, {
		desc:     "multipart",
		path:     "/foo/bar/baz/qux",
		expected: []string{"foo", "bar", "baz", "qux"},
	}, {
		desc:     "complex",
		path:     "////foo/bar//baz/qux///////",
		expected: []string{"foo", "bar", "baz", "qux"},
	}}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expected, anchorpath.Parts(tC.path))
		})
	}
}

func TestJoin(t *testing.T) {
	testCases := []struct {
		desc, expected string
		parts          []string
	}{{
		desc:     "empty",
		parts:    []string{},
		expected: "/",
	}, {
		desc:     "root",
		parts:    []string{"/"},
		expected: "/",
	}, {
		desc:     "complex",
		parts:    []string{"foo/", "//bar//"},
		expected: "/foo/bar",
	}}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expected, anchorpath.Join(tC.parts...))
		})
	}
}

func TestRoot(t *testing.T) {
	testCases := []struct {
		desc     string
		parts    []string
		expected bool
	}{{
		desc:     "empty",
		parts:    []string{""},
		expected: true,
	}, {
		desc:     "clean",
		parts:    []string{"/"},
		expected: true,
	}, {
		desc:     "dirty",
		parts:    []string{"//////"},
		expected: true,
	}, {
		desc:     "nonroot",
		parts:    []string{"/foo"},
		expected: false,
	}}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

		})
	}
}
