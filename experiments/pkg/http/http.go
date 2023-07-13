package http

import (
	"context"
	"fmt"

	api "github.com/wetware/ww/experiments/api/http"
)

// Response contains the most basic attributes of an HTTP GET response
type Response struct {
	Body   []byte
	Status uint32
	Error  string
}

func (r Response) String() string {
	bodyLen := 15
	if len(r.Body) < bodyLen {
		bodyLen = len(r.Body)
	}

	return fmt.Sprintf("status: %d, error: %s, body: %s", r.Status, r.Error, string(r.Body)[:bodyLen])
}

type Requester api.Requester

// Get calls the top-level Get function and passes r as the requester
func (r Requester) Get(ctx context.Context, url string) (Response, error) {
	return Get(ctx, api.Requester(r), url)
}

// Get uses the getter capability to perform HTTP GET requests
func Get(ctx context.Context, requester api.Requester, url string) (Response, error) {
	f, release := requester.Get(ctx, func(hg api.Requester_get_Params) error {
		return hg.SetUrl(url)
	})
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return Response{}, err
	}

	resp, err := res.Response()
	if err != nil {
		return Response{}, err
	}

	status := resp.Status()

	resErr, err := resp.Error()
	if err != nil {
		return Response{}, err
	}

	buf, err := resp.Body()
	if err != nil {
		return Response{}, err
	}
	body := make([]byte, len(buf)) // avoid garbage-collecting the body
	copy(body, buf)

	return Response{
		Body:   body,
		Status: status,
		Error:  resErr,
	}, nil
}
