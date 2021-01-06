package index

import (
	"github.com/creekorful/trandoshan/internal/event"
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

	resDto, err := extractResource("https://example.org/300", time.Time{}, body, map[string]string{"Content-Type": "application/json"})
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

	if resDto.Description != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["description"] != "Zhello world" {
		t.Fail()
	}

	if resDto.Meta["og:url"] != "https://example.org" {
		t.Fail()
	}

	if resDto.Headers["content-type"] != "application/json" {
		t.Fail()
	}
}
