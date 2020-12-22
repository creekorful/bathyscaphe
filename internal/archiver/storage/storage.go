package storage

import "time"

//go:generate mockgen -destination=../storage_mock/storage_mock.go -package=storage_mock . Storage

type Storage interface {
	Store(url string, time time.Time, body []byte) error
}
