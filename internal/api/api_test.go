package api

import "testing"

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
