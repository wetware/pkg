package http

import (
	"context"
	"fmt"

	http_api "github.com/wetware/ww/experiments/api/http"
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

// get uses the getter capability to perform HTTP GET requests
func Get(ctx context.Context, getter http_api.HttpGetter, url string) (Response, error) {
	f, release := getter.Get(ctx, func(hg http_api.HttpGetter_get_Params) error {
		return hg.SetUrl(url)
	})
	defer release()
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return Response{}, err
	}

	status := res.Status()

	resErr, err := res.Error()
	if err != nil {
		return Response{}, err
	}

	buf, err := res.Body()
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
