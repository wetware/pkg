package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/lthibault/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/metrics"
	"github.com/wetware/ww/pkg/runtime"
	"go.uber.org/fx"
)

func TestEnv_Options(t *testing.T) {
	t.Parallel()

	/*
		Test that optional env dependencies are created.
	*/

	fs := flags{
		"ns": "test",
	}

	env := runtime.Env{
		Flags: fs,
	}

	vat := newVat()
	defer vat.Host.Close()

	var (
		ctx    context.Context
		logger log.Logger
		mc     metrics.Client
	)

	app := fx.New(env.Options(),
		fx.Supply(vat),
		fx.Populate(
			&ctx,
			&logger,
			&mc))

	err := app.Start(context.Background())
	require.NoError(t, err, "fx application should start")

	assert.NotNil(t, ctx, "should populate context")
	assert.NotNil(t, logger, "should populate logger")
	assert.NotNil(t, mc, "should populate metrics")
}

type flags map[string]interface{}

func (f flags) IsSet(name string) bool {
	_, ok := f[name]
	return ok
}

func (f flags) Bool(name string) bool {
	v, _ := f[name].(bool)
	return v
}

func (f flags) Path(name string) string {
	v, _ := f[name].(string)
	return v
}

func (f flags) String(name string) string {
	v, _ := f[name].(string)
	return v
}

func (f flags) StringSlice(name string) []string {
	v, _ := f[name].([]string)
	return v
}

func (f flags) Duration(name string) time.Duration {
	v, _ := f[name].(time.Duration)
	return v
}

func (f flags) Float64(name string) float64 {
	v, _ := f[name].(float64)
	return v
}

func newVat() casm.Vat {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()))
	if err != nil {
		panic(err)
	}

	return casm.Vat{
		NS:   "test",
		Host: h,
	}
}
