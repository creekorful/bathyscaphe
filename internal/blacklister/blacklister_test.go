package blacklister

import (
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/cache_mock"
	configapi "github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestHandleTimeoutURLEventNoDispatch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	hostnameCacheMock := cache_mock.NewMockCache(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion",
		}).Return(nil)

	configClientMock.EXPECT().GetBlackListThreshold().Return(configapi.BlackListThreshold{Threshold: 10}, nil)

	hostnameCacheMock.EXPECT().GetInt64("hostnames:down-example.onion").Return(int64(0), cache.ErrNIL)
	hostnameCacheMock.EXPECT().SetInt64("hostnames:down-example.onion", int64(1), cache.NoTTL).Return(nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}

func TestHandleTimeoutURLEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	hostnameCacheMock := cache_mock.NewMockCache(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion",
		}).Return(nil)

	configClientMock.EXPECT().GetBlackListThreshold().Return(configapi.BlackListThreshold{Threshold: 10}, nil)

	hostnameCacheMock.EXPECT().GetInt64("hostnames:down-example.onion").Return(int64(9), nil)

	configClientMock.EXPECT().
		GetForbiddenHostnames().
		Return([]configapi.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)
	configClientMock.EXPECT().
		Set(configapi.ForbiddenHostnamesKey, []configapi.ForbiddenHostname{
			{Hostname: "facebookcorewwwi.onion"},
			{Hostname: "down-example.onion"},
		}).
		Return(nil)

	hostnameCacheMock.EXPECT().SetInt64("hostnames:down-example.onion", int64(10), cache.NoTTL).Return(nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}

func TestHandleTimeoutURLEventNoDuplicates(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	hostnameCacheMock := cache_mock.NewMockCache(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://facebookcorewwwi.onion",
		}).Return(nil)

	configClientMock.EXPECT().GetBlackListThreshold().Return(configapi.BlackListThreshold{Threshold: 3}, nil)

	hostnameCacheMock.EXPECT().GetInt64("hostnames:facebookcorewwwi.onion").Return(int64(2), nil)

	configClientMock.EXPECT().
		GetForbiddenHostnames().
		Return([]configapi.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)
	// Config not updated since hostname is already 'blacklisted'
	// this may due because of change in threshold

	hostnameCacheMock.EXPECT().SetInt64("hostnames:facebookcorewwwi.onion", int64(3), cache.NoTTL).Return(nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}
