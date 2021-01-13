package crawler

import (
	"errors"
	"github.com/darkspot-org/bathyscaphe/internal/clock_mock"
	"github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/configapi/client_mock"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/event_mock"
	"github.com/darkspot-org/bathyscaphe/internal/http"
	"github.com/darkspot-org/bathyscaphe/internal/http_mock"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/process_mock"
	"github.com/darkspot-org/bathyscaphe/internal/test"
	"github.com/golang/mock/gomock"
	"strings"
	"testing"
	"time"
)

func TestState_Name(t *testing.T) {
	s := State{}
	if s.Name() != "crawler" {
		t.Fail()
	}
}

func TestState_Features(t *testing.T) {
	s := State{}
	test.CheckProcessFeatures(t, &s, []process.Feature{process.EventFeature, process.ConfigFeature, process.CrawlingFeature})
}

func TestState_CustomFlags(t *testing.T) {
	s := State{}
	test.CheckProcessCustomFlags(t, &s, nil)
}

func TestState_Initialize(t *testing.T) {
	test.CheckInitialize(t, &State{}, func(p *process_mock.MockProviderMockRecorder) {
		p.HTTPClient()
		p.Clock()
		p.ConfigClient([]string{client.AllowedMimeTypesKey, client.ForbiddenHostnamesKey})
	})
}

func TestState_Subscribers(t *testing.T) {
	s := State{}
	test.CheckProcessSubscribers(t, &s, []test.SubscriberDef{
		{Queue: "crawlingQueue", Exchange: "url.new"},
	})
}

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
		// The expected error
		err error
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
		},
		{
			url:              "https://example.onion",
			responseHeaders:  map[string]string{"Content-Type": "text/plain"},
			responseBody:     "Hello",
			allowedMimeTypes: []client.MimeType{},
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
			err: errContentTypeNotAllowed,
		},
		{
			url:             "https://downhostname.onion",
			responseHeaders: map[string]string{"Content-Type": "text/plain"},
			responseBody:    "Hello",
			allowedMimeTypes: []client.MimeType{
				{
					ContentType: "text/plain",
					Extensions:  nil,
				},
			},
			err: http.ErrTimeout,
		},
	}

	for _, test := range tests {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.NewURLEvent{}).
			SetArg(1, event.NewURLEvent{URL: test.url}).
			Return(nil)

		// mock crawling
		switch test.err {
		case http.ErrTimeout:
			httpClientMock.EXPECT().Get(test.url).Return(httpResponseMock, http.ErrTimeout)
			subscriberMock.EXPECT().PublishEvent(&event.TimeoutURLEvent{URL: test.url}).Return(nil)
			break
		default:
			httpResponseMock.EXPECT().Headers().Return(test.responseHeaders)
			httpClientMock.EXPECT().Get(test.url).Return(httpResponseMock, nil)

			// mock config retrieval
			configClientMock.EXPECT().GetAllowedMimeTypes().Return(test.allowedMimeTypes, nil)
			break
		}

		configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

		if test.err == nil {
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
		if test.err == nil && err != nil {
			t.Errorf("test should have passed but has failed with: %s", err)
		}
		if !errors.Is(err, test.err) {
			t.Errorf("test shouldn't have passed but hasn't returned expected error: %s", err)
		}
	}
}

func TestHandleNewURLEventHostnameForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	s := State{
		configClient: configClientMock,
	}

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewURLEvent{}).
		SetArg(1, event.NewURLEvent{URL: "https://l.facebookcorewwwi.onion/test.php"}).
		Return(nil)

	configClientMock.EXPECT().GetForbiddenHostnames().
		Return([]client.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)

	if err := s.handleNewURLEvent(subscriberMock, msg); !errors.Is(err, errHostnameNotAllowed) {
		t.Fail()
	}
}
