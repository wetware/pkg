package core_test

import (
	"reflect"
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type evalTestCase struct {
	desc    string
	getEnv  func() runtime.Runtime
	form    runtime.Value
	want    runtime.Value
	wantErr bool
}

func runEvalTests(t *testing.T, cases []evalTestCase) {
	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			var env runtime.Runtime
			if tt.getEnv != nil {
				env = tt.getEnv()
			}

			got, err := tt.form.Eval(env)
			if tt.wantErr {
				assert.Error(t, err, "expected error, but got nil")
				return
			}

			require.NoError(t, err, "%s.Eval() error = %v, wantErr %v",
				reflect.TypeOf(tt.form),
				err,
				tt.wantErr,
			)

			assert.Equal(t, tt.want.String(), got.String())
		})
	}
}
