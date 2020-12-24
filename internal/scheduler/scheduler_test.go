package scheduler

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/api_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestHandleMessageNotOnion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	urls := []string{"https://example.org", "https://pastebin.onionsearchengine.com"}

	for _, url := range urls {
		msg := bytes.NewReader(nil)
		subscriberMock.EXPECT().
			Read(msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: url}).
			Return(nil)

		s := state{
			apiClient:           apiClientMock,
			refreshDelay:        -1,
			forbiddenExtensions: []string{},
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errNotOnionHostname) {
			t.FailNow()
		}
	}
}

func TestHandleMessageWrongProtocol(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)

	s := state{
		apiClient:           apiClientMock,
		refreshDelay:        -1,
		forbiddenExtensions: []string{},
	}

	for _, protocol := range []string{"irc", "ftp"} {
		subscriberMock.EXPECT().
			Read(msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: fmt.Sprintf("%s://example.onion", protocol)}).
			Return(nil)

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errProtocolNotAllowed) {
			t.FailNow()
		}
	}
}

func TestHandleMessageAlreadyCrawled(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://example.onion"}).
		Return(nil)

	params := api.ResSearchParams{
		URL:        "https://example.onion",
		PageSize:   1,
		PageNumber: 1,
	}
	apiClientMock.EXPECT().
		SearchResources(&params).
		Return([]api.ResourceDto{}, int64(1), nil)

	s := state{
		apiClient:           apiClientMock,
		refreshDelay:        -1,
		forbiddenExtensions: []string{"png"},
	}

	if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errShouldNotSchedule) {
		t.FailNow()
	}
}

func TestHandleMessageForbiddenExtensions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	urls := []string{"https://example.onion/image.png?id=12&test=2", "https://example.onion/image.PNG"}

	for _, url := range urls {
		msg := bytes.NewReader(nil)
		subscriberMock.EXPECT().
			Read(msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: url}).
			Return(nil)

		s := state{
			apiClient:           apiClientMock,
			refreshDelay:        -1,
			forbiddenExtensions: []string{"png"},
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errExtensionNotAllowed) {
			t.FailNow()
		}
	}
}

func TestHandleMessageHostnameForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	type test struct {
		url                string
		forbiddenHostnames []string
	}

	tests := []test{
		{
			url:                "https://facebookcorewwwi.onion/image.png?id=12&test=2",
			forbiddenHostnames: []string{"facebookcorewwwi.onion"},
		},
		{
			url:                "https://google.onion:9099",
			forbiddenHostnames: []string{"google.onion"},
		},
		{
			url:                "http://facebook.onion:443/news/test.php?id=12&username=test",
			forbiddenHostnames: []string{"facebook.onion"},
		},
		{
			url:                "https://www.facebookcorewwwi.onion/recover/initiate?ars=facebook_login",
			forbiddenHostnames: []string{"facebookcorewwwi.onion"},
		},
	}

	for _, test := range tests {
		msg := bytes.NewReader(nil)
		subscriberMock.EXPECT().
			Read(msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: test.url}).
			Return(nil)

		s := state{
			apiClient:           apiClientMock,
			refreshDelay:        -1,
			forbiddenExtensions: []string{},
			forbiddenHostnames:  test.forbiddenHostnames,
		}

		if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errHostnameNotAllowed) {
			t.Errorf("%s has not returned errHostnameNotAllowed", test.url)
		}
	}
}

func TestHandleMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://www.facebookcorewwwi.onion/recover/initiate?ars=facebook_login"}).
		Return(nil)

	params := api.ResSearchParams{
		URL:        "https://www.facebookcorewwwi.onion/recover/initiate?ars=facebook_login",
		PageSize:   1,
		PageNumber: 1,
	}
	apiClientMock.EXPECT().
		SearchResources(&params).
		Return([]api.ResourceDto{}, int64(0), nil)

	subscriberMock.EXPECT().
		Publish(&event.NewURLEvent{URL: "https://www.facebookcorewwwi.onion/recover/initiate?ars=facebook_login"}).
		Return(nil)

	s := state{
		apiClient:           apiClientMock,
		refreshDelay:        -1,
		forbiddenExtensions: []string{},
	}

	if err := s.handleURLFoundEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}
