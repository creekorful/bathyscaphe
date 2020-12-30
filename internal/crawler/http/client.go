package http

//go:generate mockgen -destination=../http_mock/client_mock.go -package=http_mock . Client

import (
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"strings"
)

// ErrTimeout is returned when the crawling failed because of timeout issue
var ErrTimeout = errors.New("timeout has occurred")

// Client is an HTTP client
type Client interface {
	// Get the corresponding URL
	// this methods follows redirections
	Get(URL string) (Response, error)
}

type client struct {
	c *fasthttp.Client
}

// NewFastHTTPClient create a new Client using fasthttp.Client as backend
func NewFastHTTPClient(c *fasthttp.Client) Client {
	return &client{c: c}
}

func (c *client) Get(URL string) (Response, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(URL)

	if err := c.c.Do(req, resp); err != nil {
		// TODO better
		if strings.Contains(err.Error(), "unknown error TTL expired") {
			return nil, ErrTimeout
		}

		return nil, err
	}

	switch code := resp.StatusCode(); {
	case code > 302:
		return nil, fmt.Errorf("non-managed error code %d", code)
	// follow redirect
	case code == 301 || code == 302:
		if location := string(resp.Header.Peek("Location")); location != "" {
			return c.Get(location)
		}
	}

	r := &response{}
	resp.CopyTo(&r.raw)

	return r, nil
}
