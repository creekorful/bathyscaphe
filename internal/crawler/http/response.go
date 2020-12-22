package http

//go:generate mockgen -destination=../http_mock/response_mock.go -package=http_mock . Response

import (
	"bytes"
	"github.com/valyala/fasthttp"
	"io"
)

// Response is an HTTP response
type Response interface {
	// Headers returns the response headers
	Headers() map[string]string
	// Body return the response body
	Body() io.Reader
}

type response struct {
	raw fasthttp.Response
}

func (r *response) Headers() map[string]string {
	headers := map[string]string{}
	r.raw.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value) // TODO manage multiple values?
	})
	return headers
}

func (r *response) Body() io.Reader {
	return bytes.NewReader(r.raw.Body())
}
