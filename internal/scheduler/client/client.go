package client

import (
	"fmt"
	"github.com/go-resty/resty/v2"
)

// Client is the interface to interact with the scheduler API
type Client interface {
	ScheduleURL(url string) error
}

type client struct {
	httpClient *resty.Client
	baseURL    string
}

func (c *client) ScheduleURL(url string) error {
	targetEndpoint := fmt.Sprintf("%s/urls", c.baseURL)

	req := c.httpClient.R()
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(fmt.Sprintf("\"%s\"", url))

	_, err := req.Post(targetEndpoint)
	return err
}

// NewClient create a new API client using given details
func NewClient(baseURL string) Client {
	httpClient := resty.New()
	httpClient.OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
		if r.StatusCode() > 302 {
			return fmt.Errorf("error when making HTTP request: %s", r.Status())
		}
		return nil
	})

	client := &client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}

	return client
}
