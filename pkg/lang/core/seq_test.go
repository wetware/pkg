package core_test

// func TestCons(t *testing.T) {
// 	items := valueRange(16)

// 	arena := capnp.SingleSegment(nil)

// 	var err error
// 	var list core.List = core.EmptyList
// 	for _, item := range items {
// 		if list, err = core.Cons(arena, item, list); err != nil {
// 			break
// 		}
// 	}

// 	require.NoError(t, err)

// 	// count ok?
// 	cnt, err := list.Count()
// 	require.NoError(t, err)
// 	require.Equal(t, len(items), cnt)

// 	// items ok?
// 	results, err := core.ToSlice(list)
// 	require.NoError(t, err)

// 	for i, got := range results {
// 		want := results[len(items)-(1+i)]

// 		eq, err := core.Eq(want, got)
// 		require.NoError(t, err)
// 		require.True(t, eq, "unequal values at index %d", i)
// 	}
// }
