package indexer

import (
	"errors"
	"github.com/darkspot-org/bathyscaphe/internal/configapi/client"
	"github.com/darkspot-org/bathyscaphe/internal/configapi/client_mock"
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"github.com/darkspot-org/bathyscaphe/internal/event_mock"
	"github.com/darkspot-org/bathyscaphe/internal/indexer/index"
	"github.com/darkspot-org/bathyscaphe/internal/indexer/index_mock"
	"github.com/darkspot-org/bathyscaphe/internal/process"
	"github.com/darkspot-org/bathyscaphe/internal/process_mock"
	"github.com/darkspot-org/bathyscaphe/internal/test"
	"github.com/golang/mock/gomock"
	"reflect"
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
		p.GetStrValue("index-driver").Return("local")
		p.GetStrValue("index-dest")
		p.GetIntValue(process.EventPrefetchFlag).Return(10)
		p.ConfigClient([]string{client.ForbiddenHostnamesKey})
	})

	if s.indexDriver != "local" {
		t.Errorf("wrong driver: got: %s want: %s", s.indexDriver, "local")
	}
	if s.bufferThreshold != 10 {
		t.Errorf("wrong buffer threshold: got: %d want: %d", s.bufferThreshold, 10)
	}
}

func TestState_Subscribers(t *testing.T) {
	s := State{indexDriver: "elastic"}
	test.CheckProcessSubscribers(t, &s, []test.SubscriberDef{
		{Queue: "elasticIndexingQueue", Exchange: "resource.new"},
	})
}

func TestHandleNewResourceEvent_NoBuffering(t *testing.T) {
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
	indexMock.EXPECT().IndexResource(index.Resource{
		URL:     "https://example.onion",
		Time:    tn,
		Body:    body,
		Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
	})

	s := State{index: indexMock, configClient: configClientMock, bufferThreshold: 1}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}

func TestHandleNewResourceEvent_Buffering_NoDispatch(t *testing.T) {
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

	s := State{index: indexMock, configClient: configClientMock, bufferThreshold: 5}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}

	if len(s.resources) != 1 {
		t.FailNow()
	}

	if s.resources[0].URL != "https://example.onion" {
		t.Fail()
	}
	if s.resources[0].Body != body {
		t.Fail()
	}
	if !reflect.DeepEqual(s.resources[0].Headers, map[string]string{"Server": "Traefik", "Content-Type": "application/html"}) {
		t.Fail()
	}
	if s.resources[0].Time != tn {
		t.Fail()
	}
}

func TestHandleNewResourceEvent_Buffering_Dispatch(t *testing.T) {
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
	indexMock.EXPECT().IndexResources([]index.Resource{
		{
			URL: "https://google.onion",
		},
		{
			URL:     "https://example.onion",
			Time:    tn,
			Body:    body,
			Headers: map[string]string{"Server": "Traefik", "Content-Type": "application/html"},
		},
	})

	s := State{
		index:           indexMock,
		configClient:    configClientMock,
		bufferThreshold: 2,
		resources:       []index.Resource{{URL: "https://google.onion"}},
	}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}

	// should be reset
	if len(s.resources) != 0 {
		t.Fail()
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
