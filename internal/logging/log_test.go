package logging

import (
	"testing"
)

func TestGetLogFlag(t *testing.T) {
	flag := GetLogFlag()
	if flag.Name != "log-level" {
		t.Fail()
	}
	if flag.Usage != "Set the application log level" {
		t.Fail()
	}
	if flag.Value != "info" {
		t.Fail()
	}
}
