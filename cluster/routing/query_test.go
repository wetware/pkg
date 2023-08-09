package routing_test

// func TestIndex_string(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	for _, tt := range []struct {
// 		index routing.Index
// 		want  string
// 	}{
// 		{
// 			want:  "id",
// 			index: newIndex(api.View_Index_Which_peer),
// 		},
// 		{
// 			want:  "id_prefix",
// 			index: newIndex(api.View_Index_Which_peerPrefix),
// 		},
// 		{
// 			want:  "host",
// 			index: newIndex(api.View_Index_Which_host),
// 		},
// 		{
// 			want:  "host_prefix",
// 			index: newIndex(api.View_Index_Which_hostPrefix),
// 		},
// 		{
// 			want:  "meta",
// 			index: newIndex(api.View_Index_Which_meta),
// 		},
// 		{
// 			want:  "meta_prefix",
// 			index: newIndex(api.View_Index_Which_metaPrefix),
// 		},
// 	} {
// 		t.Run(tt.want, func(t *testing.T) {
// 			assert.Equal(t, tt.want, tt.index.String())
// 		})
// 	}
// }

// func TestIndex_matcher(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	for _, tt := range []struct {
// 		name  string
// 		index routing.Index
// 		match *record
// 		skip  *record
// 	}{
// 		{
// 			name:  "id",
// 			index: newPeerIndex("test"),
// 			match: &record{id: peer.ID("test")},
// 			skip:  &record{id: peer.ID("test_foo")},
// 		},
// 		{
// 			name:  "id_prefix",
// 			index: newPeerPrefixIndex("test"),
// 			match: &record{id: peer.ID("test_foo")},
// 			skip:  &record{id: peer.ID("tes")},
// 		},
// 		{
// 			name:  "host",
// 			index: newIndex(api.View_Index_Which_host),
// 			match: &record{host: "test"},
// 			skip:  &record{host: "test_foo"},
// 		},
// 		{
// 			name:  "host_prefix",
// 			index: newIndex(api.View_Index_Which_hostPrefix),
// 			match: &record{host: "test_foo"},
// 			skip:  &record{host: "tes"},
// 		},
// 		{
// 			name:  "meta",
// 			index: newMetaIndex("foo=bar"),
// 			match: &record{meta: newMeta("foo=bar", "bar=baz")},
// 			skip:  &record{meta: newMeta()},
// 		},
// 		{
// 			name:  "meta_prefix",
// 			index: newMetaPrefixIndex("foo=bar"),
// 			match: &record{meta: newMeta("foo=bar_baz", "bar=baz")},
// 			skip:  &record{meta: newMeta()},
// 		},
// 		{
// 			name:  "meta__empty",
// 			index: newMetaIndex(),
// 			match: &record{meta: newMeta()},
// 			skip:  &record{meta: newMeta("foo=bar")},
// 		},
// 		{
// 			name:  "meta_prefix__empty",
// 			index: newMetaPrefixIndex(),
// 			match: &record{meta: newMeta()},
// 			skip:  &record{meta: newMeta("foo=bar")},
// 		},
// 	} {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if tt.match != nil {
// 				require.True(t, tt.index.Match(tt.match),
// 					"should match record")
// 			}

// 			if tt.skip != nil {
// 				require.False(t, tt.index.Match(tt.skip),
// 					"should not match record")
// 			}
// 		})
// 	}
// }

// func TestQuery_Get(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	t.Run("Empty", func(t *testing.T) {
// 		table := routing.New(t0)

// 		it, err := table.NewQuery().Get(newIndex(api.View_Index_Which_peer))
// 		require.NoError(t, err, "should succeed on empty table")
// 		assert.NotNil(t, it, "should return iterator from empty table")
// 	})

// 	t.Run("NonEmpty", func(t *testing.T) {
// 		table := routing.New(t0)

// 		id := newPeerID()
// 		table.Upsert(&record{id: id})
// 		for i := 0; i < 10; i++ {
// 			table.Upsert(&record{id: newPeerID()}) // add extras
// 		}

// 		it, err := table.NewQuery().Get(newPeerIndex(id))
// 		require.NoError(t, err, "should succeed")
// 		require.NotNil(t, it, "should return iterator")

// 		assert.Equal(t, id, it.Next().Peer(), "should match peer index")
// 		assert.Nil(t, it.Next(), "iterator should be exhausted")
// 	})
// }

// func TestQuery_LowerBound(t *testing.T) {
// 	t.Parallel()
// 	t.Helper()

// 	t.Run("Empty", func(t *testing.T) {
// 		table := routing.New(t0)

// 		it, err := table.NewQuery().LowerBound(newIndex(api.View_Index_Which_peer))
// 		require.NoError(t, err, "should succeed on empty table")
// 		assert.NotNil(t, it, "should return iterator from empty table")
// 	})
// }

// func newIndex(w api.View_Index_Which) routing.Index {
// 	_, seg := capnp.NewSingleSegmentMessage(nil)
// 	ix, err := api.NewRootView_Index(seg)
// 	if err != nil {
// 		panic(err)
// 	}

// 	switch w {
// 	case api.View_Index_Which_peer:
// 		ix.SetPeer("test")
// 	case api.View_Index_Which_peerPrefix:
// 		ix.SetPeerPrefix("test")
// 	case api.View_Index_Which_host:
// 		ix.SetHost("test")
// 	case api.View_Index_Which_hostPrefix:
// 		ix.SetHostPrefix("test")
// 	case api.View_Index_Which_meta:
// 		ix.NewMeta(0)
// 	case api.View_Index_Which_metaPrefix:
// 		ix.NewMetaPrefix(0)
// 	}

// 	return routing.Index{View_Index: ix}
// }

// func newPeerIndex(id peer.ID) routing.Index {
// 	index := newIndex(api.View_Index_Which_peer)
// 	index.SetPeer(string(id))
// 	return index
// }

// func newPeerPrefixIndex(id peer.ID) routing.Index {
// 	index := newIndex(api.View_Index_Which_peerPrefix)
// 	index.SetPeerPrefix(string(id))
// 	return index
// }

// func newMetaIndex(ss ...string) routing.Index {
// 	meta := newMeta(ss...)
// 	index := newIndex(api.View_Index_Which_meta)
// 	index.SetMeta(capnp.TextList(meta))
// 	return index
// }

// func newMetaPrefixIndex(ss ...string) routing.Index {
// 	meta := newMeta(ss...)
// 	index := newIndex(api.View_Index_Which_metaPrefix)
// 	index.SetMetaPrefix(capnp.TextList(meta))
// 	return index
// }

// func newMeta(ss ...string) routing.Meta {
// 	_, seg := capnp.NewSingleSegmentMessage(nil)
// 	m, err := capnp.NewTextList(seg, int32(len(ss)))
// 	if err != nil {
// 		panic(err)
// 	}

// 	for i, s := range ss {
// 		if err = m.Set(i, s); err != nil {
// 			panic(err)
// 		}
// 	}

// 	return routing.Meta(m)
// }
