package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"time"
)

const (
	// PaginationPageHeader is the header to determinate current page in paginated endpoint
	PaginationPageHeader = "X-Pagination-Page"
	// PaginationSizeHeader is the header to determinate page size in paginated endpoint
	PaginationSizeHeader = "X-Pagination-Size"
	// PaginationCountHeader is the header to determinate total count of element in paginated endpoint
	PaginationCountHeader = "X-Pagination-Count"
	// PaginationPageQueryParam is the query parameter used to set current page in paginated endpoint
	PaginationPageQueryParam = "pagination-page"
	// PaginationSizeQueryParam is the query parameter used to set page size in paginated endpoint
	PaginationSizeQueryParam = "pagination-size"

	contentTypeJSON = "application/json"
)

// ResourceDto represent a resource as given by the API
type ResourceDto struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

// Client is the interface to interact with the API process
type Client interface {
	SearchResources(url, keyword string, startDate, endDate time.Time,
		paginationPage, paginationSize int) ([]ResourceDto, int64, error)
	AddResource(res ResourceDto) (ResourceDto, error)
	ScheduleURL(url string) error
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func (c *client) SearchResources(url, keyword string,
	startDate, endDate time.Time, paginationPage, paginationSize int) ([]ResourceDto, int64, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources?", c.baseURL)

	if url != "" {
		targetEndpoint += fmt.Sprintf("url=%s&", url)
	}

	if keyword != "" {
		targetEndpoint += fmt.Sprintf("keyword=%s&", keyword)
	}

	if !startDate.IsZero() {
		targetEndpoint += fmt.Sprintf("start-date=%s&", startDate.Format(time.RFC3339))
	}

	if !endDate.IsZero() {
		targetEndpoint += fmt.Sprintf("end-date=%s&", endDate.Format(time.RFC3339))
	}

	headers := map[string]string{}

	if paginationPage != 0 {
		headers[PaginationPageHeader] = strconv.Itoa(paginationPage)
	}
	if paginationSize != 0 {
		headers[PaginationSizeHeader] = strconv.Itoa(paginationSize)
	}

	var resources []ResourceDto
	res, err := jsonGet(c.httpClient, targetEndpoint, headers, &resources)
	if err != nil {
		return nil, 0, err
	}

	count, err := strconv.ParseInt(res.Header[PaginationCountHeader][0], 10, 64)
	if err != nil {
		return nil, 0, err
	}

	return resources, count, nil
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

func jsonGet(httpClient *http.Client, url string, headers map[string]string, response interface{}) (*http.Response, error) {
	log.Trace().Str("verb", "GET").Str("url", url).Msg("")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// populate custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	r, err := httpClient.Do(req)
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
