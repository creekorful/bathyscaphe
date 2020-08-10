package http

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

// StatusCreated HTTP 201
const StatusCreated = 201

const contentTypeJSON = "application/json"

// Client an http client with built-in JSON (de)serialization
type Client struct {
	client http.Client
}

// JSONGet perform a GET request and serialize response body into given interface if any
func (c *Client) JSONGet(url string, response interface{}) (*http.Response, error) {
	logrus.Tracef("GET %s", url)

	r, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}

	return r, nil
}

// JSONPost perform a POST request and serialize request/response body into given interface if any
func (c *Client) JSONPost(url string, request, response interface{}) (*http.Response, error) {
	logrus.Tracef("POST %s", url)

	var err error
	var b []byte
	if request != nil {
		b, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}

	r, err := c.client.Post(url, contentTypeJSON, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	if response != nil {
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			return nil, err
		}
	}

	return r, nil
}
