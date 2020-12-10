package mem_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/internal/api"
	"github.com/wetware/ww/pkg/mem"
)

func TestNil(t *testing.T) {
	assert.Equal(t, api.Value_Which_nil, mem.NilValue.Which())
}
