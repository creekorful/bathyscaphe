package index

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

func TestLocalIndex_IndexResource(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	s := localIndex{baseDir: d}

	ti := time.Date(2020, time.October, 29, 12, 4, 9, 0, time.UTC)
	if err := s.IndexResource(Resource{
		URL:     "https://google.com",
		Time:    ti,
		Body:    "Hello, world",
		Headers: map[string]string{"Server": "Traefik"},
	}); err != nil {
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
	if string(b) != "https://google.com\n\nServer: Traefik\n\nHello, world" {
		t.Fail()
	}
}

func TestLocalIndex_IndexResources(t *testing.T) {
	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.FailNow()
	}
	defer os.RemoveAll(d)

	s := localIndex{baseDir: d}

	ti := time.Date(2020, time.October, 29, 12, 4, 9, 0, time.UTC)

	resources := []Resource{
		{
			URL:     "https://google.com",
			Time:    ti,
			Body:    "Hello, world",
			Headers: map[string]string{"Server": "Traefik"},
		},
	}

	if err := s.IndexResources(resources); err != nil {
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
	if string(b) != "https://google.com\n\nServer: Traefik\n\nHello, world" {
		t.Fail()
	}
}

func TestFormatResource(t *testing.T) {
	res, err := formatResource("https://google.com", "Hello, world", map[string]string{"Server": "Traefik", "Content-Type": "text/html"})
	if err != nil {
		t.FailNow()
	}

	if string(res) != "https://google.com\n\nContent-Type: text/html\nServer: Traefik\n\nHello, world" {
		t.Errorf("got %s want %s", string(res), "https://google.com\n\nServer: Traefik\nContent-Type: text/html\n\nHello, world")
	}
}
