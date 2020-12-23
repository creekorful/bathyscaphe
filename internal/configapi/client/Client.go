package client

//go:generate mockgen -destination=../client_mock/client_mock.go -package=client_mock . Client

const (
	forbiddenMimeTypes = "forbidden-mime-types"
	forbiddenHostnames = "forbidden-hostnames"
)

// ForbiddenMimeType is the mime types who's crawling is forbidden
type ForbiddenMimeType struct {
	// The content-type
	ContentType string `json:"content-type"`
	// The list of associated extensions
	Extensions []string `json:"extensions"`
}

// ForbiddenHostname is the hostnames who's crawling is forbidden
type ForbiddenHostname struct {
	Hostname string `json:"hostname"`
}

// Client is a nice client interface for the ConfigAPI
type Client interface {
	GetForbiddenMimeTypes() ([]ForbiddenMimeType, error)
	SetForbiddenMimeTypes(values []ForbiddenMimeType) error

	GetForbiddenHostnames() ([]ForbiddenHostname, error)
	SetForbiddenHostnames(values []ForbiddenHostname) error
}
