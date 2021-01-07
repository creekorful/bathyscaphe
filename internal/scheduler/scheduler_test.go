package scheduler

import (
	"errors"
	"github.com/creekorful/trandoshan/internal/cache"
	"github.com/creekorful/trandoshan/internal/cache_mock"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/creekorful/trandoshan/internal/process_mock"
	"github.com/creekorful/trandoshan/internal/test"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestState_Name(t *testing.T) {
	s := State{}
	if s.Name() != "scheduler" {
		t.Fail()
	}
}

func TestState_Features(t *testing.T) {
	s := State{}
	test.CheckProcessFeatures(t, &s, []process.Feature{process.EventFeature, process.ConfigFeature, process.CacheFeature})
}

func TestState_CustomFlags(t *testing.T) {
	s := State{}
	test.CheckProcessCustomFlags(t, &s, nil)
}

func TestState_Initialize(t *testing.T) {
	test.CheckInitialize(t, &State{}, func(p *process_mock.MockProviderMockRecorder) {
		p.Cache("url")
		p.ConfigClient([]string{client.AllowedMimeTypesKey, client.ForbiddenHostnamesKey, client.RefreshDelayKey})
	})
}

func TestState_Subscribers(t *testing.T) {
	s := State{}
	test.CheckProcessSubscribers(t, &s, []test.SubscriberDef{
		{Queue: "schedulingQueue", Exchange: "resource.new"},
	})
}

func TestNormalizeURL(t *testing.T) {
	url, err := normalizeURL("https://this-is-sparta.de?url=url-query-param#fragment-23")
	if err != nil {
		t.FailNow()
	}

	if url != "https://this-is-sparta.de?url=url-query-param" {
		t.Fail()
	}
}

func TestProcessURL_NotDotOnion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	urls := []string{"https://example.org", "https://pastebin.onionsearchengine.com"}

	for _, url := range urls {
		state := State{}
		if err := state.processURL(url, nil); !errors.Is(err, errNotOnionHostname) {
			t.Fail()
		}
	}
}

func TestProcessURL_ProtocolForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	urls := []string{"ftp://example.onion", "irc://example.onion"}

	for _, url := range urls {
		state := State{}
		if err := state.processURL(url, nil); !errors.Is(err, errProtocolNotAllowed) {
			t.Fail()
		}
	}
}

func TestProcessURL_ExtensionForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configClientMock := client_mock.NewMockClient(mockCtrl)

	urls := []string{"https://example.onion/image.PNG?id=12&test=2", "https://example.onion/favicon.ico"}

	for _, url := range urls {
		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)

		state := State{configClient: configClientMock}
		if err := state.processURL(url, nil); !errors.Is(err, errExtensionNotAllowed) {
			t.Fail()
		}
	}
}

func TestProcessURL_HostnameForbidden(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configClientMock := client_mock.NewMockClient(mockCtrl)

	type testDef struct {
		url                string
		forbiddenHostnames []client.ForbiddenHostname
	}

	tests := []testDef{
		{
			url:                "https://facebookcorewwwi.onion/login.html?id=12&test=2",
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

	for _, tst := range tests {
		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return(tst.forbiddenHostnames, nil)

		state := State{configClient: configClientMock}
		if err := state.processURL(tst.url, nil); !errors.Is(err, errHostnameNotAllowed) {
			t.Fail()
		}
	}
}

func TestProcessURL_AlreadyScheduled(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	urlCacheMock := cache_mock.NewMockCache(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	urlCacheMock.EXPECT().GetInt64("https://facebookcorewwi.onion/test.php?id=12").Return(int64(1), nil)
	configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

	state := State{urlCache: urlCacheMock, configClient: configClientMock}
	if err := state.processURL("https://facebookcorewwi.onion/test.php?id=12", nil); !errors.Is(err, errAlreadyScheduled) {
		t.Fail()
	}
}

func TestHandleNewResourceEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	urlCacheMock := cache_mock.NewMockCache(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	urls := []string{"https://example.onion/index.php", "http://google.onion/admin.secret/login.html",
		"https://example.onion", "https://www.facebookcorewwwi.onion/recover.now/initiate?ars=facebook_login"}

	for _, url := range urls {
		urlCacheMock.EXPECT().GetInt64(url).Return(int64(0), cache.ErrNIL)
		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)
		configClientMock.EXPECT().GetRefreshDelay().Return(client.RefreshDelay{Delay: 10 * time.Hour}, nil)

		urlCacheMock.EXPECT().SetInt64(url, int64(1), time.Duration(10*time.Hour)).Return(nil)
		pubMock.EXPECT().PublishEvent(&event.NewURLEvent{URL: url}).Return(nil)

		state := State{urlCache: urlCacheMock, configClient: configClientMock}
		if err := state.processURL(url, pubMock); err != nil {
			t.Fail()
		}
	}
}
