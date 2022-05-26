package proc

import (
	"context"
)

type Process interface {
	Start(context.Context) error
	Wait(ctx context.Context) error
}
