package extractor

import (
	"github.com/creekorful/trandoshan/internal/messaging"
	"testing"
)

func TestExtractResource(t *testing.T) {
	msg := messaging.NewResourceMsg{
		URL:  "https://example.org/300",
		Body: "<title>Creekorful Inc</title>This is sparta<a href\"https://google.com/test?test=test#12\"",
	}

	resDto, urls, err := extractResource(msg)
	if err != nil {
		t.FailNow()
	}

	if resDto.URL != "example.org/300" {
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
}

func TestExtractTitle(t *testing.T) {
	c := "hello this <title>is A</title>TEST"
	if val := extractTitle(c); val != "is A" {
		t.Errorf("Wanted: %s Got: %s", "is A", val)
	}

	c = "hello this is another test"
	if val := extractTitle(c); val != "" {
		t.Errorf("No matches should have been returned")
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
