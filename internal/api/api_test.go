package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/api/database"
	"github.com/creekorful/trandoshan/internal/api/database_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWritePagination(t *testing.T) {
	rec := httptest.NewRecorder()
	searchParams := &api.ResSearchParams{
		PageSize:   15,
		PageNumber: 7,
	}
	total := int64(1200)

	writePagination(rec, searchParams, total)

	if rec.Header().Get(api.PaginationPageHeader) != "7" {
		t.Fail()
	}
	if rec.Header().Get(api.PaginationSizeHeader) != "15" {
		t.Fail()
	}
	if rec.Header().Get(api.PaginationCountHeader) != "1200" {
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

	body := api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.FailNow()
	}

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/resources", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(&searchParamsMatcher{target: api.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}}).Return(int64(0), nil)

	dbMock.EXPECT().AddResource(database.ResourceIdx{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})

	s := State{db: dbMock, refreshDelay: 5 * time.Hour}

	s.addResource(rec, req)
	if rec.Code != http.StatusOK {
		t.FailNow()
	}

	var res api.ResourceDto
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
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

	body := api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.FailNow()
	}

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/resources", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(&searchParamsMatcher{target: api.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}, endDateZero: true}).Return(int64(1), nil)

	s := State{db: dbMock, refreshDelay: -1}

	s.addResource(rec, req)
	if rec.Code != http.StatusOK {
		t.FailNow()
	}
}

func TestAddResourceTooYoung(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	body := api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.FailNow()
	}

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/resources", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(&searchParamsMatcher{target: api.ResSearchParams{
		URL:        "https://example.onion",
		EndDate:    time.Now().Add(-10 * time.Minute),
		PageSize:   1,
		PageNumber: 1,
	}}).Return(int64(1), nil)

	s := State{db: dbMock, refreshDelay: -10 * time.Minute}

	s.addResource(rec, req)
	if rec.Code != http.StatusOK {
		t.FailNow()
	}
}

func TestSearchResources(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/v1/resources?keyword=example", nil)
	rec := httptest.NewRecorder()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(gomock.Any()).Return(int64(150), nil)
	dbMock.EXPECT().SearchResources(gomock.Any()).Return([]database.ResourceIdx{
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

	s := State{db: dbMock}
	s.searchResources(rec, req)

	if rec.Header().Get(api.PaginationCountHeader) != "150" {
		t.Fail()
	}

	var resources []api.ResourceDto
	if err := json.NewDecoder(rec.Body).Decode(&resources); err != nil {
		t.Fatalf("error while decoding body: %s", err)
	}
	if len(resources) != 2 {
		t.Errorf("got %d resources want 2", len(resources))
	}
}

// custom matcher to ignore time field when doing comparison ;(
// todo: do less crappy?
type searchParamsMatcher struct {
	target      api.ResSearchParams
	endDateZero bool
}

func (sm *searchParamsMatcher) Matches(x interface{}) bool {
	arg := x.(*api.ResSearchParams)
	return arg.URL == sm.target.URL && arg.PageSize == sm.target.PageSize && arg.PageNumber == sm.target.PageNumber &&
		sm.endDateZero == arg.EndDate.IsZero()
}

func (sm *searchParamsMatcher) String() string {
	return "is valid search params"
}
