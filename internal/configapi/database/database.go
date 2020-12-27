package database

//go:generate mockgen -destination=../database_mock/database_mock.go -package=database_mock . Database

// Database is the underlying storage for the ConfigAPI
type Database interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}
