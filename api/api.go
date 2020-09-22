package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

const contentTypeJSON = "application/json"

// ResourceDto represent a resource as given by the API
type ResourceDto struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

// Client is the interface to interact with the API process
type Client interface {
	SearchResources(url string) ([]ResourceDto, error)
	AddResource(res ResourceDto) (ResourceDto, error)
	ScheduleURL(url string) error
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func (c *client) SearchResources(url string) ([]ResourceDto, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources", c.baseURL)

	if url != "" {
		targetEndpoint += "?url=" + url
	}

	var resources []ResourceDto
	_, err := jsonGet(c.httpClient, targetEndpoint, &resources)
	return resources, err
}

func (c *client) AddResource(res ResourceDto) (ResourceDto, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources", c.baseURL)

	var resourceDto ResourceDto
	_, err := jsonPost(c.httpClient, targetEndpoint, res, &resourceDto)
	return resourceDto, err
}

func (c *client) ScheduleURL(url string) error {
	targetEndpoint := fmt.Sprintf("%s/v1/urls", c.baseURL)
	_, err := jsonPost(c.httpClient, targetEndpoint, url, nil)
	return err
}

// NewClient create a new Client instance to dial with the API located on given address
func NewClient(baseURL string) Client {
	return &client{
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		baseURL: baseURL,
	}
}

func jsonGet(httpClient *http.Client, url string, response interface{}) (*http.Response, error) {
	log.Trace().Str("verb", "GET").Str("url", url).Msg("")

	r, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}

	return r, nil
}

func jsonPost(httpClient *http.Client, url string, request, response interface{}) (*http.Response, error) {
	log.Trace().Str("verb", "POST").Str("url", url).Msg("")

	var err error
	var b []byte
	if request != nil {
		b, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
	}

	r, err := httpClient.Post(url, contentTypeJSON, bytes.NewBuffer(b))
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
