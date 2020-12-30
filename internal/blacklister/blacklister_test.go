package blacklister

import (
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestHandleTimeoutURLEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion",
		}).Return(nil)

	configClientMock.EXPECT().
		GetForbiddenHostnames().
		Return([]configapi.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)
	configClientMock.EXPECT().
		Set(configapi.ForbiddenHostnamesKey, []configapi.ForbiddenHostname{
			{Hostname: "facebookcorewwwi.onion"},
			{Hostname: "down-example.onion"},
		}).
		Return(nil)

	s := State{configClient: configClientMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}
