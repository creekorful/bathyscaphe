package scheduler

import (
	"bytes"
	"fmt"
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/api_mock"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/messaging_mock"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestHandleMessageNotOnion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		ReadMsg(msg, &messaging.URLFoundMsg{}).
		SetArg(1, messaging.URLFoundMsg{URL: "https://example.org"}).
		Return(nil)

	if err := handleMessage(apiClientMock, -1, []string{})(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleMessageWrongProtocol(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)

	for _, protocol := range []string{"irc", "ftp"} {
		subscriberMock.EXPECT().
			ReadMsg(msg, &messaging.URLFoundMsg{}).
			SetArg(1, messaging.URLFoundMsg{URL: fmt.Sprintf("%s://example.onion", protocol)}).
			Return(nil)

		if err := handleMessage(apiClientMock, -1, []string{})(subscriberMock, msg); err != nil {
			t.FailNow()
		}
	}
}

func TestHandleMessageAlreadyCrawled(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		ReadMsg(msg, &messaging.URLFoundMsg{}).
		SetArg(1, messaging.URLFoundMsg{URL: "https://example.onion"}).
		Return(nil)

	apiClientMock.EXPECT().
		SearchResources("https://example.onion", "", time.Time{}, time.Time{}, 1, 1).
		Return([]api.ResourceDto{}, int64(1), nil)

	if err := handleMessage(apiClientMock, -1, []string{"png"})(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleMessageForbiddenExtensions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		ReadMsg(msg, &messaging.URLFoundMsg{}).
		SetArg(1, messaging.URLFoundMsg{URL: "https://example.onion/image.png?id=12&test=2"}).
		Return(nil)

	if err := handleMessage(apiClientMock, -1, []string{"png"})(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := bytes.NewReader(nil)
	subscriberMock.EXPECT().
		ReadMsg(msg, &messaging.URLFoundMsg{}).
		SetArg(1, messaging.URLFoundMsg{URL: "https://example.onion"}).
		Return(nil)

	apiClientMock.EXPECT().
		SearchResources("https://example.onion", "", time.Time{}, time.Time{}, 1, 1).
		Return([]api.ResourceDto{}, int64(0), nil)

	subscriberMock.EXPECT().
		PublishMsg(&messaging.URLTodoMsg{URL: "https://example.onion"}).
		Return(nil)

	if err := handleMessage(apiClientMock, -1, []string{})(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}
