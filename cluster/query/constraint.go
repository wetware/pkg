package query

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wetware/pkg/cluster/routing"
)

// Matcher reports whether a routing record matches some
// arbitrary criteria.
type Matcher interface {
	// Match returns true if the supplied record matches
	// some arbitrary criteria.  It is used to implement
	// filtering.  See:  Where.
	Match(routing.Record) bool
}

// Constraints restrict the results of a selector to a subset
// of current matches.
type Constraint func(routing.Iterator) Selector

// Where restricts a selection according to some arbitrary
// criteria.  It is effectively a filter.
func Where(match Matcher) Constraint {
	return func(it routing.Iterator) Selector {
		return just(&filterIter{
			Matcher:  match,
			Iterator: it,
		})
	}
}

// While iterates over a selection until the predicate returns
// boolean false.
func While(predicate Matcher) Constraint {
	return func(it routing.Iterator) Selector {
		return just(&predicateIter{
			Matcher:  predicate,
			Iterator: it,
		})
	}
}

// Limit restricts the selection to n items.
func Limit(n int) Constraint {
	if n <= 0 {
		return func(routing.Iterator) Selector {
			return Failuref("expected limit > 0 (got %d)", n)
		}
	}

	return While(matchFunc(func(r routing.Record) (ok bool) {
		ok = n > 0
		n--
		return
	}))
}

// To restricts the selection to items less-than-or-equal-to
// the index.  Use with From to implement range queries.
func To(index routing.Index) Constraint {
	matcher, err := matchIndex(index)
	if err != nil {
		return failure(err)
	}

	return While(leq(matcher))
}

// First restricts the selection to a single item.
func First() Constraint {
	return Limit(1)
}

func just(it routing.Iterator) Selector {
	return func(routing.Snapshot) (routing.Iterator, error) {
		return it, nil
	}
}

func failure(err error) Constraint {
	return func(routing.Iterator) Selector {
		return Failure(err)
	}
}

func leq(m Matcher) matchFunc {
	var reached bool
	return func(r routing.Record) bool {
		match := m.Match(r)
		reached = reached || match
		return !reached || match
	}
}

func matchIndex(index routing.Index) (matchFunc, error) {
	switch index.String() {
	case "id":
		return matchPeer(index)

	case "host":
		return matchHost(index)
		// host, err := x.HostBytes()
		// return func(r routing.Record) bool {
		// 	name, err := r.Host()
		// 	if err != nil {
		// 		return false
		// 	}

		// 	if ix.Prefix() {
		// 		return strings.HasPrefix(name, string(host)) // TODO:  unsafe.Pointer
		// 	}

		// 	return name == string(host)
		// }, err

	case "meta":
		return matchMeta(index)
		// index, err := x.Meta()
		// return func(r routing.Record) bool {
		// 	meta, err := r.Meta()
		// 	if err != nil {
		// 		return false
		// 	}

		// 	if ix.Prefix() {
		// 		return matchMeta(strings.HasPrefix, meta, routing.Meta(index))
		// 	}

		// 	return err == nil && matchMeta(fieldEq, meta, routing.Meta(index))
		// }, err
	}

	return nil, fmt.Errorf("invalid index: %s", index)
}

func matchPeer(index routing.Index) (matchFunc, error) {
	var (
		id  string
		err error
	)

	switch ix := index.(type) {
	case routing.PeerIndex:
		var b []byte
		if b, err = ix.PeerBytes(); err == nil {
			id = string(b) // TODO:  unsafe.Pointer
		}

	case interface{ Peer() (string, error) }:
		id, err = ix.Peer()

	default:
		err = errors.New("not a peer index")
	}

	if err != nil {
		return nil, err
	}

	if index.Prefix() {
		return func(r routing.Record) bool {
			return strings.HasPrefix(string(r.Peer()), id) // TODO:  unsafe.Pointer
		}, nil
	}

	return func(r routing.Record) bool {
		return string(r.Peer()) == id // TODO:  unsafe.Pointer
	}, nil
}

func matchHost(index routing.Index) (matchFunc, error) {
	var (
		host string
		err  error
	)

	switch ix := index.(type) {
	case routing.HostIndex:
		var b []byte
		if b, err = ix.HostBytes(); err == nil {
			host = string(b) // TODO:  unsafe.Pointer
		}

	case interface{ Host() (string, error) }:
		host, err = ix.Host()

	default:
		err = errors.New("not a host index")
	}

	if err != nil {
		return nil, err
	}

	if index.Prefix() {
		return func(r routing.Record) bool {
			return matchStr(strEq, r.Host, host)
		}, nil
	}

	return func(r routing.Record) bool {
		return matchStr(strings.HasPrefix, r.Host, host)
	}, nil
}

func strEq(s0, s1 string) bool {
	return s0 == s1
}

func matchStr(match func(s0, s1 string) bool, f func() (string, error), target string) bool {
	s, err := f()
	return err == nil && match(s, target)
}

func matchMeta(index routing.Index) (matchFunc, error) {
	ix, ok := index.(interface{ Meta() (routing.Meta, error) })
	if !ok {
		return nil, errors.New("not a meta index")
	}

	meta, err := ix.Meta()
	if err != nil {
		return nil, err
	}

	if index.Prefix() {
		return func(r routing.Record) bool {
			return metaEq(strings.HasPrefix, r, meta)
		}, nil
	}

	return func(r routing.Record) bool {
		return metaEq(strEq, r, meta)
	}, nil

}

type matchFunc func(routing.Record) bool

func (match matchFunc) Match(r routing.Record) bool {
	return match(r)
}

func metaEq(match func(s0, s1 string) bool, r routing.Record, meta routing.Meta) bool {
	rmeta, err := r.Meta()
	if err != nil {
		return false
	}

	if rmeta.Len() == 0 {
		return meta.Len() == 0
	}

	// TODO:  reduce allocations with FooBytes()
	for i := 0; i < rmeta.Len(); i++ {
		f, err := rmeta.At(i)
		if err != nil {
			return false
		}

		value, err := meta.Get(f.Key)
		if err != nil || !match(value, f.Value) {
			return false
		}
	}

	return true
}
