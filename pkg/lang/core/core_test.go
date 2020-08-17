package core_test

import (
	"reflect"
	"testing"

	"github.com/spy16/sabre/runtime"
	"github.com/stretchr/testify/assert"
)

type evalTestCase struct {
	title   string
	getEnv  func() runtime.Runtime
	form    runtime.Value
	want    runtime.Value
	wantErr bool
}

func runEvalTests(t *testing.T, cases []evalTestCase) {
	for _, tt := range cases {
		t.Run(tt.title, func(t *testing.T) {
			var env runtime.Runtime
			if tt.getEnv != nil {
				env = tt.getEnv()
			}

			got, err := tt.form.Eval(env)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s.Eval() error = %v, wantErr %v",
					reflect.TypeOf(tt.form), err, tt.wantErr)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
