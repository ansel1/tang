package cache

import (
	"sync"
	"time"
)

// Cache is a simple in-memory key-value store with TTL support.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]entry
	ttl     time.Duration
}

type entry struct {
	value     string
	expiresAt time.Time
}

// New creates a new Cache with the given TTL.
func New(ttl time.Duration) *Cache {
	return &Cache{
		items: make(map[string]entry),
		ttl:   ttl,
	}
}

// Set stores a value with the configured TTL.
func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Get retrieves a value. Returns ("", false) if not found or expired.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

// Len returns the number of items (including expired ones).
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Purge removes all expired entries and returns how many were removed.
func (c *Cache) Purge() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	now := time.Now()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
			count++
		}
	}
	return count
}
