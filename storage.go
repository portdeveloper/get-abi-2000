package main

import "sync"

type ABIStorage struct {
	mu    sync.RWMutex
	cache map[string]StorageItem
}

type StorageItem struct {
	ABI            string
	Implementation interface{}
	IsProxy        bool
	IsDecompiled   bool
}

func NewABIStorage() *ABIStorage {
	return &ABIStorage{
		cache: make(map[string]StorageItem),
	}
}

func (s *ABIStorage) Set(key string, item StorageItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = item
}

func (s *ABIStorage) Get(key string) (StorageItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.cache[key]
	return item, ok
}
