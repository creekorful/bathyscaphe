package duration

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	if ParseDuration("") != -1 {
		t.Fail()
	}
	if ParseDuration("50s") != time.Second*50 {
		t.Fail()
	}
	if ParseDuration("50m") != time.Minute*50 {
		t.Fail()
	}
	if ParseDuration("50h") != time.Hour*50 {
		t.Fail()
	}
	if ParseDuration("50d") != time.Hour*24*50 {
		t.Fail()
	}
}
