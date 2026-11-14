package tokenizer

import (
	"hash/fnv"
	"sync"
)

// TokenCache provides thread-safe LRU-like token count memoization.
type TokenCache struct {
	mu      sync.RWMutex
	maxSize int
	items   map[uint64]int
	keys    []uint64
}

// NewTokenCache constructs a cache with a maximum capacity.
func NewTokenCache(maxSize int) *TokenCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &TokenCache{
		maxSize: maxSize,
		items:   make(map[uint64]int, maxSize),
		keys:    make([]uint64, 0, maxSize),
	}
}

// hashKey computes a 64-bit FNV-1a hash of the input string.
func hashKey(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// Get returns the cached token count and true if present.
func (c *TokenCache) Get(text string) (int, bool) {
	key := hashKey(text)
	c.mu.RLock()
	val, ok := c.items[key]
	c.mu.RUnlock()
	return val, ok
}

// Set stores the token count for a text string, evicting the oldest key if capacity is reached.
func (c *TokenCache) Set(text string, count int) {
	key := hashKey(text)
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[key]; exists {
		c.items[key] = count
		return
	}

	if len(c.keys) >= c.maxSize {
		// Evict oldest item (FIFO eviction for fast O(1) performance)
		oldest := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.items, oldest)
	}

	c.items[key] = count
	c.keys = append(c.keys, key)
}

// Size returns the number of items currently in cache.
func (c *TokenCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear flushes the cache.
func (c *TokenCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[uint64]int, c.maxSize)
	c.keys = c.keys[:0]
}
