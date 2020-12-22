package storage

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type localStorage struct {
	baseDir string
}

func NewLocalStorage(root string) (Storage, error) {
	return &localStorage{baseDir: root}, nil
}

func (s *localStorage) Store(url string, time time.Time, body []byte) error {
	path, err := formatPath(url, time)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.baseDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fullPath, body, 0640); err != nil {
		return err
	}

	return nil
}

func formatPath(rawUrl string, time time.Time) (string, error) {
	b := strings.Builder{}

	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}

	// Protocol
	b.WriteString(u.Scheme)
	b.WriteRune(os.PathSeparator)

	// Hostname
	b.WriteString(u.Host)
	b.WriteRune(os.PathSeparator)

	// Write path
	if uri := u.RequestURI(); uri != "/" {
		b.WriteString(strings.TrimPrefix(u.RequestURI(), "/"))
		b.WriteRune(os.PathSeparator)
	}

	// Write unix time
	b.WriteString(fmt.Sprintf("%d", time.Unix()))

	return b.String(), nil
}
