package database

import "sync"

// MemoryDatabase is an memory only database structure
type MemoryDatabase struct {
	values map[string][]byte
	mutex  sync.Mutex
}

// Get value using his key
func (md *MemoryDatabase) Get(key string) ([]byte, error) {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	return md.values[key], nil
}

// Set value for given key
func (md *MemoryDatabase) Set(key string, value []byte) error {
	md.mutex.Lock()
	defer md.mutex.Unlock()

	md.values[key] = value
	return nil
}
