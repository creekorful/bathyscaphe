package proto

const (
	// URLTodoSubject represent the subject used by the crawler process to read the URL to crawl
	URLTodoSubject = "url.todo"
	// URLDoneSubject represent the subject used by the scheduler process to read the URL to schedule
	URLDoneSubject = "url.done"
)

// URLTodoMessage represent the URL read by the crawler process as input
type URLTodoMessage struct {
	URL string `json:"url"`
}

// URLDoneMessage represent the URL read by the scheduler process as input
type URLDoneMessage struct {
	URL string `json:"url"`
}