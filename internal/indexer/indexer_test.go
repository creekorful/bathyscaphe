package indexer

import (
	"errors"
	"github.com/creekorful/trandoshan/internal/configapi/client"
	"github.com/creekorful/trandoshan/internal/configapi/client_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/creekorful/trandoshan/internal/indexer/index_mock"
	"github.com/creekorful/trandoshan/internal/process"
	"github.com/creekorful/trandoshan/internal/process_mock"
	"github.com/creekorful/trandoshan/internal/test"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestState_Name(t *testing.T) {
	s := State{}
	if s.Name() != "indexer" {
		t.Fail()
	}
}

func TestState_Features(t *testing.T) {
	s := State{}
	test.CheckProcessFeatures(t, &s, []process.Feature{process.EventFeature, process.ConfigFeature})
}

func TestState_CustomFlags(t *testing.T) {
	s := State{}
	test.CheckProcessCustomFlags(t, &s, []string{"index-driver", "index-dest"})
}

func TestState_Initialize(t *testing.T) {
	s := State{}
	test.CheckInitialize(t, &s, func(p *process_mock.MockProviderMockRecorder) {
		p.GetValue("index-driver").Return("local")
		p.GetValue("index-dest")
		p.ConfigClient([]string{client.ForbiddenHostnamesKey})
	})

	if s.indexDriver != "local" {
		t.Errorf("wrong driver: got: %s want: %s", s.indexDriver, "local")
	}
}

func TestState_Subscribers(t *testing.T) {
	s := State{indexDriver: "elastic"}
	test.CheckProcessSubscribers(t, &s, []test.SubscriberDef{
		{Queue: "elasticIndexingQueue", Exchange: "resource.new"},
	})
}

func TestHandleNewResourceEvent(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta (hosted on https://example.org)

<a href="https://google.com/test?test=test#12">

Thanks to https://help.facebook.onion/ for the hosting :D

<meta name="DescriptIon" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)
	indexMock := index_mock.NewMockIndex(mockCtrl)

	tn := time.Now()

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewResourceEvent{}).
		SetArg(1, event.NewResourceEvent{
			URL:     "https://example.onion",
			Body:    body,
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
			Time:    tn,
		}).Return(nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{{Hostname: "example2.onion"}}, nil)
	indexMock.EXPECT().IndexResource("https://example.onion", tn, body, map[string]string{"Server": "Traefik", "Content-Type": "application/html"})

	s := State{index: indexMock, configClient: configClientMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleMessageForbiddenHostname(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta (hosted on https://example.org)

<a href="https://google.com/test?test=test#12">

<meta name="DescriptIon" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)
	configClientMock := client_mock.NewMockClient(mockCtrl)

	tn := time.Now()

	msg := event.RawMessage{}
	subscriberMock.EXPECT().
		Read(&msg, &event.NewResourceEvent{}).
		SetArg(1, event.NewResourceEvent{
			URL:     "https://example.onion",
			Body:    body,
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
			Time:    tn,
		}).Return(nil)

	configClientMock.EXPECT().GetForbiddenHostnames().Return([]client.ForbiddenHostname{{Hostname: "example.onion"}}, nil)

	s := State{configClient: configClientMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); !errors.Is(err, errHostnameNotAllowed) {
		t.FailNow()
	}
}
