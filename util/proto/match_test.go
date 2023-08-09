package proto_test

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/ww/util/proto"

	"github.com/stretchr/testify/assert"
)

func TestMatchers(t *testing.T) {
	t.Parallel()
	t.Helper()

	for _, tt := range []matcherTest{
		{
			name:    "Exactly/match",
			matcher: proto.Exactly("foo"),
			input:   "/foo/bar/",
		},
		{
			name:          "Exactly/reject",
			matcher:       proto.Exactly("bar"),
			input:         "/foo/bar/",
			expectNoMatch: true,
		},
		{
			name:    "Prefix/match",
			matcher: proto.Prefix("/foo/bar/"),
			input:   "/foo/bar/baz/qux",
		},
		{
			name:          "Prefix/reject",
			matcher:       proto.Prefix("/foo/bar/"),
			input:         "/bar/foo/baz/qux/",
			expectNoMatch: true,
		},
		{
			name:    "Suffix/Match",
			matcher: proto.Suffix("/baz/qux"),
			input:   "/foo/bar/baz/qux",
		},
		{
			name:          "Suffix/reject",
			matcher:       proto.Suffix("/baz/qux/"),
			input:         "/foo/bar/qux/baz/",
			expectNoMatch: true,
		},
		{
			name: "MatchComplex",
			matcher: proto.Match(
				proto.Prefix("ww"),
				proto.SemVer("1.5.1"),
				proto.Exactly("ns"),
				proto.Exactly("rpc")),
			input: "/ww/1.0.0/ns/rpc/",
		},
		{
			name: "Chain",
			matcher: proto.Match(
				proto.Prefix("ww"),
				proto.SemVer("0.0.0")).
				Then(proto.Exactly("ns")),
			input: "/ww/0.0.0/ns",
		},
	} {
		tt.Run(t)
	}
}

func TestSemVer(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Match", func(t *testing.T) {
		t.Helper()

		for _, tt := range []matcherTest{
			{
				name:    "Identical",
				matcher: proto.SemVer("1.0.0"),
				input:   "/1.0.0/",
			},
			{
				name:    "MinorVersion/Higher",
				matcher: proto.SemVer("1.0.0"),
				input:   "/1.2.0/",
			},
			{
				name:    "MinorVersion/Lower",
				matcher: proto.SemVer("1.2.0"),
				input:   "/1.0.0/",
			},
			{
				name:    "PatchVersion/Higher",
				matcher: proto.SemVer("1.0.0"),
				input:   "/1.0.3/",
			},
			{
				name:    "PatchVersion/Lower",
				matcher: proto.SemVer("1.0.3"),
				input:   "/1.0.0/",
			},
			{
				name:    "Pre-Release/Local",
				matcher: proto.SemVer("1.0.0-alpha.1"),
				input:   "/1.0.0/",
			},
			{
				name:    "Pre-Release/Remote",
				matcher: proto.SemVer("1.0.0"),
				input:   "/1.0.0-alpha.1/",
			},
		} {
			tt.Run(t)
		}
	})

	t.Run("Reject", func(t *testing.T) {
		t.Helper()

		for _, tt := range []matcherTest{
			{
				name:          "MajorVersionsDiffer",
				matcher:       proto.SemVer("1.0.0"),
				input:         "/2.0.0/",
				expectNoMatch: true,
			},
			{
				name:          "MajorVersionsDiffer/MinorVersionsMatch",
				matcher:       proto.SemVer("1.1.0"),
				input:         "/2.1.0/",
				expectNoMatch: true,
			},
			{
				name:          "MajorVersionsDiffer/PatchVersionsMatch",
				matcher:       proto.SemVer("1.0.1"),
				input:         "/2.0.1/",
				expectNoMatch: true,
			},
			{
				name:          "SemVerMalformed",
				matcher:       proto.SemVer("1.0.0"),
				input:         "/not a semver string/",
				expectNoMatch: true,
			},
		} {
			tt.Run(t)
		}
	})
}

type matcherTest struct {
	name          string
	matcher       proto.MatchFunc
	input         protocol.ID
	expectNoMatch bool
}

func (mt matcherTest) Run(t *testing.T) {
	t.Run(mt.name, func(t *testing.T) {
		if match := mt.matcher.Match(mt.input); mt.expectNoMatch {
			assert.False(t, match, "should not match '%s'", mt.input)
		} else {
			assert.True(t, match, "should match '%s'", mt.input)
		}
	})
}
