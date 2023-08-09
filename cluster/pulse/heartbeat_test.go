package pulse_test

import (
	"testing"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/cluster/pulse"
)

func TestHeartbeat_MarshalUnmarshal(t *testing.T) {
	t.Parallel()

	var h pulse.Heartbeat
	require.NotPanics(t, func() {
		h = pulse.NewHeartbeat()
	}, "heartbeat uses single-segment arena and should never panic")

	h.SetTTL(time.Millisecond)
	require.Equal(t, time.Millisecond, h.TTL())

	err := h.SetHost("test.node.local")
	require.NoError(t, err, "should set hostname")

	fields, err := h.NewMeta(3)
	require.NoError(t, err, "should allocate meta fields")

	for i, entry := range []struct{ key, value string }{
		{key: "cloud_region", value: "us-east-1"},
		{key: "special_field", value: "special-value"},
		{key: "home", value: "/home/wetware"},
	} {
		err = fields.Set(i, entry.key+"="+entry.value)
		require.NoError(t, err, "must set field")
	}

	// marshal
	b, err := h.Message().MarshalPacked()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	t.Logf("payload size:  %d bytes", len(b))

	// unmarshal
	hb2 := pulse.Heartbeat{}
	m, err := capnp.UnmarshalPacked(b)
	require.NoError(t, err)
	err = hb2.ReadMessage(m)
	require.NoError(t, err)

	assert.Equal(t, h.TTL(), hb2.TTL())

	meta, err := h.Meta()
	require.NoError(t, err, "should return meta")

	for key, want := range map[string]string{
		"cloud_region":  "us-east-1",
		"special_field": "special-value",
		"home":          "/home/wetware",
		"missing":       "",
	} {
		value, err := meta.Get(key)
		require.NoError(t, err)
		require.Equal(t, want, value)
	}
}
