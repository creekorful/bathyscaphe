package extractor

import (
	"github.com/creekorful/trandoshan/api"
	"github.com/creekorful/trandoshan/api_mock"
	"github.com/creekorful/trandoshan/internal/messaging"
	"github.com/creekorful/trandoshan/internal/messaging_mock"
	"github.com/golang/mock/gomock"
	"github.com/nats-io/nats.go"
	"testing"
)

func TestExtractResource(t *testing.T) {
	body := `
<title>Creekorful Inc</title>

This is sparta

<a href="https://google.com/test?test=test#12">

<meta name="description" content="Zhello world">
<meta property="og:url" content="https://example.org">
`

	msg := messaging.NewResourceMsg{
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

	if len(urls) == 0 {
		t.FailNow()
	}
	if urls[0] != "https://google.com/test?test=test" {
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

This is sparta

<a href="https://google.com/test?test=test#12">

<meta name="description" content="Zhello world">
<meta property="og:url" content="https://example.org">`

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	apiClientMock := api_mock.NewMockClient(mockCtrl)
	subscriberMock := messaging_mock.NewMockSubscriber(mockCtrl)

	msg := nats.Msg{}
	subscriberMock.EXPECT().
		ReadMsg(&msg, &messaging.NewResourceMsg{}).
		SetArg(1, messaging.NewResourceMsg{URL: "https://example.onion", Body: body}).
		Return(nil)

	// make sure we are creating the resource
	apiClientMock.EXPECT().AddResource(&resMatcher{target: api.ResourceDto{
		URL:         "https://example.onion",
		Body:        body,
		Title:       "Creekorful Inc",
		Meta:        map[string]string{"description": "Zhello world", "og:url": "https://example.org"},
		Description: "Zhello world",
	}}).Return(api.ResourceDto{}, nil)

	// make sure we are pushing found URLs
	subscriberMock.EXPECT().
		PublishMsg(&messaging.URLFoundMsg{URL: "https://example.org"}).
		Return(nil)
	subscriberMock.EXPECT().
		PublishMsg(&messaging.URLFoundMsg{URL: "https://google.com/test?test=test"}).
		Return(nil)

	if err := handleMessage(apiClientMock)(subscriberMock, &msg); err != nil {
		t.FailNow()
	}
}

// custom matcher to ignore time field when doing comparison
// todo: do less crappy?
type resMatcher struct {
	target api.ResourceDto
}

func (rm *resMatcher) Matches(x interface{}) bool {
	arg := x.(api.ResourceDto)
	return arg.Title ==
		rm.target.Title &&
		arg.URL == rm.target.URL &&
		arg.Body == rm.target.Body &&
		arg.Description == rm.target.Description &&
		exactMatch(arg.Meta, rm.target.Meta)
}

func (rm *resMatcher) String() string {
	return "is valid resource"
}

func exactMatch(left, right map[string]string) bool {
	for key, want := range left {
		if got, exist := right[key]; !exist || got != want {
			return false
		}
	}

	return true
}
