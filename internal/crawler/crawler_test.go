package crawler

import (
	"errors"
	"github.com/creekorful/trandoshan/internal/clock_mock"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/crawler/http_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"strings"
	"testing"
	"time"
)

func TestHandleNewURLEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)
	clockMock := clock_mock.NewMockClock(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	s := State{
		httpClient:   httpClientMock,
		configClient: configClientMock,
		clock:        clockMock,
	}

	type test struct {
		// the incoming url
		url string
		// the response headers
		responseHeaders map[string]string
		// the response body
		responseBody string
		// internal state: allowed mime types
		allowedMimeTypes []client.MimeType
		// is the test expected to pass?
		pass bool
	}

	tests := []test{
		{
			url:             "https://example.onion/image.png?id=12&test=2",
			responseHeaders: map[string]string{"Content-Type": "text/plain", "Server": "Debian"},
			responseBody:    "Hello",
			allowedMimeTypes: []client.MimeType{
				{ContentType: "text/plain", Extensions: nil},
				{ContentType: "text/css", Extensions: nil},
			},
			pass: true,
		},
		{
			url:              "https://example.onion",
			responseHeaders:  map[string]string{"Content-Type": "text/plain"},
			responseBody:     "Hello",
			allowedMimeTypes: []client.MimeType{},
			pass:             true,
		},
		{
			url:             "https://example.onion",
			responseHeaders: map[string]string{"Content-Type": "text/plain"},
			responseBody:    "Hello",
			allowedMimeTypes: []client.MimeType{
				{
					ContentType: "text/",
					Extensions:  nil,
				},
			},
			pass: true,
		},
		{
			url:             "https://example.onion/image.png",
			responseHeaders: map[string]string{"Content-Type": "image/png"},
			responseBody:    "Hello",
			allowedMimeTypes: []client.MimeType{
				{
					ContentType: "text/plain",
					Extensions:  nil,
				},
			},
			pass: false,
		},
	}

	for _, test := range tests {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.NewURLEvent{}).
			SetArg(1, event.NewURLEvent{URL: test.url}).
			Return(nil)

		// mock crawling
		httpResponseMock.EXPECT().Headers().Return(test.responseHeaders)
		httpClientMock.EXPECT().Get(test.url).Return(httpResponseMock, nil)

		// mock config retrieval
		configClientMock.EXPECT().GetAllowedMimeTypes().Return(test.allowedMimeTypes, nil)

		if test.pass {
			httpResponseMock.EXPECT().Headers().Return(test.responseHeaders)
			httpResponseMock.EXPECT().Body().Return(strings.NewReader(test.responseBody))

			tn := time.Now()
			clockMock.EXPECT().Now().Return(tn)

			// if test should pass expect event publishing
			subscriberMock.EXPECT().PublishEvent(&event.NewResourceEvent{
				URL:     test.url,
				Body:    test.responseBody,
				Headers: test.responseHeaders,
				Time:    tn,
			}).Return(nil)
		}

		err := s.handleNewURLEvent(subscriberMock, msg)
		if test.pass && err != nil {
			t.Errorf("test should have passed but has failed with: %s", err)
		}
		if !test.pass && !errors.Is(err, errContentTypeNotAllowed) {
			t.Errorf("test shouldn't have passed but hasn't returned expected error: %s", err)
		}
	}
}
