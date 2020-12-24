package crawler

import (
	"github.com/creekorful/trandoshan/internal/clock_mock"
	"github.com/creekorful/trandoshan/internal/crawler/http_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"strings"
	"testing"
	"time"
)

func TestCrawlURLForbiddenContentType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{"text/plain"}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Return(map[string]string{"Content-Type": "image/png"})

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if body != "" || headers != nil || err == nil {
		t.Fail()
	}
}

func TestCrawlURLSameContentType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{"text/plain"}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if err != nil {
		t.Fail()
	}
	if body != "Hello" {
		t.Fail()
	}
	if len(headers) != 1 {
		t.Fail()
	}
	if headers["Content-Type"] != "text/plain" {
		t.Fail()
	}
}

func TestCrawlURLNoContentTypeFiltering(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	httpClientMock := http_mock.NewMockClient(mockCtrl)
	url := "https://example.onion"
	allowedContentTypes := []string{""}

	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get(url).Return(httpResponseMock, nil)

	body, headers, err := crawURL(httpClientMock, url, allowedContentTypes)
	if err != nil {
		t.Fail()
	}
	if body != "Hello" {
		t.Fail()
	}
	if len(headers) != 1 {
		t.Fail()
	}
	if headers["Content-Type"] != "text/plain" {
		t.Fail()
	}
}

func TestHandleNewURLEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	clockMock := clock_mock.NewMockClock(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewURLEvent{}).
		SetArg(1, event.NewURLEvent{URL: "https://example.onion/image.png?id=12&test=2"}).
		Return(nil)

	httpResponseMock.EXPECT().Headers().Times(2).Return(map[string]string{"Content-Type": "text/plain", "Server": "Debian"})
	httpResponseMock.EXPECT().Body().Return(strings.NewReader("Hello"))

	httpClientMock.EXPECT().Get("https://example.onion/image.png?id=12&test=2").Return(httpResponseMock, nil)

	tn := time.Now()
	clockMock.EXPECT().Now().Return(tn)

	subscriberMock.EXPECT().PublishEvent(&event.NewResourceEvent{
		URL:     "https://example.onion/image.png?id=12&test=2",
		Body:    "Hello",
		Headers: map[string]string{"Content-Type": "text/plain", "Server": "Debian"},
		Time:    tn,
	}).Return(nil)

	s := state{
		httpClient:          httpClientMock,
		allowedContentTypes: []string{"text/plain", "text/css"},
		clock:               clockMock,
	}
	if err := s.handleNewURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}
