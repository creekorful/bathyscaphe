package extractor

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/api_mock"
	"github.com/creekorful/trandoshan/internal/event"
	"github.com/creekorful/trandoshan/internal/event_mock"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

func TestExtractResource(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta

<a href="https://google.com/test?test=test#12">

<meta name="Description" content="Zhello world">
<meta property="og:url" content="https://example.org">
`

	msg := event.NewResourceEvent{
		URL:  "https://example.org/300",
		Body: body,
	}

	resDto, urls, err := extractResource(msg)
	if err != nil {
		t.FailNow()
	}

	if resDto.URL != "https://example.org/300" {
		t.Fail()
	}
	if resDto.Title != "Creekorful Inc" {
		t.Fail()
	}
	if resDto.Body != msg.Body {
		t.Fail()
	}

	if len(urls) != 2 {
		t.FailNow()
	}
	if urls[0] != "https://google.com/test?test=test" {
		t.Fail()
	}
	if urls[1] != "https://example.org" {
		t.Fail()
	}

	if resDto.Description != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["description"] != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["og:url"] != "https://example.org" {
		t.Fail()
	}
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

func TestHandleMessage(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta (hosted on https://example.org)

<a href="https://google.com/test?test=test#12">

<meta name="DescriptIon" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockAPI(mockCtrl)
	subscriberMock := event_mock.NewMockSubscriber(mockCtrl)

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

	// make sure we are creating the resource
	apiClientMock.EXPECT().AddResource(api.ResourceDto{
		URL:         "https://example.onion",
		Body:        body,
		Title:       "Creekorful Inc",
		Meta:        map[string]string{"description": "Zhello world", "og:url": "https://example.org"},
		Description: "Zhello world",
		Headers:     map[string]string{"server": "Traefik", "content-type": "application/html"},
		Time:        tn,
	}).Return(api.ResourceDto{}, nil)

	// make sure we are pushing found URLs

	// should be called only one time
	subscriberMock.EXPECT().
		PublishEvent(&event.FoundURLEvent{URL: "https://example.org"}).
		Return(nil)
	subscriberMock.EXPECT().
		PublishEvent(&event.FoundURLEvent{URL: "https://google.com/test?test=test"}).
		Return(nil)

	s := state{apiClient: apiClientMock}
	if err := s.handleNewResourceEvent(subscriberMock, msg); err != nil {
		t.FailNow()
	}
}
