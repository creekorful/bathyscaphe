package storage

import "time"

//go:generate mockgen -destination=../storage_mock/storage_mock.go -package=storage_mock . Storage

// Storage is a abstraction layer where we store resource
type Storage interface {
	// Store the resource
	Store(url string, time time.Time, body []byte) error
}
