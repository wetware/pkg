package http

import (
	"context"
	"encoding/base64"
	"fmt"

	"capnproto.org/go/capnp/v3"
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

// NewBasicAuth creates a new Basic HTTP Auth header content from a username and password
func NewBasicAuth(user, password string) string {
	joined := []byte(fmt.Sprintf("%s:%s", user, password))
	encoded := base64.StdEncoding.EncodeToString(joined)
	return fmt.Sprintf("Basic %s", encoded)
}

type Requester api.Requester

// Get calls the top-level Get function and passes r as the requester
func (r Requester) Get(ctx context.Context, url string) (Response, error) {
	return Get(ctx, api.Requester(r), url)
}

func (r Requester) Post(ctx context.Context, url string, headers map[string]string, body []byte) (Response, error) {
	return Post(ctx, api.Requester(r), url, headers, body)
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

// Post uses the getter capability to perform HTTP POST requests
func Post(ctx context.Context, requester api.Requester, url string, headers map[string]string, body []byte) (Response, error) {
	f, release := requester.Post(ctx, func(hp api.Requester_post_Params) error {

		// headers
		nh, err := hp.NewHeaders(int32(len(headers)))
		if err != nil {
			return err
		}
		i := 0
		for k, v := range headers {
			_, seg, _ := capnp.NewMessage(capnp.SingleSegment(nil))
			header, _ := api.NewRequester_Header(seg)
			header.SetKey(k)
			header.SetValue(v)
			if err = nh.Set(i, header); err != nil {
				return err
			}
			i++
		}
		if err = hp.SetHeaders(nh); err != nil {
			return err
		}

		// body
		buf := make([]byte, len(body)) // avoid garbage-collecting the body
		copy(buf, body)
		if err = hp.SetBody(buf); err != nil {
			return err
		}

		// url
		return hp.SetUrl(url)
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
	resBody := make([]byte, len(buf)) // avoid garbage-collecting the body
	copy(resBody, buf)

	return Response{
		Body:   resBody,
		Status: status,
		Error:  resErr,
	}, nil
}
