package api

//go:generate mockgen -destination=../api_mock/api_mock.go -package=api_mock . ConfigAPI

// ConfigAPI expose the functionality of the config API
type ConfigAPI interface {
	// Get value of given key
	Get(key string) ([]byte, error)
	// Set value of given key
	Set(key string, value []byte) error
}
