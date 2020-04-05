package proto

const (
	URLTodoSubject = "url.todo"
	URLDoneSubject = "url.done"
)

type URLTodoMessage struct {
	Url string `json:"url"`
}

type URLDoneMessage struct {
	Url string `json:"url"`
}