package api

import (
	"encoding/base64"
	"fmt"
	"github.com/go-resty/resty/v2"
	"strconv"
	"time"
)

//go:generate mockgen -destination=../api_mock/api_mock.go -package=api_mock . Client

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

	contentTypeJSON     = "application/json"
	authorizationHeader = "Authorization"
)

// ResourceDto represent a resource as given by the API
type ResourceDto struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}

// CredentialsDto represent the credential when logging in the API
type CredentialsDto struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Client is the interface to interact with the API component
type Client interface {
	SearchResources(url, keyword string, startDate, endDate time.Time,
		paginationPage, paginationSize int) ([]ResourceDto, int64, error)
	AddResource(res ResourceDto) (ResourceDto, error)
	ScheduleURL(url string) error
	Authenticate(credentials CredentialsDto) (string, error)
}

type client struct {
	httpClient *resty.Client
	baseURL    string
	token      string
}

func (c *client) SearchResources(url, keyword string,
	startDate, endDate time.Time, paginationPage, paginationSize int) ([]ResourceDto, int64, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/resources?", c.baseURL)

	req := c.httpClient.R()

	if url != "" {
		b64URL := base64.URLEncoding.EncodeToString([]byte(url))
		req.SetQueryParam("url", b64URL)
	}

	if keyword != "" {
		req.SetQueryParam("keyword", keyword)
	}

	if !startDate.IsZero() {
		req.SetQueryParam("start-date", startDate.Format(time.RFC3339))
	}

	if !endDate.IsZero() {
		req.SetQueryParam("end-date", endDate.Format(time.RFC3339))
	}

	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", c.token))

	if paginationPage != 0 {
		req.Header.Set(PaginationPageHeader, strconv.Itoa(paginationPage))
	}
	if paginationSize != 0 {
		req.Header.Set(PaginationSizeHeader, strconv.Itoa(paginationSize))
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

	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", c.token))

	var resourceDto ResourceDto
	req.SetResult(&resourceDto)

	_, err := req.Post(targetEndpoint)
	return resourceDto, err
}

func (c *client) ScheduleURL(url string) error {
	targetEndpoint := fmt.Sprintf("%s/v1/urls", c.baseURL)

	req := c.httpClient.R()
	req.SetBody(url)

	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", c.token))

	_, err := req.Post(targetEndpoint)
	return err
}

func (c *client) Authenticate(credentials CredentialsDto) (string, error) {
	targetEndpoint := fmt.Sprintf("%s/v1/sessions", c.baseURL)

	req := c.httpClient.R()
	req.SetBody(credentials)

	var token string
	req.SetResult(&token)

	_, err := req.Post(targetEndpoint)
	return token, err
}

// NewAuthenticatedClient create a new Client & authenticate it against the API
func NewAuthenticatedClient(baseURL string, credentials CredentialsDto) (Client, error) {
	client := &client{
		httpClient: resty.New(),
		baseURL:    baseURL,
	}

	token, err := client.Authenticate(credentials)
	if err != nil {
		return nil, err
	}
	client.token = token

	return client, nil
}
