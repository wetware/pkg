package lang_test

import (
	"strings"
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/lang"
)

func TestBind(t *testing.T) {
	ww := lang.New(nil)

	t.Run("BindDoc", func(t *testing.T) {
		for _, tC := range []struct {
			desc, symbol string
			val          runtime.Value
			doc          []string
		}{{
			desc: "basic",
			val:  runtime.String("foo"),
			doc:  []string{"foo", "bar"},
		}, {
			desc: "whitespace",
			val:  runtime.String("foo"),
			doc:  []string{"foo", "bar", "\n", "\n"},
		}} {
			t.Run(tC.desc, func(t *testing.T) {
				require.NoError(t, ww.BindDoc(tC.symbol, tC.val, tC.doc...))

				v, err := ww.Resolve(tC.symbol)
				assert.NoError(t, err)
				assert.Equal(t, tC.val.String(), v.String())

				assert.Equal(t, doc(tC.doc), ww.Doc(tC.symbol))
			})
		}
	})

	t.Run("Bind", func(t *testing.T) {
		require.NoError(t, ww.Bind("foo", runtime.String("foo")), runtime.String("foo"))

		v, err := ww.Resolve("foo")
		assert.NoError(t, err)
		assert.Equal(t, runtime.String("foo").String(), v.String())
		assert.Equal(t, "", ww.Doc("foo"))
	})

	t.Run("ResolveMissing", func(t *testing.T) {
		_, err := ww.Resolve("fail")
		assert.EqualError(t, err, runtime.ErrNotFound.Error())
	})
}

func doc(ss []string) string {
	return strings.TrimSpace(strings.Join(ss, "\n"))
}
