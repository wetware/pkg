package io_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_io "github.com/wetware/ww/internal/mock/io"

	"github.com/wetware/ww/pkg/ocap/io"
)

var errTest = errors.New("test")

func TestReadWriteCloser(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		rwc  = mock_io.NewMockReadWriteCloser(ctrl)
		rwcc = io.NewReadWriteCloser(rwc, nil)
	)

	t.Run("Close", func(t *testing.T) {
		t.Parallel()

		succeed := rwc.EXPECT().
			Close().
			Times(1)

		fail := rwc.EXPECT().
			Close().
			Return(errTest).
			Times(1)

		/*
			Test that capnp's E-Order invariant is respected by running
			tests concurrently.  The call order defined on the mock value
			will be violated if E-order is not preserved.
		*/
		fail.After(succeed)

		err0 := rwcc.Close(context.TODO())
		err1 := rwcc.Close(context.TODO())

		assert.NoError(t, err0, "should succeed")
		assert.ErrorIs(t, err1, errTest, "should fail with errTest")
	})

	t.Run("Read", func(t *testing.T) {
		t.Parallel()

		var b = make([]byte, 4)

		full := rwc.EXPECT().
			Read(b).
			DoAndReturn(func(b []byte) (int, error) {
				return copy(b, "test"), nil
			}).
			Times(1)

		partial := rwc.EXPECT().
			Read(b).
			DoAndReturn(func(b []byte) (int, error) {
				return copy(b, "tes"), nil
			}).
			Times(1)

		fail := rwc.EXPECT().
			Read(b).
			DoAndReturn(func(b []byte) (int, error) {
				return copy(b, "tes"), errTest
			}).
			Times(1)

		/*
			Test that capnp's E-Order invariant is respected by running
			tests concurrently.  The call order defined on the mock value
			will be violated if E-order is not preserved.
		*/
		fail.After(partial).After(full)

		t.Run("FullRead", func(t *testing.T) {
			f, release := rwcc.Read(context.TODO(), 4)
			defer release()

			p, err := f.Await(context.TODO())
			require.NoError(t, err, "should read")
			assert.Equal(t, "test", string(p), "should return full read")
		})

		t.Run("PartialRead", func(t *testing.T) {
			f, release := rwcc.Read(context.TODO(), 4)
			defer release()

			p, err := f.Await(context.TODO())
			require.NoError(t, err, "should read")
			assert.Equal(t, "tes", string(p), "should return partial read")
		})

		t.Run("Error", func(t *testing.T) {
			f, release := rwcc.Read(context.TODO(), 4)
			defer release()

			p, err := f.Await(context.TODO())
			assert.ErrorIs(t, err, errTest, "should fail with errTest")
			assert.Equal(t, "tes", string(p), "should return partial read")
		})
	})

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		var b = []byte("test")

		full := rwc.EXPECT().
			Write(gomock.Eq(b)).
			Return(4, nil).
			Times(1)

		partial := rwc.EXPECT().
			Write(gomock.Eq(b)).
			Return(3, errTest).
			Times(1)

		/*
			Test that capnp's E-Order invariant is respected by running
			tests concurrently.  The call order defined on the mock value
			will be violated if E-order is not preserved.
		*/
		partial.After(full)

		t.Run("FullWrite", func(t *testing.T) {
			f, release := rwcc.Write(context.TODO(), b)
			defer release()

			n, err := f.Await(context.TODO())
			require.NoError(t, err, "should write")
			assert.Equal(t, int64(4), n, "should perform full write")
		})

		t.Run("PartialWrite", func(t *testing.T) {
			f, release := rwcc.Write(context.TODO(), b)
			defer release()

			n, err := f.Await(context.TODO())
			assert.ErrorIs(t, err, errTest, "should fail with errTest")
			assert.Equal(t, int64(3), n, "should perform partial write")
		})
	})
}
