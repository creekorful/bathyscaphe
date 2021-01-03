package storage

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type localStorage struct {
	baseDir string
}

// NewLocalStorage returns a new Storage that use local file system
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

func formatPath(rawURL string, time time.Time) (string, error) {
	b := strings.Builder{}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Protocol
	b.WriteString(u.Scheme)
	b.WriteRune(os.PathSeparator)

	// Hostname
	b.WriteString(u.Host)
	b.WriteRune(os.PathSeparator)

	if uri := u.RequestURI(); uri != "/" {
		// Write path (hash it to prevent too long filename)
		c := fnv.New64()
		if _, err := c.Write([]byte(strings.TrimPrefix(u.RequestURI(), "/"))); err != nil {
			return "", err
		}

		b.WriteString(strconv.FormatUint(c.Sum64(), 10))
		b.WriteRune(os.PathSeparator)
	}

	// Write unix time
	b.WriteString(fmt.Sprintf("%d", time.Unix()))

	return b.String(), nil
}
