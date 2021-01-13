package index

import (
	"github.com/darkspot-org/bathyscaphe/internal/event"
	"testing"
	"time"
)

func TestIndexResource(t *testing.T) {
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

	resIdx, err := indexResource(Resource{
		URL:     "https://example.org/300",
		Time:    time.Time{},
		Body:    body,
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		t.FailNow()
	}

	if resIdx.URL != "https://example.org/300" {
		t.Fail()
	}
	if resIdx.Title != "Creekorful Inc" {
		t.Fail()
	}
	if resIdx.Body != msg.Body {
		t.Fail()
	}

	if resIdx.Description != "Zhello world" {
		t.Fail()
	}

	if resIdx.Meta["description"] != "Zhello world" {
		t.Fail()
	}

	if resIdx.Meta["og:url"] != "https://example.org" {
		t.Fail()
	}

	if resIdx.Headers["content-type"] != "application/json" {
		t.Fail()
	}
}
