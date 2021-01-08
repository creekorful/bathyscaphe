package index

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type localIndex struct {
	baseDir string
}

func newLocalIndex(root string) (Index, error) {
	return &localIndex{baseDir: root}, nil
}

func (s *localIndex) IndexResource(resource Resource) error {
	path, err := formatPath(resource.URL, resource.Time)
	if err != nil {
		return err
	}

	content, err := formatResource(resource.URL, resource.Body, resource.Headers)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.baseDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	if err := ioutil.WriteFile(fullPath, content, 0640); err != nil {
		return err
	}

	return nil
}

func (s *localIndex) IndexResources(resources []Resource) error {
	// No specific implementation for the local driver.
	// we simply call IndexResource n-times
	for _, resource := range resources {
		if err := s.IndexResource(resource); err != nil {
			return err
		}
	}

	return nil
}

func formatResource(url string, body string, headers map[string]string) ([]byte, error) {
	builder := strings.Builder{}

	// First URL
	builder.WriteString(fmt.Sprintf("%s\n\n", url))

	// Sort headers to have deterministic output
	var headerNames []string
	for headerName := range headers {
		headerNames = append(headerNames, headerName)
	}
	sort.Strings(headerNames)

	// Then headers
	for _, name := range headerNames {
		builder.WriteString(fmt.Sprintf("%s: %s\n", name, headers[name]))
	}
	builder.WriteString("\n")

	// Then body
	builder.WriteString(body)

	return []byte(builder.String()), nil
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
