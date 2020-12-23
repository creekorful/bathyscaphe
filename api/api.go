package api

import (
	"encoding/base64"
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"time"
)

//go:generate mockgen -destination=../api_mock/api_mock.go -package=api_mock . API

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
)

// ResourceDto represent a resource as given by the API
type ResourceDto struct {
	URL         string            `json:"url"`
	Body        string            `json:"body"`
	Time        time.Time         `json:"time"`
	Title       string            `json:"title"`
	Meta        map[string]string `json:"meta"`
	Description string            `json:"description"`
	Headers     map[string]string `json:"headers"`
}

// CredentialsDto represent the credential when logging in the API
type CredentialsDto struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ResSearchParams is the search params used
type ResSearchParams struct {
	URL        string
	Keyword    string
	StartDate  time.Time
	EndDate    time.Time
	WithBody   bool
	PageSize   int
	PageNumber int
	// TODO allow searching by meta
	// TODO allow searching by headers
}

// API is the interface to interact with the API component
type API interface {
	SearchResources(params *ResSearchParams) ([]ResourceDto, int64, error)
	AddResource(res ResourceDto) (ResourceDto, error)
	ScheduleURL(url string) error
}

type client struct {
	httpClient *resty.Client
	baseURL    string
}

func (c *client) SearchResources(params *ResSearchParams) ([]ResourceDto, int64, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources?", c.baseURL)

	req := c.httpClient.R()

	if params.URL != "" {
		b64URL := base64.URLEncoding.EncodeToString([]byte(params.URL))
		req.SetQueryParam("url", b64URL)
	}

	if params.Keyword != "" {
		req.SetQueryParam("keyword", params.Keyword)
	}

	if !params.StartDate.IsZero() {
		req.SetQueryParam("start-date", params.StartDate.Format(time.RFC3339))
	}

	if !params.EndDate.IsZero() {
		req.SetQueryParam("end-date", params.EndDate.Format(time.RFC3339))
	}

	if params.PageNumber != 0 {
		req.Header.Set(PaginationPageHeader, strconv.Itoa(params.PageNumber))
	}
	if params.PageSize != 0 {
		req.Header.Set(PaginationSizeHeader, strconv.Itoa(params.PageSize))
	}

	var resources []ResourceDto
	req.SetResult(&resources)

	res, err := req.Get(targetEndpoint)
	if err != nil {
		return nil, 0, err
	}

	count, err := strconv.ParseInt(res.Header().Get(PaginationCountHeader), 10, 64)
	if err != nil {
		return nil, 0, err
	}

	return resources, count, nil
}

func (c *client) AddResource(res ResourceDto) (ResourceDto, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources", c.baseURL)

	req := c.httpClient.R()
	req.SetBody(res)

	var resourceDto ResourceDto
	req.SetResult(&resourceDto)

	_, err := req.Post(targetEndpoint)
	return resourceDto, err
}

func (c *client) ScheduleURL(url string) error {
	targetEndpoint := fmt.Sprintf("%s/v1/urls", c.baseURL)

	req := c.httpClient.R()
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(fmt.Sprintf("\"%s\"", url))

	_, err := req.Post(targetEndpoint)
	return err
}

// NewClient create a new API client using given details
func NewClient(baseURL, token string) API {
	httpClient := resty.New()
	httpClient.SetAuthScheme("Bearer")
	httpClient.SetAuthToken(token)
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
