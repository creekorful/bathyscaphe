package scheduler

import (
	"testing"
	"time"
)

func TestParseRefreshDelay(t *testing.T) {
	if parseRefreshDelay("") != -1 {
		t.Fail()
	}
	if parseRefreshDelay("50s") != time.Second*50 {
		t.Fail()
	}
	if parseRefreshDelay("50m") != time.Minute*50 {
		t.Fail()
	}
	if parseRefreshDelay("50h") != time.Hour*50 {
		t.Fail()
	}
	if parseRefreshDelay("50d") != time.Hour*24*50 {
		t.Fail()
	}
}
