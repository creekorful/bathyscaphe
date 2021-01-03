package scheduler

import (
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleMessageNotOnion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	urls := []string{"https://example.org", "https://pastebin.onionsearchengine.com"}

	for _, url := range urls {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: url}).
			Return(nil)

		s := State{
			configClient: configClientMock,
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errNotOnionHostname) {
			t.FailNow()
		}
	}
}

func TestHandleMessageWrongProtocol(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := event.RawMessage{}

	s := State{
		configClient: configClientMock,
	}

	for _, protocol := range []string{"irc", "ftp"} {
		subscriberMock.EXPECT().
			Read(&msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: fmt.Sprintf("%s://example.onion", protocol)}).
			Return(nil)

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errProtocolNotAllowed) {
			t.FailNow()
		}
	}
}

func TestHandleMessageForbiddenExtensions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	urls := []string{"https://example.onion/image.PNG?id=12&test=2", "https://example.onion/favicon.ico"}

	for _, url := range urls {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: url}).
			Return(nil)

		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"php", "html"}}}, nil)

		s := State{
			configClient: configClientMock,
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errExtensionNotAllowed) {
			t.FailNow()
		}
	}
}

func TestHandleMessageHostnameForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	type test struct {
		url                string
		forbiddenHostnames []client.ForbiddenHostname
	}

	tests := []test{
		{
			url:                "https://facebookcorewwwi.onion/image.png?id=12&test=2",
			forbiddenHostnames: []client.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}},
		},
		{
			url:                "https://google.onion:9099",
			forbiddenHostnames: []client.ForbiddenHostname{{Hostname: "google.onion"}},
		},
		{
			url:                "http://facebook.onion:443/news/test.php?id=12&username=test",
			forbiddenHostnames: []client.ForbiddenHostname{{Hostname: "facebook.onion"}},
		},
		{
			url:                "https://www.facebookcorewwwi.onion/recover/initiate?ars=facebook_login",
			forbiddenHostnames: []client.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}},
		},
	}

	for _, test := range tests {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: test.url}).
			Return(nil)

		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"png", "php"}}}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return(test.forbiddenHostnames, nil)

		s := State{
			configClient: configClientMock,
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errHostnameNotAllowed) {
			t.Errorf("%s has not returned errHostnameNotAllowed", test.url)
		}
	}
}

func TestHandleMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	urls := []string{"https://example.onion/index.php", "http://google.onion/admin.secret/login.html",
		"https://example.onion", "https://www.facebookcorewwwi.onion/recover.now/initiate?ars=facebook_login"}

	for _, u := range urls {
		msg := event.RawMessage{}
		subscriberMock.EXPECT().
			Read(&msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: u}).
			Return(nil)

		subscriberMock.EXPECT().
			PublishEvent(&event.NewURLEvent{URL: u}).
			Return(nil)

		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

		s := State{
			configClient: configClientMock,
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); err != nil {
			t.FailNow()
		}
	}
}

func TestScheduleURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// The requests
	req := httptest.NewRequest(http.MethodPost, "/urls", strings.NewReader("\"https://google.onion\""))
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
