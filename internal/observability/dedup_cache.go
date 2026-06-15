package observability

import (
	"sync"
	"time"
)

type EventDeduplicator interface {
	SeenOrAdd(key string) bool
}

type InMemoryEventDedupCache struct {
	mu         sync.Mutex
	items      map[string]time.Time
	ttl        time.Duration
	writeCount int
}

func NewInMemoryEventDedupCache(ttl time.Duration) *InMemoryEventDedupCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &InMemoryEventDedupCache{
		items: make(map[string]time.Time),
		ttl:   ttl,
	}
}

func (c *InMemoryEventDedupCache) SeenOrAdd(key string) bool {
	if key == "" {
		return false
	}

	now := time.Now()
	expiresAt := now.Add(c.ttl)

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.items[key]; ok {
		if existing.After(now) {
			return true
		}
	}

	c.items[key] = expiresAt
	c.writeCount++

	if c.writeCount%200 == 0 {
		c.evictExpiredLocked(now)
	}

	return false
}

func (c *InMemoryEventDedupCache) evictExpiredLocked(now time.Time) {
	for k, exp := range c.items {
		if !exp.After(now) {
			delete(c.items, k)
		}
	}
}
