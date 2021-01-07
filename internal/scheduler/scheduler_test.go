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
		if err := state.processURL(url, nil, nil); !errors.Is(err, errNotOnionHostname) {
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
		if err := state.processURL(url, nil, nil); !errors.Is(err, errProtocolNotAllowed) {
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
		if err := state.processURL(url, nil, nil); !errors.Is(err, errExtensionNotAllowed) {
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
		if err := state.processURL(tst.url, nil, nil); !errors.Is(err, errHostnameNotAllowed) {
			t.Fail()
		}
	}
}

func TestProcessURL_AlreadyScheduled(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configClientMock := client_mock.NewMockClient(mockCtrl)

	configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

	urlCache := map[string]int64{"https://facebookcorewwi.onion/test.php?id=12": 1}
	state := State{configClient: configClientMock}
	if err := state.processURL("https://facebookcorewwi.onion/test.php?id=12", nil, urlCache); !errors.Is(err, errAlreadyScheduled) {
		t.Fail()
	}
}

func TestProcessURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	configClientMock := client_mock.NewMockClient(mockCtrl)
	pubMock := event_mock.NewMockPublisher(mockCtrl)

	urls := []string{"https://example.onion/index.php", "http://google.onion/admin.secret/login.html",
		"https://example.onion", "https://www.facebookcorewwwi.onion/recover.now/initiate?ars=facebook_login"}

	// pre fill cache
	urlCache := map[string]int64{}
	for _, url := range urls {
		configClientMock.EXPECT().GetAllowedMimeTypes().Return([]client.MimeType{{Extensions: []string{"html", "php"}}}, nil)
		configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{}, nil)

		pubMock.EXPECT().PublishEvent(&event.NewURLEvent{URL: url}).Return(nil)

		state := State{configClient: configClientMock}
		if err := state.processURL(url, pubMock, urlCache); err != nil {
			t.Fail()
		}

		if val, exist := urlCache[url]; !exist || val != 1 {
			t.Fail()
		}
	}
}

func TestHandleNewResourceEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	urlCacheMock := cache_mock.NewMockCache(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewResourceEvent{}).
		SetArg(1, event.NewResourceEvent{
			URL: "https://l.facebookcorewwwi.onion/test.php",
			Body: `
<a href=\"https://facebook.onion/test.php?id=1\">This is a little test</a>. 
Check out https://google.onion. This is an image https://example.onion/test.png
This domain is blacklisted: https://m.fbi.onion/test.php
`,
		}).
		Return(nil)

	urlCacheMock.EXPECT().
		GetManyInt64([]string{"https://facebook.onion/test.php?id=1", "https://google.onion", "https://example.onion/test.png", "https://m.fbi.onion/test.php"}).
		Return(map[string]int64{
			"https://google.onion": 1,
		}, nil)

	configClientMock.EXPECT().GetAllowedMimeTypes().
		Times(4).
		Return([]client.MimeType{{Extensions: []string{"php"}}}, nil)
	configClientMock.EXPECT().GetForbiddenHostnames().
		Times(3).
		Return([]client.ForbiddenHostname{
			{Hostname: "fbi.onion"},
		}, nil)
	configClientMock.EXPECT().GetRefreshDelay().Return(client.RefreshDelay{Delay: -1}, nil)

	subscriberMock.EXPECT().PublishEvent(&event.NewURLEvent{
		URL: "https://facebook.onion/test.php?id=1",
	})

	urlCacheMock.EXPECT().SetManyInt64(map[string]int64{
		"https://google.onion":                 1,
		"https://facebook.onion/test.php?id=1": 1,
	}, cache.NoTTL).Return(nil)

	s := State{urlCache: urlCacheMock, configClient: configClientMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.Fail()
	}
}
