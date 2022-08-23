//go:generate mockgen -source=channel.go -destination=../../../internal/mock/pkg/cluster/channel/channel.go -package=mock_channel

package channel

import (
	"context"
	"errors"

	"github.com/wetware/ww/internal/api/channel"
)

var (
	ErrEmpty  = errors.New("empty")
	ErrClosed = errors.New("closed")
)

type (
	MethodClose = channel.Closer_close
	MethodSend  = channel.Sender_send
	MethodRecv  = channel.Recver_recv
	MethodPeek  = channel.Peeker_peek
)

type CloseServer interface {
	Close(context.Context, MethodClose) error
}

type SendServer interface {
	Send(context.Context, MethodSend) error
}

type RecvServer interface {
	Recv(context.Context, MethodRecv) error
}

type PeekServer interface {
	Peek(context.Context, MethodPeek) error
}

type SendCloseServer interface {
	SendServer
	CloseServer
}

type PeekRecvServer interface {
	PeekServer
	RecvServer
}

type Server interface {
	CloseServer
	SendServer
	RecvServer
}

type PeekableServer interface {
	Server
	PeekServer
}
