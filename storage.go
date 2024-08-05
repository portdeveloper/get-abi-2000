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

func (s *ABIStorage) Set(key string, abi string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = abi
}

func (s *ABIStorage) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	abi, ok := s.cache[key]
	return abi, ok
}
