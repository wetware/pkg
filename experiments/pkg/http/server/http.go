package server

import (
	"context"
	"io"
	"net/http"

	api "github.com/wetware/ww/experiments/api/http"
)

type HttpServer struct{}

// TODO try wasi-go instead of breaking encapsulation
func (HttpServer) Get(ctx context.Context, call api.Requester_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	response, err := res.NewResponse()
	if err != nil {
		return err
	}

	url, err := call.Args().Url()
	if err != nil {
		return err
	}

	resp, err := http.Get(url)

	if err != nil {
		response.SetError(err.Error())
		return res.SetResponse(response)
	}

	// defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err = response.SetBody(body); err != nil {
		response.SetError(err.Error())
		return res.SetResponse(response)
	}

	response.SetStatus(uint32(resp.StatusCode))
	return res.SetResponse(response)
}
