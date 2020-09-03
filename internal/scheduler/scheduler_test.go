package scheduler

import "testing"

func TestNormalizeURL(t *testing.T) {
	url, err := normalizeURL("https://this-is-sparta.de?url=url-query-param#fragment-23")
	if err != nil {
		t.FailNow()
	}

	if url.String() != "https://this-is-sparta.de?url=url-query-param" {
		t.Fail()
	}
}
