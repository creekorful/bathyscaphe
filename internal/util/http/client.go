package http

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

const StatusCreated = 201

const contentTypeJson = "application/json"

type Client struct {
	client http.Client
}

func (c *Client) JsonGet(url string, response interface{}) (*http.Response, error) {
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

func (c *Client) JsonPost(url string, body, response interface{}) (*http.Response, error) {
	logrus.Tracef("POST %s", url)

	var err error
	var b []byte
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	r, err := c.client.Post(url, contentTypeJson, bytes.NewBuffer(b))
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
