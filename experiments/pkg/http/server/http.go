package server

import (
	"bytes"
	"context"
	"io"
	"net/http"

	api "github.com/wetware/ww/experiments/api/http"
)

type HttpServer struct{}

// TODO try wasi-go instead of breaking encapsulation

// Get calls the native HTTP GET method
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
	// http request failed, send feedback as to why
	if err != nil {
		response.SetError(err.Error())
		res.SetResponse(response)
		return err
	}

	buildResponse(&response, resp)
	return res.SetResponse(response)
}

// Post calls the native HTTP POST method
func (HttpServer) Post(ctx context.Context, call api.Requester_post) error {
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

	headers := make(map[string]string)
	if call.Args().HasHeaders() {
		headerList, err := call.Args().Headers()
		if err != nil {
			return err
		}
		for i := 0; i < headerList.Len(); i++ {
			header := headerList.At(i)
			if header.IsValid() {
				k, _ := header.Key()
				v, _ := header.Value()
				headers[k] = v
			}
		}
	}

	var body []byte
	if call.Args().HasBody() {
		body, err = call.Args().Body()
		if err != nil {
			return err
		}
	} else {
		body = make([]byte, 0)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	// request was malformed, send feedback as to why
	if err != nil {
		response.SetError(err.Error())
		res.SetResponse(response)
		return err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	// http request failed, send feedback as to why
	if err != nil {
		response.SetError(err.Error())
		res.SetResponse(response)
		return err
	}

	buildResponse(&response, resp)
	return res.SetResponse(response)
}

func buildResponse(apiR *api.Requester_Response, httpR *http.Response) error {
	// defer resp.Body.Close()
	body, err := io.ReadAll(httpR.Body)
	if err != nil {
		return err
	}
	if err = apiR.SetBody(body); err != nil {
		return err
	}

	apiR.SetStatus(uint32(httpR.StatusCode))
	return nil
}
