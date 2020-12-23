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

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://example.org"}).
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

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		Read(msg, &event.FoundURLEvent{}).
		SetArg(1, event.FoundURLEvent{URL: "https://example.onion/image.png?id=12&test=2"}).
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

func TestHandleMessage(t *testing.T) {
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
		Return([]api.ResourceDto{}, int64(0), nil)

	subscriberMock.EXPECT().
		Publish(&event.NewURLEvent{URL: "https://example.onion"}).
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
