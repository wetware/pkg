package csp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_csp "github.com/wetware/ww/internal/mock/pkg/csp"
	"github.com/wetware/ww/pkg/csp"
)

func TestSendStream(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := mock_csp.NewMockSendServer(ctrl)
	server.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(128)

	sender := csp.NewSender(server)
	defer sender.Release()

	stream := sender.NewStream(context.Background())
	for i := 0; i < 128; i++ {
		err := stream.Send(csp.Text("hello, world!"))
		require.NoError(t, err, "should send text")
	}

	err := stream.Close()
	assert.NoError(t, err, "should close gracefully")
}

func TestSendStream_Error(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	want := errors.New("test")

	server := mock_csp.NewMockSendServer(ctrl)
	server.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(64)
	server.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(want).
		Times(1)
	server.EXPECT().
		Send(gomock.Any(), gomock.Any()).
		Return(nil).
		MaxTimes(64)

	sender := csp.NewSender(server)
	defer sender.Release()

	stream := sender.NewStream(context.Background())
	for i := 0; i < 64; i++ {
		err := stream.Send(csp.Text("hello, world!"))
		require.NoError(t, err, "should send text")
	}

	// The next call will trigger the error, but it might
	// not be detected synchronously.   We have no way of
	// knowing which of these calls will detect the error.
	for i := 0; i < 64; i++ {
		// Maximize the chance that of detecting the error
		// in-flight.  This helps with code coverage.
		time.Sleep(time.Millisecond)

		err := stream.Send(csp.Text("hello, world!"))
		if err != nil {
			require.ErrorIs(t, err, want, "should return error")
		}
	}

	err := stream.Close()
	require.ErrorIs(t, err, want, "should return error")
}
