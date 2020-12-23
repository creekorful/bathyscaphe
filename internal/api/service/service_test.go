package service

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/internal/api/database"
	"github.com/creekorful/trandoshan/internal/api/database_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestSearchResources(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := &api.ResSearchParams{Keyword: "example"}

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(params).Return(int64(150), nil)
	dbMock.EXPECT().SearchResources(params).Return([]database.ResourceIdx{
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

	s := Service{db: dbMock}

	res, count, err := s.SearchResources(params)
	if err != nil {
		t.FailNow()
	}
	if count != 150 {
		t.Error()
	}
	if len(res) != 2 {
		t.Error()
	}
}

func TestAddResource(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

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

	s := Service{db: dbMock, refreshDelay: 5 * time.Hour}

	res, err := s.AddResource(api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})
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

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(&searchParamsMatcher{target: api.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}, endDateZero: true}).Return(int64(1), nil)

	s := Service{db: dbMock, refreshDelay: -1}

	_, err := s.AddResource(api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})
	if err != nil {
		t.FailNow()
	}
}

func TestAddResourceTooYoung(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbMock := database_mock.NewMockDatabase(mockCtrl)

	dbMock.EXPECT().CountResources(&searchParamsMatcher{target: api.ResSearchParams{
		URL:        "https://example.onion",
		EndDate:    time.Now().Add(-10 * time.Minute),
		PageSize:   1,
		PageNumber: 1,
	}}).Return(int64(1), nil)

	s := Service{db: dbMock, refreshDelay: -10 * time.Minute}

	_, err := s.AddResource(api.ResourceDto{
		URL:         "https://example.onion",
		Body:        "TheBody",
		Title:       "Example",
		Time:        time.Time{},
		Meta:        map[string]string{"content": "content-meta"},
		Description: "the description",
		Headers:     map[string]string{"Content-Type": "application/html", "Server": "Traefik"},
	})
	if err != nil {
		t.FailNow()
	}
}

func TestScheduleURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	pubMock := event_mock.NewMockPublisher(mockCtrl)

	s := Service{pub: pubMock}

	pubMock.EXPECT().Publish(&event.FoundURLEvent{URL: "https://example.onion"})

	if err := s.ScheduleURL("https://example.onion"); err != nil {
		t.FailNow()
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
