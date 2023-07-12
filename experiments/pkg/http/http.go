package http

import (
	"context"
	"io"
	"net/http"

	api "github.com/wetware/ww/experiments/api/http"
)

type HttpServer struct{}

// TODO try wasi-go instead of breaking encapsulation
func (HttpServer) Get(ctx context.Context, call api.HttpGetter_get) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	url, err := call.Args().Url()
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return res.SetError(err.Error())
	}

	// defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	res.SetStatus(uint32(resp.StatusCode))
	return res.SetBody(body)
}
