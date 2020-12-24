package event

import "time"

//go:generate mockgen -destination=../event_mock/event_mock.go -package=event_mock . Publisher,Subscriber

const (
	// NewURLExchange is the subject used when an URL is schedule for crawling
	NewURLExchange = "url.new"
	// FoundURLExchange is the subject used when an URL is extracted from resource
	FoundURLExchange = "url.found"
	// NewResourceExchange is the subject used when a new resource has been crawled
	NewResourceExchange = "resource.new"
	// ConfigExchange is the exchange used to dispatch new configuration
	ConfigExchange = "config"
)

// Event represent a event
type Event interface {
	// Exchange returns the exchange where event should be push
	Exchange() string
}

// NewURLEvent represent an URL to crawl
type NewURLEvent struct {
	URL string `json:"url"`
}

// Exchange returns the exchange where event should be push
func (msg *NewURLEvent) Exchange() string {
	return NewURLExchange
}

// FoundURLEvent represent a found URL
type FoundURLEvent struct {
	URL string `json:"url"`
}

// Exchange returns the exchange where event should be push
func (msg *FoundURLEvent) Exchange() string {
	return FoundURLExchange
}

// NewResourceEvent represent a crawled resource
type NewResourceEvent struct {
	URL     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Time    time.Time         `json:"time"`
}

// Exchange returns the exchange where event should be push
func (msg *NewResourceEvent) Exchange() string {
	return NewResourceExchange
}
