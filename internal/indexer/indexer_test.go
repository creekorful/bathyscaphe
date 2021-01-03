package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/cache_mock"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	client2 "github.com/creekorful/trandoshan/internal/indexer/client"
	"github.com/creekorful/trandoshan/internal/indexer/index"
	"github.com/creekorful/trandoshan/internal/indexer/index_mock"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWritePagination(t *testing.T) {
	rec := httptest.NewRecorder()
	searchParams := &client2.ResSearchParams{
		PageSize:   15,
		PageNumber: 7,
	}
	total := int64(1200)

	writePagination(rec, searchParams, total)

	if rec.Header().Get(client2.PaginationPageHeader) != "7" {
		t.Fail()
	}
	if rec.Header().Get(client2.PaginationSizeHeader) != "15" {
		t.Fail()
	}
	if rec.Header().Get(client2.PaginationCountHeader) != "1200" {
		t.Fail()
	}
}

func TestReadPagination(t *testing.T) {
	// valid params
	req := httptest.NewRequest(http.MethodGet, "/index.php?pagination-page=1&pagination-size=10", nil)
	if page, size := getPagination(req); page != 1 || size != 10 {
		t.Errorf("wanted page: 1, size: 10 (got %d, %d)", page, size)
	}

	// make sure invalid parameter are set as wanted
	req = httptest.NewRequest(http.MethodGet, "/index.php?pagination-page=abcd&pagination-size=lol", nil)
	if page, size := getPagination(req); page != 1 || size != defaultPaginationSize {
		t.Errorf("wanted page: 1, size: %d (got %d, %d)", defaultPaginationSize, page, size)
	}

	// make sure we prevent too much results from being returned
	target := fmt.Sprintf("/index.php?pagination-page=10&pagination-size=%d", maxPaginationSize+1)
	req = httptest.NewRequest(http.MethodGet, target, nil)
	if page, size := getPagination(req); page != 10 || size != maxPaginationSize {
		t.Errorf("wanted page: 10, size: %d (got %d, %d)", maxPaginationSize, page, size)
	}

	// make sure no parameter we set to default
	req = httptest.NewRequest(http.MethodGet, "/index.php", nil)
	if page, size := getPagination(req); page != 1 || size != defaultPaginationSize {
		t.Errorf("wanted page: 1, size: %d (got %d, %d)", defaultPaginationSize, page, size)
	}
}

func TestGetSearchParams(t *testing.T) {
	startDate := time.Now()
	target := fmt.Sprintf("/resources?with-body=true&pagination-page=1&keyword=keyword&url=dXJs&start-date=%s", startDate.Format(time.RFC3339))

	req := httptest.NewRequest(http.MethodPost, target, nil)

	params, err := getSearchParams(req)
	if err != nil {
		t.Errorf("error while parsing search params: %s", err)
		t.FailNow()
	}

	if !params.WithBody {
		t.Errorf("wrong withBody: %v", params.WithBody)
	}
	if params.PageSize != 50 {
		t.Errorf("wrong pagination-size: %d", params.PageSize)
	}
	if params.PageNumber != 1 {
		t.Errorf("wrong pagination-page: %d", params.PageNumber)
	}
	if params.Keyword != "keyword" {
		t.Errorf("wrong keyword: %s", params.Keyword)
	}
	if params.StartDate.Year() != startDate.Year() {
		t.Errorf("wrong start-date (year)")
	}
	if params.StartDate.Month() != startDate.Month() {
		t.Errorf("wrong start-date (month)")
	}
	if params.StartDate.Day() != startDate.Day() {
		t.Errorf("wrong start-date (day)")
	}
	if params.StartDate.Hour() != startDate.Hour() {
		t.Errorf("wrong start-date (hour)")
	}
	if params.StartDate.Minute() != startDate.Minute() {
		t.Errorf("wrong start-date (minute)")
	}
	if params.StartDate.Second() != startDate.Second() {
		t.Errorf("wrong start-date (second)")
	}
	if params.URL != "url" {
		t.Errorf("wrong url: %s", params.URL)
	}
}

func TestScheduleURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/urls", strings.NewReader("\"https://google.onion\""))
	rec := httptest.NewRecorder()

	// Mocking status
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	s := State{pub: pubMock}

	pubMock.EXPECT().PublishEvent(&event.FoundURLEvent{URL: "https://google.onion"}).Return(nil)

	s.scheduleURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fail()
	}
}

func TestAddResource(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := client2.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}

	indexMock := index_mock.NewMockIndex(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	indexMock.EXPECT().CountResources(&searchParamsMatcher{target: client2.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}}).Return(int64(0), nil)

	indexMock.EXPECT().AddResource(index.ResourceIdx{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})

	pubMock.EXPECT().PublishEvent(&event.NewIndexEvent{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

	s := State{index: indexMock, configClient: configClientMock, pub: pubMock}
	res, err := s.tryAddResource(body, 5*time.Hour)
	if err != nil {
		t.FailNow()
	}

	if res.URL != "https://example.onion" {
		t.FailNow()
	}
	if res.Body != "TheBody" {
		t.FailNow()
	}
	if res.Title != "Example" {
		t.FailNow()
	}
	if !res.Time.IsZero() {
		t.FailNow()
	}
	if res.Meta["content"] != "content-meta" {
		t.FailNow()
	}
	if res.Description != "the description" {
		t.FailNow()
	}
	if res.Headers["Content-Type"] != "application/html" {
		t.FailNow()
	}
	if res.Headers["Server"] != "Traefik" {
		t.FailNow()
	}
}

func TestAddResourceDuplicateNotAllowed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := client2.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}

	indexMock := index_mock.NewMockIndex(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	indexMock.EXPECT().CountResources(&searchParamsMatcher{target: client2.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}, endDateZero: true}).Return(int64(1), nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

	s := State{index: indexMock, configClient: configClientMock}

	if _, err := s.tryAddResource(body, -1); !errors.Is(err, errAlreadyIndexed) {
		t.FailNow()
	}
}

func TestAddResourceTooYoung(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := client2.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}

	indexMock := index_mock.NewMockIndex(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	indexMock.EXPECT().CountResources(&searchParamsMatcher{target: client2.ResSearchParams{
		URL:        "https://example.onion",
		EndDate:    time.Now().Add(-10 * time.Minute),
		PageSize:   1,
		PageNumber: 1,
	}}).Return(int64(1), nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

	s := State{index: indexMock, configClient: configClientMock}

	if _, err := s.tryAddResource(body, 10*time.Minute); !errors.Is(err, errAlreadyIndexed) {
		t.FailNow()
	}
}

func TestAddResourceForbiddenHostname(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := client2.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}

	configClientMock := client_mock.NewMockClient(mockCtrl)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{{Hostname: "example.onion"}}, nil)

	s := State{configClient: configClientMock}

	if _, err := s.tryAddResource(body, -1); err != errHostnameNotAllowed {
		t.FailNow()
	}
}

func TestSearchResources(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/resources?keyword=example", nil)
	rec := httptest.NewRecorder()

	indexMock := index_mock.NewMockIndex(mockCtrl)

	indexMock.EXPECT().CountResources(gomock.Any()).Return(int64(150), nil)
	indexMock.EXPECT().SearchResources(gomock.Any()).Return([]index.ResourceIdx{
		{
			URL:   "example-1.onion",
			Body:  "Example 1",
			Title: "Example 1",
			Time:  time.Time{},
		},
		{
			URL:   "example-2.onion",
			Body:  "Example 2",
			Title: "Example 2",
			Time:  time.Time{},
		},
	}, nil)

	s := State{index: indexMock}
	s.searchResources(rec, req)

	if rec.Code != http.StatusOK {
		t.Fail()
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fail()
	}
	if rec.Header().Get(client2.PaginationCountHeader) != "150" {
		t.Fail()
	}

	var resources []client2.ResourceDto
	if err := json.NewDecoder(rec.Body).Decode(&resources); err != nil {
		t.Fatalf("error while decoding body: %s", err)
	}
	if len(resources) != 2 {
		t.Errorf("got %d resources want 2", len(resources))
	}
}

func TestExtractResource(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta

<a href="https://google.com/test?test=test#12">

<meta name="Description" content="Zhello world">
<meta property="og:url" content="https://example.org">
`

	msg := event.NewResourceEvent{
		URL:  "https://example.org/300",
		Body: body,
	}

	resDto, urls, err := extractResource(msg)
	if err != nil {
		t.FailNow()
	}

	if resDto.URL != "https://example.org/300" {
		t.Fail()
	}
	if resDto.Title != "Creekorful Inc" {
		t.Fail()
	}
	if resDto.Body != msg.Body {
		t.Fail()
	}

	if len(urls) != 2 {
		t.FailNow()
	}
	if urls[0] != "https://google.com/test?test=test" {
		t.Fail()
	}
	if urls[1] != "https://example.org" {
		t.Fail()
	}

	if resDto.Description != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["description"] != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["og:url"] != "https://example.org" {
		t.Fail()
	}
}

func TestNormalizeURL(t *testing.T) {
	url, err := normalizeURL("https://this-is-sparta.de?url=url-query-param#fragment-23")
	if err != nil {
		t.FailNow()
	}

	if url != "https://this-is-sparta.de?url=url-query-param" {
		t.Fail()
	}
}

func TestHandleNewResourceEvent(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta (hosted on https://example.org)

<a href="https://google.com/test?test=test#12">

Thanks to https://help.facebook.onion/ for the hosting :D

<meta name="DescriptIon" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	indexMock := index_mock.NewMockIndex(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)
	urlCacheMock := cache_mock.NewMockCache(mockCtrl)

	tn := time.Now()

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewResourceEvent{}).
		SetArg(1, event.NewResourceEvent{
			URL:     "https://example.onion",
			Body:    body,
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
			Time:    tn,
		}).Return(nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{{Hostname: "example2.onion"}}, nil)
	configClientMock.EXPECT().GetRefreshDelay().Times(1).Return(client.RefreshDelay{Delay: -1}, nil)

	indexMock.EXPECT().CountResources(&client2.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}).Return(int64(0), nil)

	// make sure we are creating the resource
	indexMock.EXPECT().AddResource(index.ResourceIdx{
		URL:         "https://example.onion",
		Body:        body,
		Title:       "Creekorful Inc",
		Meta:        map[string]string{"description": "Zhello world", "og:url": "https://example.org"},
		Description: "Zhello world",
		Headers:     map[string]string{"server": "Traefik", "content-type": "application/html"},
		Time:        tn,
	}).Return(nil)

	pubMock.EXPECT().PublishEvent(&event.NewIndexEvent{
		URL:         "https://example.onion",
		Body:        body,
		Title:       "Creekorful Inc",
		Meta:        map[string]string{"description": "Zhello world", "og:url": "https://example.org"},
		Description: "Zhello world",
		Headers:     map[string]string{"server": "Traefik", "content-type": "application/html"},
		Time:        tn,
	}).Return(nil)

	// make sure we are pushing found URLs (but only if refresh delay elapsed)
	urlCacheMock.EXPECT().GetInt64("urls:https://example.org").Return(int64(0), cache.ErrNIL)
	indexMock.EXPECT().CountResources(&client2.ResSearchParams{
		URL:        "https://example.org",
		PageSize:   1,
		PageNumber: 1,
	}).Return(int64(0), nil)

	urlCacheMock.EXPECT().GetInt64("urls:https://example.org").Return(int64(1), nil)

	urlCacheMock.EXPECT().GetInt64("urls:https://help.facebook.onion").Return(int64(1), nil)

	urlCacheMock.EXPECT().GetInt64("urls:https://google.com/test?test=test").Return(int64(0), cache.ErrNIL)
	indexMock.EXPECT().CountResources(&client2.ResSearchParams{
		URL:        "https://google.com/test?test=test",
		PageSize:   1,
		PageNumber: 1,
	}).Return(int64(1), nil)

	subscriberMock.EXPECT().
		PublishEvent(&event.FoundURLEvent{URL: "https://example.org"}).
		Return(nil)
	urlCacheMock.EXPECT().SetInt64("urls:https://example.org", int64(1), cache.NoTTL).Return(nil)

	s := State{index: indexMock, configClient: configClientMock, pub: pubMock, urlCache: urlCacheMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleMessageForbiddenHostname(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta (hosted on https://example.org)

<a href="https://google.com/test?test=test#12">

<meta name="DescriptIon" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	tn := time.Now()

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewResourceEvent{}).
		SetArg(1, event.NewResourceEvent{
			URL:     "https://example.onion",
			Body:    body,
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
			Time:    tn,
		}).Return(nil)

	configClientMock.EXPECT().GetRefreshDelay().Return(client.RefreshDelay{Delay: -1}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{{Hostname: "example.onion"}}, nil)

	s := State{configClient: configClientMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != errHostnameNotAllowed {
		t.FailNow()
	}
}

// custom matcher to ignore time field when doing comparison ;(
// todo: do less crappy?
type searchParamsMatcher struct {
	target      client2.ResSearchParams
	endDateZero bool
}

func (sm *searchParamsMatcher) Matches(x interface{}) bool {
	arg := x.(*client2.ResSearchParams)
	return arg.URL == sm.target.URL && arg.PageSize == sm.target.PageSize && arg.PageNumber == sm.target.PageNumber &&
		sm.endDateZero == arg.EndDate.IsZero()
}

func (sm *searchParamsMatcher) String() string {
	return "is valid search params"
}
