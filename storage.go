// storage.go
package main

import "sync"

type ABIStorage struct {
	mu    sync.RWMutex
	cache map[string]string
}

func NewABIStorage() *ABIStorage {
	return &ABIStorage{
		cache: make(map[string]string),
	}
}

func (s *ABIStorage) Set(address, abi string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[address] = abi
}

func (s *ABIStorage) Get(address string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	abi, ok := s.cache[address]
	return abi, ok
}
