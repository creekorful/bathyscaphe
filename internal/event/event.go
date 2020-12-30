package event

import "time"

//go:generate mockgen -destination=../event_mock/event_mock.go -package=event_mock . Publisher,Subscriber

const (
	// NewURLExchange is the exchange used when an URL is schedule for crawling
	NewURLExchange = "url.new"
	// FoundURLExchange is the exchange used when an URL is extracted from resource
	FoundURLExchange = "url.found"
	// TimeoutURLExchange is the exchange used when a crawling fail because of timeout
	TimeoutURLExchange = "url.timeout"
	// NewResourceExchange is the exchange used when a new resource has been crawled
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

// TimeoutURLEvent represent a failed crawling because of timeout
type TimeoutURLEvent struct {
	URL string `json:"url"`
}

// Exchange returns the exchange where event should be push
func (msg *TimeoutURLEvent) Exchange() string {
	return TimeoutURLExchange
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
