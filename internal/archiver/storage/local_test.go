package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatPath(t *testing.T) {
	type test struct {
		url  string
		time time.Time
		path string
	}

	ti := time.Date(2020, time.October, 29, 12, 4, 9, 0, time.UTC)

	tests := []test{
		{
			url:  "https://google.com",
			time: ti,
			path: "https/google.com/1603973049",
		},
		{
			url:  "http://facebook.com/admin/login.php?username=admin",
			time: ti,
			path: "http/facebook.com/16609974401560122507/1603973049",
		},
		{
			url:  "http://thisisalonghostname.onion/admin/tools/list-accounts.php?token=123223453&username=test",
			time: ti,
			path: "http/thisisalonghostname.onion/7883137132857825203/1603973049",
		},
	}

	for _, test := range tests {
		res, err := formatPath(test.url, test.time)
		if err != nil {
			t.Error()
		}

		if res != test.path {
			t.Errorf("got: %s, want: %s", res, test.path)
		}
	}
}

func TestLocalStorage_Store(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	s := localStorage{baseDir: d}

	ti := time.Date(2020, time.October, 29, 12, 4, 9, 0, time.UTC)

	if err := s.Store("https://google.com", ti, []byte("Hello, world")); err != nil {
		t.Fail()
	}

	p := filepath.Join(d, "https", "google.com", "1603973049")

	inf, err := os.Stat(p)
	if err != nil {
		t.Fail()
	}
	if inf.Mode() != 0640 {
		t.Fail()
	}

	b, err := ioutil.ReadFile(p)
	if err != nil {
		t.Fail()
	}
	if string(b) != "Hello, world" {
		t.Fail()
	}
}
