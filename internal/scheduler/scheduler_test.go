package scheduler

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/api_mock"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
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
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://example.org"}).
		Return(nil)

	s := state{
		apiClient:    apiClientMock,
		configClient: configClientMock,
	}

	if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errNotOnionHostname) {
		t.FailNow()
	}
}

func TestHandleMessageWrongProtocol(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := bytes.NewReader(nil)

	s := state{
		apiClient:    apiClientMock,
		configClient: configClientMock,
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
	configClientMock := client_mock.NewMockClient(mockCtrl)

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

	configClientMock.EXPECT().GetForbiddenMimeTypes().Return([]client.ForbiddenMimeType{{Extensions: []string{"png"}}}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)
	configClientMock.EXPECT().GetRefreshDelay().Return(client.RefreshDelay{Delay: -1}, nil)

	s := state{
		apiClient:    apiClientMock,
		configClient: configClientMock,
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
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://example.onion/image.png?id=12&test=2"}).
		Return(nil)

	configClientMock.EXPECT().GetForbiddenMimeTypes().Return([]client.ForbiddenMimeType{{Extensions: []string{"png"}}}, nil)

	s := state{
		apiClient:    apiClientMock,
		configClient: configClientMock,
	}

	if err := s.handleURLFoundEvent(subscriberMock, msg); !errors.Is(err, errExtensionNotAllowed) {
		t.FailNow()
	}
}

func TestHandleMessageHostnameForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
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
		msg := bytes.NewReader(nil)
		subscriberMock.EXPECT().
			Read(msg, &event.FoundURLEvent{}).
			SetArg(1, event.FoundURLEvent{URL: test.url}).
			Return(nil)

		configClientMock.EXPECT().GetForbiddenMimeTypes().Return([]client.ForbiddenMimeType{}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return(test.forbiddenHostnames, nil)

		s := state{
			apiClient:    apiClientMock,
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

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

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
		Return([]api.ResourceDto{}, int64(0), nil)

	subscriberMock.EXPECT().
		PublishEvent(&event.NewURLEvent{URL: "https://example.onion"}).
		Return(nil)

	configClientMock.EXPECT().GetForbiddenMimeTypes().Return([]client.ForbiddenMimeType{}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)
	configClientMock.EXPECT().GetRefreshDelay().Return(client.RefreshDelay{Delay: -1}, nil)

	s := state{
		apiClient:    apiClientMock,
		configClient: configClientMock,
	}

	if err := s.handleURLFoundEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}
