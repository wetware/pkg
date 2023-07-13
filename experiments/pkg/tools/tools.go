package tools

import (
	"context"

	http_api "github.com/wetware/ww/experiments/api/http"
	api "github.com/wetware/ww/experiments/api/tools"
	http "github.com/wetware/ww/experiments/pkg/http/server"
)

type ToolServer struct{}

func (ToolServer) Http(ctx context.Context, call api.Tools_http) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	httpServer := http_api.Requester_ServerToClient(http.HttpServer{})
	return res.SetHttp(httpServer.AddRef())
}
