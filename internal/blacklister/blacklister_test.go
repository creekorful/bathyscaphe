package blacklister

import (
	"errors"
	"github.com/darkspot-org/bathyscaphe/internal/cache_mock"
	configapi "github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/configapi/client_mock"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/event_mock"
	"github.com/darkspot-org/bathyscaphe/internal/http"
	"github.com/darkspot-org/bathyscaphe/internal/http_mock"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/process_mock"
	"github.com/darkspot-org/bathyscaphe/internal/test"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestState_Name(t *testing.T) {
	s := State{}
	if s.Name() != "blacklister" {
		t.Fail()
	}
}

func TestState_Features(t *testing.T) {
	s := State{}
	test.CheckProcessFeatures(t, &s, []process.Feature{process.EventFeature, process.ConfigFeature, process.CacheFeature, process.CrawlingFeature})
}

func TestState_CustomFlags(t *testing.T) {
	s := State{}
	test.CheckProcessCustomFlags(t, &s, nil)
}

func TestState_Initialize(t *testing.T) {
	test.CheckInitialize(t, &State{}, func(p *process_mock.MockProviderMockRecorder) {
		p.Cache("down-hostname")
		p.ConfigClient([]string{configapi.ForbiddenHostnamesKey, configapi.BlackListConfigKey})
		p.HTTPClient()
	})
}

func TestState_Subscribers(t *testing.T) {
	s := State{}
	test.CheckProcessSubscribers(t, &s, []test.SubscriberDef{
		{Queue: "blacklistingQueue", Exchange: "url.timeout"},
	})
}

func TestHandleTimeoutURLEventNoTimeout(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	hostnameCacheMock := cache_mock.NewMockCache(mockCtrl)
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion:8080/reset-password?username=test",
		}).Return(nil)

	httpClientMock.EXPECT().Get("https://down-example.onion:8080").Return(httpResponseMock, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]configapi.ForbiddenHostname{}, nil)

	hostnameCacheMock.EXPECT().Remove("down-example.onion")

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock, httpClient: httpClientMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}

func TestHandleTimeoutURLEventNoDispatch(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	hostnameCacheMock := cache_mock.NewMockCache(mockCtrl)
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion/login.php",
		}).Return(nil)

	httpClientMock.EXPECT().Get("https://down-example.onion").Return(httpResponseMock, http.ErrTimeout)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]configapi.ForbiddenHostname{}, nil)
	configClientMock.EXPECT().GetBlackListConfig().Return(configapi.BlackListConfig{
		Threshold: 10,
		TTL:       5,
	}, nil)

	hostnameCacheMock.EXPECT().GetInt64("down-example.onion").Return(int64(0), nil)
	hostnameCacheMock.EXPECT().SetInt64("down-example.onion", int64(1), time.Duration(5)).Return(nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock, httpClient: httpClientMock}
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
	httpClientMock := http_mock.NewMockClient(mockCtrl)
	httpResponseMock := http_mock.NewMockResponse(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.TimeoutURLEvent{}).
		SetArg(1, event.TimeoutURLEvent{
			URL: "https://down-example.onion/test.html",
		}).Return(nil)

	httpClientMock.EXPECT().Get("https://down-example.onion").Return(httpResponseMock, http.ErrTimeout)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]configapi.ForbiddenHostname{}, nil)
	configClientMock.EXPECT().GetBlackListConfig().Return(configapi.BlackListConfig{
		Threshold: 10,
		TTL:       5,
	}, nil)

	hostnameCacheMock.EXPECT().GetInt64("down-example.onion").Return(int64(9), nil)

	configClientMock.EXPECT().
		GetForbiddenHostnames().
		Return([]configapi.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)
	configClientMock.EXPECT().
		Set(configapi.ForbiddenHostnamesKey, []configapi.ForbiddenHostname{
			{Hostname: "facebookcorewwwi.onion"},
			{Hostname: "down-example.onion"},
		}).
		Return(nil)

	hostnameCacheMock.EXPECT().
		SetInt64("down-example.onion", int64(10), time.Duration(5)).
		Return(nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock, httpClient: httpClientMock}
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
			URL: "https://facebookcorewwwi.onion/morning-routine.php?id=12",
		}).Return(nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]configapi.ForbiddenHostname{{Hostname: "facebookcorewwwi.onion"}}, nil)

	s := State{configClient: configClientMock, hostnameCache: hostnameCacheMock}
	if err := s.handleTimeoutURLEvent(subscriberMock, msg); !errors.Is(err, errAlreadyBlacklisted) {
		t.Fail()
	}
}
