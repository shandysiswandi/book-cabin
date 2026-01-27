package cache

import (
	"sync"
	"time"
)

type entry[T any] struct {
	value  T
	expiry time.Time
}

type Cache[T any] struct {
	mu      sync.RWMutex
	entries map[string]entry[T]
	clone   func(T) T
}

func New[T any](clone func(T) T) *Cache[T] {
	return &Cache[T]{
		entries: make(map[string]entry[T]),
		clone:   clone,
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		var zero T
		return zero, false
	}
	if time.Now().After(entry.expiry) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		var zero T
		return zero, false
	}
	if c.clone != nil {
		return c.clone(entry.value), true
	}
	return entry.value, true
}

func (c *Cache[T]) Set(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	c.entries[key] = entry[T]{value: c.cloneValue(value), expiry: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *Cache[T]) cloneValue(value T) T {
	if c.clone == nil {
		return value
	}
	return c.clone(value)
}
