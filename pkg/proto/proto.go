package proto

import "time"

const (
	// URLTodoSubject represent the subject used by the crawler process to read the URL to crawl
	URLTodoSubject = "url.todo"
	// URLFoundSubject represent the subject used by the scheduler process to read the URL to schedule
	URLFoundSubject = "url.found"
	// ResourceSubject represent the subject used by the persister process to store the resource body
	ResourceSubject = "resource"
)

// URLTodoMsg represent an URL to crawl
type URLTodoMsg struct {
	URL string `json:"url"`
}

// URLFoundMsg represent a found URL
type URLFoundMsg struct {
	URL string `json:"url"`
}

// ResourceMsg represent the body of a crawled resource
type ResourceMsg struct {
	URL  string `json:"url"`
	Body string `json:"body"`
}

// ResourceDto represent a resource as given by the API
type ResourceDto struct {
	URL   string    `json:"url"`
	Body  string    `json:"body"`
	Title string    `json:"title"`
	Time  time.Time `json:"time"`
}
