package proto

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name  string
		input []protocol.ID
		want  protocol.ID
	}{
		{
			name:  "Empty",
			input: []protocol.ID{"", ""},
		},
		{
			name:  "Root",
			input: []protocol.ID{"/", ""},
			want:  "/",
		},
		{
			name:  "ShouldHandleSlashes",
			input: []protocol.ID{"/", "/", "/foo/", "/bar/", "/"},
			want:  "/foo/bar",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Join(tt.input...))
		})
	}
}

func TestAppendStrings(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		id   protocol.ID
		ss   []string
		want protocol.ID
	}{
		{
			name: "Empty",
			ss:   []string{"", ""},
		},
		{
			name: "Root",
			id:   "/",
			ss:   []string{"", ""},
			want: "/",
		},
		{
			name: "ShouldHandleSlashes",
			id:   "/",
			ss:   []string{"/", "/", "/foo/", "/bar/", "/"},
			want: "/foo/bar",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AppendStrings(tt.id, tt.ss...))
		})
	}
}

func TestParts(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name string
		id   protocol.ID
		want []protocol.ID
	}{
		{
			name: "Empty",
		},
		{
			name: "Root",
			id:   "/",
		},
		{
			name: "RelativePath",
			id:   "foo/bar",
			want: []protocol.ID{"foo", "bar"},
		},
		{
			name: "ShouldHandleSlashes",
			id:   "//foo//bar//baz//qux//",
			want: []protocol.ID{"foo", "bar", "baz", "qux"},
		},
	} {
		if len(tt.want) == 0 {
			assert.Empty(t, Parts(tt.id))
		} else {
			assert.Equal(t, tt.want, Parts(tt.id))
		}
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []struct {
		name          string
		id, base, end protocol.ID
	}{
		{
			name: "Empty",
		},
		{
			name: "Root",
			id:   "/",
		},
		{
			name: "Single",
			id:   "/foo/",
			end:  "foo",
		},
		{
			name: "Path",
			id:   "/foo/bar/baz",
			base: "/foo/bar",
			end:  "baz",
		},
		{
			name: "ShouldHandleSlashes",
			id:   "//foo//bar//baz//",
			base: "/foo/bar",
			end:  "baz",
		},
	} {
		gotBase, gotEnd := Split(tt.id)
		assert.Equal(t, tt.base, gotBase,
			"should have base='%s', got '%s'", tt.base, gotBase)
		assert.Equal(t, tt.end, gotEnd,
			"should have end='%s', got '%s'", tt.end, gotEnd)
	}
}
